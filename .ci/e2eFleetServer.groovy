#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    REPO = 'fleet-server'
    ELASTIC_REPO = "elastic/${env.REPO}"
    BASE_DIR = "src/github.com/${env.ELASTIC_REPO}"
    BEATS_REPO = 'beats'
    BEATS_ELASTIC_REPO = "elastic/${env.BEATS_REPO}"
    BEATS_BASE_DIR = "src/github.com/${env.BEATS_ELASTIC_REPO}"
    E2E_REPO = 'e2e-testing'
    E2E_ELASTIC_REPO = "elastic/${env.E2E_REPO}"
    E2E_BASE_DIR = "src/github.com/${env.E2E_ELASTIC_REPO}"
    DOCKER_REGISTRY = 'docker.elastic.co'
    DOCKER_REGISTRY_NAMESPACE = 'observability-ci'
    DOCKER_ELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    GITHUB_APP_SECRET = 'secret/observability-team/ci/github-app'
    GITHUB_CHECK_E2E_TESTS_NAME = 'E2E Tests'
    JOB_GIT_CREDENTIALS = "2a9602aa-ab9f-4e52-baf3-b71ca88469c7-UserAndToken"
    PIPELINE_LOG_LEVEL = "INFO"
    ELASTIC_AGENT_DROP_PATH = "/tmp/agent-drop-path"
  }
  options {
    timeout(time: 3, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    disableConcurrentBuilds()
  }
  // http://JENKINS_URL/generic-webhook-trigger/invoke
  // Pull requests events: https://docs.github.com/en/developers/webhooks-and-events/github-event-types#pullrequestevent
  triggers {
    GenericTrigger(
     genericVariables: [
      [key: 'GT_REPO', value: '$.repository.full_name'],
      [key: 'GT_PR', value: '$.issue.number'],
      [key: 'GT_BODY', value: '$.comment.body'],
      [key: 'GT_COMMENT_ID', value: '$.comment.id']
     ],
    genericHeaderVariables: [
     [key: 'x-github-event', regexpFilter: 'comment']
    ],
     causeString: 'Triggered on #$GT_PR via comment: $GT_BODY',
     printContributedVariables: false,
     printPostContent: false,
     silentResponse: true,
     regexpFilterText: '$GT_REPO$GT_BODY',
     regexpFilterExpression: '^elastic/fleet-server/run-fleet-e2e-tests$'
    )
  }
  parameters {
    string(name: 'fleet_server_pr', defaultValue: "master", description: "PR ID to use to run the E2E tests (e.g 10000)")
  }
  stages {
    stage('Initialize'){
      agent { label 'ubuntu-20 && immutable' }
      options { skipDefaultCheckout() }
      environment {
        HOME = "${env.WORKSPACE}/${BASE_DIR}"
      }
      stages {
        stage('Prepare workspace') {
          steps {
            checkPermissions()
            setEnvVar('E2E_BASE_BRANCH', getE2EBaseBranch())
            sh(label:'Prepare Agent Drop path', script: "mkdir -p ${ELASTIC_AGENT_DROP_PATH}")
          }
        }
        stage('Build Elastic Agent dependencies') {
          options { skipDefaultCheckout() }
          parallel {
            stage('Build Fleet Server') {
              options { skipDefaultCheckout() }
              steps {
                gitCheckout(basedir: BASE_DIR, branch: getFleetServerBranch(), repo: "git@github.com:${env.ELASTIC_REPO}.git", credentialsId: env.JOB_GIT_CREDENTIALS)
                dir("${BASE_DIR}") {
                  withGoEnv(){
                    sh(label: 'Build Fleet Server', script: "make docker-release")
                    sh(label: 'Copy binaries to Agent Drop path', script: 'cp build/distributions/* ${ELASTIC_AGENT_DROP_PATH}')
                  }
                }
              }
            }
            stage('Build Elastic Agent Dependencies') {
              options { skipDefaultCheckout() }
              steps {
                gitCheckout(basedir: BEATS_BASE_DIR, branch: "${BASE_REF}", repo: "git@github.com:${env.BEATS_ELASTIC_REPO}.git", credentialsId: env.JOB_GIT_CREDENTIALS)
                dir("${BEATS_BASE_DIR}/x-pack/elastic-agent") {
                  withMageEnv(){
                    sh(label: 'Build Fleet Server', script: 'DEV=true SNAPSHOT=true PLATFORMS="+all linux/amd64" go run github.com/magefile/mage package')
                  }
                }
                dir("${BEATS_BASE_DIR}/x-pack") {
                  sh(label:'Copy Filebeat binaries to Agent Drop path', script: 'cp filebeat/build/distributions/* ${ELASTIC_AGENT_DROP_PATH}')
                  sh(label:'Copy Heartbeat binaries to Agent Drop path', script: 'cp metricbeat/build/distributions/* ${ELASTIC_AGENT_DROP_PATH}')
                  sh(label:'Copy Metricbeat binaries to Agent Drop path', script: 'cp heartbeat/build/distributions/* ${ELASTIC_AGENT_DROP_PATH}')
                }
              }
            }
          }
        }
        stage('Build Elastic Agent including Fleet Server') {
          options { skipDefaultCheckout() }
          steps {
            dir("${BEATS_BASE_DIR}/x-pack/elastic-agent") {
              withGoEnv(){
                sh(label: 'Build Fleet Server', script: "AGENT_DROP_PATH='${env.ELASTIC_AGENT_DROP_PATH}' DEV=true SNAPSHOT=true PLATFORMS='+all linux/amd64' go run github.com/magefile/mage package")
              }
            }
          }
        }
        stage('Run E2E Tests') {
          options { skipDefaultCheckout() }
          steps {
            gitCheckout(basedir: E2E_BASE_DIR, branch: "${env.E2E_BASE_BRANCH}", repo: 'git@github.com:elastic/e2e-testing.git', credentialsId: env.JOB_GIT_CREDENTIALS)
            dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
            dir("${E2E_BASE_DIR}") {
              withGoEnv(){
                withEnv(["BEATS_LOCAL_PATH='${env.BEATS_BASE_DIR}"]) {
                  sh(label: 'Run E2E Tests', script: './.ci/scripts/fleet-test.sh ')
                }
              }
            }
          }
        }
      }
    }
  }
}

