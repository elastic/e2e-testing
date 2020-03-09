#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  /*
  * this node will be reused across the nested stages because it's needed to
  * reuse the Docker cache
  */
  agent { label 'linux && immutable' }
  environment {
    REPO = 'metricbeat-tests-poc'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    GOPATH = "${env.WORKSPACE}"
    GO_VERSION = "1.12.7"
    HOME = "${env.WORKSPACE}"
    NOTIFY_TO = credentials('notify-to')
    PATH = "${env.PATH}:${env.WORKSPACE}/bin:${env.WORKSPACE}/${env.BASE_DIR}/.ci/scripts"
    JOB_GCS_BUCKET = credentials('gcs-bucket')
    JOB_GIT_CREDENTIALS = '2a9602aa-ab9f-4e52-baf3-b71ca88469c7-UserAndToken'
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  triggers {
    issueCommentTrigger('(?i).*jenkins\\W+run\\W+the\\W+tests.*')
  }
  parameters {
    string(name: 'STACK_VERSION', defaultValue: '', description: 'SemVer version of the stack to be uses.')
    string(name: 'STACK_COMMIT_ID', defaultValue: '', description: 'Git sha of the stack to be used (6 digits). Will use a release if empty')
    string(name: 'BEAT_VERSION', defaultValue: '', description: 'SemVer version of the beat to be used.')
    string(name: 'BEAT_COMMIT_ID', defaultValue: '', description: 'Git sha of the beat to be used (6 digits). Will use a release if empty')
    string(name: 'FEATURE', defaultValue: '', description: 'What feature to be tested out (feature filename or all are allowed)')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: BASE_DIR, repo: "git@github.com:elastic/${env.REPO}.git", branch: 'master',
                    credentialsId: env.JOB_GIT_CREDENTIALS, githubNotifyFirstTimeContributor: false)
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
        stash allowEmpty: false, name: 'scripts', useDefaultExcludes: true, includes: "${BASE_DIR}/.ci/**"
      }
    }
    stage('Prepare Stack artifacts') {
      // reusing agent for Docker cache
      options { skipDefaultCheckout() }
      when {
        expression {
          params.STACK_COMMIT_ID != ''
        }
      }
      steps {
        githubCheckNotify('PENDING')
        deleteDir()
        unstash 'scripts'
        dir(BASE_DIR){
          sh label: 'Build Stack', script: """.ci/scripts/get-build-artifacts.sh "elasticsearch" "${params.STACK_VERSION}" "${params.STACK_COMMIT_ID}" """
          sh label: 'Build Stack', script: """.ci/scripts/get-build-artifacts.sh "kibana" "${params.STACK_VERSION}" "${params.STACK_COMMIT_ID}" """
        }
      }
      post {
        failure {
          githubCheckNotify(currentBuild.currentResult)
        }
      }
    }
    stage('Prepare Beats artifacts') {
      // reusing agent for Docker cache
      options { skipDefaultCheckout() }
      when {
        expression {
          params.BEAT_COMMIT_ID != ''
        }
      }
      steps {
        githubCheckNotify('PENDING')
        deleteDir()
        unstash 'scripts'
        dir(BASE_DIR){
          sh label: 'Build metricbeat', script: """.ci/scripts/get-build-artifacts.sh "metricbeat" "${params.BEAT_VERSION}" "${params.BEAT_COMMIT_ID}" """
        }
      }
      post {
        failure {
          githubCheckNotify(currentBuild.currentResult)
        }
      }
    }
    stage('Functional testing') {
      options { skipDefaultCheckout() }
      environment {
        GO111MODULE = 'on'
        PATH = "${env.WORKSPACE}/${env.BASE_DIR}/bin:${env.GOPATH}/bin:${HOME}/go/bin:${env.PATH}"
        GOPROXY = 'https://proxy.golang.org'
        STACK_VERSION = "${params.STACK_VERSION}"
        METRICBEAT_VERSION = "${params.METRICBEAT_VERSION}"
      }
      steps {
        deleteDir()
        unstash 'source'
        dir(BASE_DIR){
          sh script: """.ci/scripts/functional-test.sh "${GO_VERSION}" "${params.FEATURE}" """, label: 'Run functional tests'
        }
      }
      post {
        always {
          junit(allowEmptyResults: true, keepLongStdio: true, testResults: "${BASE_DIR}/outputs/TEST-*.xml")
          archiveArtifacts allowEmptyArchive: true, artifacts: "${BASE_DIR}/outputs/TEST-*.xml"
          githubCheckNotify(currentBuild.currentResult == 'SUCCESS' ? 'SUCCESS' : 'FAILURE')
        }
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult(to: ['victor.martinez@elastic.co', 'manuel.delapena@elastic.co'])
    }
  }
}


/**
 Notify the GitHub check of the parent stream
**/
def githubCheckNotify(String status) {
  if (params.GITHUB_CHECK_NAME?.trim() && params.GITHUB_CHECK_REPO?.trim() && params.GITHUB_CHECK_SHA1?.trim()) {
    githubNotify context: "${params.GITHUB_CHECK_NAME}",
                 description: "${params.GITHUB_CHECK_NAME} ${status.toLowerCase()}",
                 status: "${status}",
                 targetUrl: "${env.RUN_DISPLAY_URL}",
                 sha: params.GITHUB_CHECK_SHA1, account: 'elastic', repo: params.GITHUB_CHECK_REPO, credentialsId: env.JOB_GIT_CREDENTIALS
  }
}
