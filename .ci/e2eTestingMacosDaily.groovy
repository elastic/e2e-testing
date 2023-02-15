#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'macos12 && x86_64' }
  environment {
    REPO = 'e2e-testing'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    ELASTIC_APM_ACTIVE = "true"
    ELASTIC_APM_ENVIRONMENT = "ci"
    ELASTIC_APM_LOG_FILE = "stderr"
    ELASTIC_APM_LOG_LEVEL = "debug"
    NOTIFY_TO = credentials('notify-to')
    PROVIDER = 'remote'
    BEAT_VERSION = "${params.BEAT_VERSION.trim()}"
    ELASTIC_AGENT_VERSION = "${params.ELASTIC_AGENT_VERSION.trim()}"
    ELASTIC_STACK_VERSION = "${params.ELASTIC_STACK_VERSION.trim()}"
    GITHUB_CHECK_REPO = "${params.GITHUB_CHECK_REPO.trim()}"
    GITHUB_CHECK_SHA1 = "${params.GITHUB_CHECK_SHA1.trim()}"
    CLUSTER_NAME = "e2e-testing-${BUILD_ID}-${BRANCH_NAME}-${ELASTIC_STACK_VERSION.replaceAll('\\.', '-')}"
    LOG_LEVEL = "${params.LOG_LEVEL}"
    GO111MODULE = 'on'
    TAGS = "fleet_mode && install"
  }
  options {
    timeout(time: 120, unit: 'MINUTES')
    buildDiscarder(logRotator(numToKeepStr: '200', artifactNumToKeepStr: '30', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  parameters {
    string(name: 'ELASTIC_AGENT_VERSION', defaultValue: "8.6.2-SNAPSHOT", description: 'SemVer version of the Elastic Agent to be used for the tests. You can use here the tag of your PR to test your changes')
    string(name: 'ELASTIC_STACK_VERSION', defaultValue: "8.6.2-SNAPSHOT", description: 'SemVer version of the stack to be used for the tests.')
    string(name: 'BEAT_VERSION', defaultValue: "8.6.2-SNAPSHOT", description: 'SemVer version of the Beat to be used for the tests. You can use here the tag of your PR to test your changes')
    string(name: 'GITHUB_CHECK_REPO', defaultValue: '', description: 'Name of the GitHub repo to be updated. Only modified if this build is triggered from another parent stream (i.e. Beats).')
    string(name: 'GITHUB_CHECK_SHA1', defaultValue: '', description: 'Git SHA for the Beats upstream project (branch or PR)')
    choice(name: 'LOG_LEVEL', choices: ['TRACE', 'DEBUG', 'INFO'], description: 'Log level to be used')
    choice(name: 'TIMEOUT_FACTOR', choices: ['5', '3', '7', '11'], description: 'Max number of minutes for timeout backoff strategies')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: BASE_DIR, githubNotifyFirstTimeContributor: true)
        stash(allowEmpty: true, name: 'source', useDefaultExcludes: false)
        setEnvVar("GO_VERSION", readFile("${env.WORKSPACE}/${env.BASE_DIR}/.go-version").trim())
      }
    }
    stage('Create cluster') {
      options { skipDefaultCheckout() }
      steps {
        echo "CLUSTER_NAME=${env.CLUSTER_NAME}"
        createCluster()
      }
    }
    stage('Functional Test') {
      options { skipDefaultCheckout() }
      environment {
        E2E_SUITES = "e2e/_suites"
        FORMAT = "pretty,cucumber:fleet_mode.json,junit:fleet_mode.xml"
        STACK_VERSION = "${env.ELASTIC_STACK_VERSION}"
      }
      steps {
        withGithubNotify(context: 'Functional Test - MacOS', tab: 'tests') {
          withGoEnv(version: "${GO_VERSION}"){
            withClusterEnv(cluster: env.CLUSTER_NAME, fleet: true, kibana: true, elasticsearch: true) {
              withOtelEnv() {
                sh(label: 'run fleet', script: 'make --no-print-directory -C "${BASE_DIR}/${E2E_SUITES}/fleet" functional-test')
              }
            }
          }
        }
      }
      post {
        failure {
          // Notify test failures if the Functional Test stage failed only
          // therefore any other failures with the cluster creation/destroy
          // won't report any slack message.
          setEnvVar('SLACK_NOTIFY', true)
        }
        always {
          junit(allowEmptyResults: true, keepLongStdio: true, testResults: "${BASE_DIR}/${E2E_SUITES}/**/*.xml")
          archiveArtifacts(allowEmptyArchive: true, artifacts: "${BASE_DIR}/${E2E_SUITES}/**/*.xml")
        }
      }
    }
  }
  post {
    always {
      destroyCluster()
    }
    cleanup {
      notifyBuildResult(prComment: true,
                        slackHeader: "*Test Suite (MacOS)*: ${env.TAGS}",
                        slackChannel: "ingest-notifications",
                        slackComment: true,
                        slackNotify: doSlackNotify())
    }
  }
}

def doSlackNotify() {
  if (env.SLACK_NOTIFY?.equals('true')) {
    return true
  }
  return false
}

def createCluster() {
  withVaultEnv(){
    dir("${BASE_DIR}") {
      sshagent(credentials: ["f6c7695a-671e-4f4f-a331-acdce44ff9ba"]) {
        sh ".ci/scripts/deployment.sh 'create'"
      }
    }
  }
}

def destroyCluster() {
  withVaultEnv(){
    dir("${BASE_DIR}") {
      sshagent(credentials: ["f6c7695a-671e-4f4f-a331-acdce44ff9ba"]) {
        sh ".ci/scripts/deployment.sh 'destroy'"
      }
    }
  }
}

def withVaultEnv(Closure body){
  getVaultSecret.readSecretWrapper {
    withEnvMask(vars: [
      [var: 'VAULT_ADDR', password: env.VAULT_ADDR],
      [var: 'VAULT_ROLE_ID', password: env.VAULT_ROLE_ID],
      [var: 'VAULT_SECRET_ID', password: env.VAULT_SECRET_ID],
      [var: 'VAULT_AUTH_METHOD', password: 'approle'],
      [var: 'VAULT_AUTHTYPE', password: 'approle']
    ]){
      body()
    }
  }
}
