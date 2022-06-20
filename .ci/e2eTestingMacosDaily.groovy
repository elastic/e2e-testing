#!/usr/bin/env groovy

@Library('apm@test/fleet-cloud') _

pipeline {
  agent { label 'darwin && orka && x86_64' }
  environment {
    REPO = 'e2e-testing'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    NOTIFY_TO = credentials('notify-to')
    BEAT_VERSION = "${params.BEAT_VERSION.trim()}"
    ELASTIC_AGENT_VERSION = "${params.ELASTIC_AGENT_VERSION.trim()}"
    ELASTIC_STACK_VERSION = "${params.ELASTIC_STACK_VERSION.trim()}"
  }
  options {
    timeout(time: 90, unit: 'MINUTES')
    buildDiscarder(logRotator(numToKeepStr: '200', artifactNumToKeepStr: '30', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  parameters {
    string(name: 'ELASTIC_AGENT_VERSION', defaultValue: '8.2.4-SNAPSHOT', description: 'SemVer version of the Elastic Agent to be used for the tests. You can use here the tag of your PR to test your changes')
    string(name: 'ELASTIC_STACK_VERSION', defaultValue: '8.2.4-SNAPSHOT', description: 'SemVer version of the stack to be used for the tests.')
    string(name: 'BEAT_VERSION', defaultValue: '8.2.4-SNAPSHOT', description: 'SemVer version of the Beat to be used for the tests. You can use here the tag of your PR to test your changes')
    choice(name: 'LOG_LEVEL', choices: ['TRACE', 'DEBUG', 'INFO'], description: 'Log level to be used')
    choice(name: 'TIMEOUT_FACTOR', choices: ['5', '3', '7', '11'], description: 'Max number of minutes for timeout backoff strategies')
  }
  stages {
    stage('Checkout') {
      steps {
        pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
        deleteDir()
        gitCheckout(basedir: BASE_DIR, githubNotifyFirstTimeContributor: true)
        stash(allowEmpty: true, name: 'source', useDefaultExcludes: false)
        setEnvVar("GO_VERSION", readFile("${env.WORKSPACE}/${env.BASE_DIR}/.go-version").trim())
      }
    }
    stage('Create cluster') {
      options { skipDefaultCheckout() }
      steps {
        echo 'TBD: create cluster'
      }
    }
    stage('Functional Test') {
      options { skipDefaultCheckout() }
      steps {
        echo 'TBD: run tests'
      }
    }
  }
  post {
		always {
			echo 'TBD: destroy cluster'
		}
    cleanup {
      notifyBuildResult()
    }
  }
}
