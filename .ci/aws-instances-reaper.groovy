#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-20' }
  environment {
    REPO = 'e2e-testing'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    HOME = "${env.WORKSPACE}"
    NOTIFY_TO = credentials('notify-to')
    PIPELINE_LOG_LEVEL = 'INFO'
    JOB_GIT_CREDENTIALS = "f6c7695a-671e-4f4f-a331-acdce44ff9ba"
    AWS_PROVISIONER_SECRET = 'secret/observability-team/ci/elastic-observability-aws-account-auth'
    AWS_EC2_INSTANCES_TAG_NAME= 'ReaperMark'
    AWS_EC2_INSTANCES_TAG_VALUE= 'e2e-testing-vm'
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
    cron '0 0 * * 0'
  }
  stages {
    stage('Checkout') {
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}",
          branch: "main",
          repo: "https://github.com/elastic/${REPO}.git",
          credentialsId: "${JOB_GIT_CREDENTIALS}"
        )
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
      }
    }
    stage('Reap AWS instances'){
      environment {
        HOME = "${env.WORKSPACE}/${BASE_DIR}"
      }
      steps {
        deleteDir()
        unstash 'source'
        withAWSEnv(secret: "${env.AWS_PROVISIONER_SECRET}", forceInstallation: true) {
          sh("aws ec2 terminate-instances --instance-ids `aws ec2 describe-instances --filters Name=tag:${env.AWS_EC2_INSTANCES_TAG_NAME},Values=${env.AWS_EC2_INSTANCES_TAG_VALUE} --query Reservations[].Instances[].InstanceId --output text`")
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
