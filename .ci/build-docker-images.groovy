#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-20' }
  environment {
    REPO = 'e2e-testing'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    DOCKER_REGISTRY = 'docker.elastic.co'
    DOCKER_REGISTRY_SECRET = 'secret/observability-team/ci/docker-registry/prod'
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
  triggers {
    cron 'H H(0-5) * * 1-5'
  }
  stages {
    stage('Checkout') {
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}",
          branch: "${params.BRANCH_REFERENCE}",
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
      agent { label 'arm && immutable && docker' }
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
