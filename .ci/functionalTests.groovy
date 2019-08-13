#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent any
  environment {
    REPO = 'metricbeat-tests-poc'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    BEATS_BASE_DIR = 'src/github.com/elastic/beats'
    NOTIFY_TO = credentials('notify-to')
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
    issueCommentTrigger('(?i).*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*')
  }
  parameters {
    string(name: 'GO_VERSION', defaultValue: '1.12.7', description: "Go version to use.")
    string(name: 'FEATURE', defaultValue: '', description: 'What feature to be tested out (feature filename or all are allowed)')
    string(name: 'GITHUB_CHECK_NAME', defaultValue: '', description: 'Name of the GitHub check to be updated.')
    string(name: 'GITHUB_CHECK_REPO', defaultValue: '', description: 'Name of the GitHub repo to be updated.')
    string(name: 'GITHUB_CHECK_SHA1', defaultValue: '', description: 'Git sha to be used.')
  }
  stages {
    stage('Initializing'){
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
      environment {
        HOME = "${env.WORKSPACE}"
        GOPATH = "${env.WORKSPACE}"
        GO_VERSION = "${params.GO_VERSION.trim()}"
        PATH = "${env.PATH}:${env.WORKSPACE}/bin:${env.WORKSPACE}/${env.BASE_DIR}/.ci/scripts"
      }
      stages {
        stage('Checkout') {
          steps {
            gitCheckout(basedir: BASE_DIR, branch: 'master', credentialsId: env.JOB_GIT_CREDENTIALS, githubNotifyFirstTimeContributor: false)
            stash allowEmpty: true, name: 'source', useDefaultExcludes: false
            stash allowEmpty: false, name: 'scripts', useDefaultExcludes: true, includes: "${BASE_DIR}/.ci/**"
          }
        }
        stage('Beats') {
          agent { label 'linux && immutable && docker' }
          options { skipDefaultCheckout() }
          environment {
            PLATFORMS = 'linux/amd64'
          }
          steps {
            githubCheckNotify('PENDING')
            deleteDir()
            gitCheckout(basedir: env.BEATS_BASE_DIR, repo: 'git@github.com:elastic/beats.git',
                        branch: params.GITHUB_CHECK_SHA1, credentialsId: env.JOB_GIT_CREDENTIALS)
            unstash 'scripts'
            dir(BASE_DIR){
              sh script: """.ci/scripts/build-metricbeats.sh "${GO_VERSION}" "${WORKSPACE}/${BEATS_BASE_DIR}/metricbeat" """, label: 'Build metricbeats'
            }
            dir("${BEATS_BASE_DIR}/metricbeat/build/distributions"){
              stash allowEmpty: false, name: 'docker', useDefaultExcludes: true, includes: '*docker.tar.gz'
            }
          }
          post {
            failure {
              githubCheckNotify(currentBuild.currentResult)
            }
          }
        }
        stage('Functional testing') {
          agent { label 'linux && immutable && docker' }
          options { skipDefaultCheckout() }
          environment {
            GO111MODULE = 'on'
            GOPROXY = 'https://proxy.golang.org'
          }
          steps {
            deleteDir()
            unstash 'source'
            dir(BASE_DIR){
              unstash 'docker'
              sh script: '.ci/scripts/tag-metricbeats.sh', label: 'Create docker tag'
              sh script: """.ci/scripts/functional-test.sh "${GO_VERSION}" "${FEATURE}" """, label: 'Run functional tests'
            }
          }
          post {
            always {
              junit(allowEmptyResults: true, keepLongStdio: true, testResults: "${BASE_DIR}/outputs/junit-*.xml")
              archiveArtifacts allowEmptyArchive: true, artifacts: "${BASE_DIR}/outputs/junit-*"
              githubCheckNotify(currentBuild.currentResult == 'SUCCESS' ? 'SUCCESS' : 'FAILURE')
            }
          }
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
