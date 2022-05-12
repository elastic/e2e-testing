#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-20' }
  environment {
    HOME = "${env.WORKSPACE}"
    NOTIFY_TO = credentials('notify-to')
    PIPELINE_LOG_LEVEL = 'INFO'
    JOB_GIT_CREDENTIALS = "f6c7695a-671e-4f4f-a331-acdce44ff9ba"
    AWS_PROVISIONER_SECRET = 'secret/observability-team/ci/elastic-observability-aws-account-auth'
    AWS_DEFAULT_REGION = 'us-east-2'
    AWS_EC2_INSTANCES_TAG_NAME= 'reaper_mark'
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
    cron '30 1 * * *'
  }
  stages {
    stage('Reap AWS instances'){
      steps {
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
