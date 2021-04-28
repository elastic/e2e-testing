#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-20' }
  environment {
    REPO = 'e2e-testing'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    DOCKER_REGISTRY = 'docker.elastic.co'
    DOCKER_ELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    HOME = "${env.WORKSPACE}"
    NOTIFY_TO = credentials('notify-to')
    PIPELINE_LOG_LEVEL = 'INFO'
    JOB_GIT_CREDENTIALS = "f6c7695a-671e-4f4f-a331-acdce44ff9ba"
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  parameters {
    string(name: 'BRANCH_SPECIFIER', defaultValue: 'master', description: 'It would not be defined on the first build, see JENKINS-41929.')
  }
  triggers {
    cron 'H H(4-5) * * 1-5'
  }
  stages {
    stage('Checkout') {
      environment {
        // Parameters will be empty for the very first build, setting an environment variable
        // with the same name will workaround the issue. see JENKINS-41929
        BRANCH_SPECIFIER = "${params?.BRANCH_SPECIFIER}"
      }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}",
          branch: "${env.BRANCH_SPECIFIER}",
          repo: "https://github.com/elastic/${REPO}.git",
          credentialsId: "${JOB_GIT_CREDENTIALS}"
        )
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
      }
    }
    stage('Build AMD Docker images'){
      agent { label 'ubuntu-20 && immutable && docker' }
      environment {
        HOME = "${env.WORKSPACE}/${BASE_DIR}"
      }
      steps {
        deleteDir()
        unstash 'source'
        dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
        dir("${BASE_DIR}") {
          withEnv(["ARCH=amd64"]) {
            sh(label: 'Build AMD images', script: '.ci/scripts/build-docker-images.sh')
          }
        }
      }
    }
    stage('Build ARM Docker images'){
      agent { label 'arm' }
      environment {
        HOME = "${env.WORKSPACE}/${BASE_DIR}"
      }
      steps {
        deleteDir()
        unstash 'source'
        dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
        dir("${BASE_DIR}") {
          withEnv(["ARCH=arm64"]) {
            sh(label: 'Build ARM images', script: '.ci/scripts/build-docker-images.sh')
          }
        }
      }
    }
    stage('Push multiplatform manifest'){
      agent { label 'ubuntu-20 && immutable && docker' }
      environment {
        HOME = "${env.WORKSPACE}/${BASE_DIR}"
      }
      steps {
        deleteDir()
        unstash 'source'
        dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
        dir("${BASE_DIR}") {
          sh(label: 'Push multiplatform manifest', script: '.ci/scripts/push-multiplatform-manifest.sh')
        }
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult()
    }
  }
}