def checkPermissions(){
  if(env.GT_PR){
    if(!githubPrCheckApproved(changeId: "${env.GT_PR}", org: 'elastic', repo: 'kibana')){
      error("Only PRs from Elasticians can be tested with Fleet E2E tests")
    }

    if(!hasCommentAuthorWritePermissions(repoName: "${env.ELASTIC_REPO}", commentId: env.GT_COMMENT_ID)){
      error("Only Elasticians can trigger Fleet E2E tests")
    }
  }
}

def getE2EBaseBranch() {
  // we need a second API request, as the issue_comment API does not retrieve data about the pull request
  // See https://docs.github.com/en/developers/webhooks-and-events/webhook-events-and-payloads#issue_comment
  def prID = getID()

  if (!prID.isInteger()) {
    // we are building a branch on Fleet Server
    setEnvVar('BASE_REF', prID)
    // in the case we are triggering the job for a branch (i.e master, 7.x) we directly use branch name as Docker tag
    return prID
  }

  def token = githubAppToken(secret: "${env.GITHUB_APP_SECRET}")

  def pullRequest = githubApiCall(token: token, url: "https://api.github.com/repos/${env.ELASTIC_REPO}/pulls/${prID}")
  def baseRef = pullRequest?.base?.ref
  setEnvVar('BASE_REF', baseRef)
  //def headSha = pullRequest?.head?.sha

  return baseRef
}

def getID(){
  if(env.GT_PR){
    return "${env.GT_PR}"
  }
  
  return "${params.fleet_server_pr}"
}

def getFleetServerBranch(){
  def fleetServerBranch = getID()

  if (!fleetServerBranch.isInteger()) {
    return fleetServerBranch
  }

  return 'PR/'+fleetServerBranch
}
