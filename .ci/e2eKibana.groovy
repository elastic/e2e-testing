#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    REPO = 'kibana'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    ELASTIC_REPO = "elastic/${env.REPO}"
    GITHUB_APP_SECRET = 'secret/observability-team/ci/github-app'
    GITHUB_CHECK_E2E_TESTS_NAME = 'E2E Tests'
    PIPELINE_LOG_LEVEL = "INFO"
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
     regexpFilterExpression: '^elastic/kibana/run-fleet-e2e-tests$'
    )
  }
  parameters {
    string(name: 'kibana_pr', defaultValue: "master", description: "PR ID to use to build the Docker image. (e.g 10000)")
  }
  stages {
    stage('Process GitHub Event') {
      environment {
        HOME = "${env.WORKSPACE}/${BASE_DIR}"
        PATH = "${env.HOME}/bin:${env.HOME}/node_modules:${env.HOME}/node_modules/.bin:${env.PATH}"
      }
      parallel {
        stage('AMD build') {
          agent { label 'ubuntu-20' }
          steps {
            buildPlatformImage()
          }
        }
        stage('ARM build') {
          agent { label 'arm' }
          steps {
            buildPlatformImage()
          }
        }
      }
    }
  }
}

def buildPlatformImage() {
  checkPermissions()
  buildKibanaDockerImage(refspec: getBranch())
  catchError(buildResult: 'UNSTABLE', message: 'Unable to run e2e tests', stageResult: 'FAILURE') {
    runE2ETests('fleet')
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

def getBranch(){
  return "PR/" + getID()
}

def getID(){
  if(env.GT_PR){
    return "${env.GT_PR}"
  }
  
  return "${params.kibana_pr}"
}

def runE2ETests(String suite) {
  // we need a second API request, as the issue_comment API does not retrieve data about the pull request
  // See https://docs.github.com/en/developers/webhooks-and-events/webhook-events-and-payloads#issue_comment
  def prID = getID()
  def token = githubAppToken(secret: "${env.GITHUB_APP_SECRET}")

  def pullRequest = githubApiCall(token: token, url: "https://api.github.com/repos/${env.ELASTIC_REPO}/pulls/${prID}")
  def baseRef = pullRequest?.base?.ref
  def headSha = pullRequest?.head?.sha

  // we are going to use the 'pr12345' tag as default
  def dockerTag = "pr${params.kibana_pr}"
  if(env.GT_PR){
    // it's a PR: we are going to use its head SHA as tag
    dockerTag = headSha
  }

  log(level: 'DEBUG', text: "Triggering '${suite}' E2E tests for PR-${prID} using '${dockerTag}' as Docker tag")

  // Kibana's maintenance branches follow the 7.11, 7.12 schema.
  def branchName = "${baseRef}"
  if (branchName != "master") {
    branchName += ".x"
  }
  def e2eTestsPipeline = "e2e-tests/e2e-testing-mbp/${branchName}"

  def parameters = [
    booleanParam(name: 'forceSkipGitChecks', value: true),
    booleanParam(name: 'forceSkipPresubmit', value: true),
    booleanParam(name: 'notifyOnGreenBuilds', value: false),
    booleanParam(name: 'BEATS_USE_CI_SNAPSHOTS', value: true),
    string(name: 'runTestsSuites', value: suite),
    string(name: 'GITHUB_CHECK_NAME', value: env.GITHUB_CHECK_E2E_TESTS_NAME),
    string(name: 'GITHUB_CHECK_REPO', value: env.REPO),
    string(name: 'KIBANA_VERSION', value: dockerTag),
  ]

  build(job: "${e2eTestsPipeline}",
    parameters: parameters,
    propagate: false,
    wait: false
  )

/*
  // commented out to avoid sending Github statuses to Kibana PRs
  def notifyContext = "${env.pr_head_sha}"
  githubNotify(context: "${notifyContext}", description: "${notifyContext} ...", status: 'PENDING', targetUrl: "${env.JENKINS_URL}search/?q=${e2eTestsPipeline.replaceAll('/','+')}")
*/
}
