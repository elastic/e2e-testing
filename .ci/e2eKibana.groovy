#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    REPO = 'kibana'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
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
      [key: 'GT_BASE_REF', value: '$.pull_request.base.ref'],
      [key: 'GT_PR', value: '$.issue.number'],
      [key: 'GT_PR_HEAD_SHA', value: '$.pull_request.head.sha'],
      [key: 'GT_BODY', value: '$.comment.body'],
      [key: 'GT_COMMENT_ID', value: '$.comment.id']
     ],
    genericHeaderVariables: [
     [key: 'x-github-event', regexpFilter: 'comment']
    ],
     causeString: 'Triggered on comment: $GT_BODY',
     printContributedVariables: false,
     printPostContent: false,
     silentResponse: true,
     regexpFilterText: '$GT_REPO$GT_BODY',
     regexpFilterExpression: '^elastic/kibana/run-fleet-e2e-tests$'
    )
  }
  parameters {
    string(name: 'kibana_sha', defaultValue: "", description: "Commit to be used to run the Fleet E2E Tests. (e.g 1234567890abcdef)")
    string(name: 'kibana_branch', defaultValue: "master", description: "Branch/PR to use to build the Docker image. (e.g PR/10000)")
    string(name: 'target_tag', defaultValue: "", description: "Tag used for the generated Docker image. If empty, it will use the HEAD commit of the Kibana PR as Docker tag")
  }
  stages {
    stage('Process GitHub Event') {
      steps {
        checkPermissions()
        buildKibanaDockerImage(refspec: getBranch(), targetTag: "${params.target_tag}")
        catchError(buildResult: 'UNSTABLE', message: 'Unable to run e2e tests', stageResult: 'FAILURE') {
          runE2ETests('fleet')
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

    if(!hasCommentAuthorWritePermissions(env.GT_PR, env.GT_COMMENT_ID)){
      error("Only Elasticians can trigger Fleet E2E tests")
    }
  }
}

def getBranch(){
  if(env.GT_PR){
    return "PR/${env.GT_PR}"
  }
  
  return "${params.kibana_branch}"
}

def getCommit(){
  if(env.GT_PR){
    return "${env.GT_PR_HEAD_SHA}"
  }

  return "${params.kibana_sha}"
}

def hasCommentAuthorWritePermissions(prId, commentId){
  def repoName = "elastic/kibana"
  def token = getGithubToken()
  def url = "https://api.github.com/repos/${repoName}/issues/${prId}/comments/${commentId}"
  def comment = githubApiCall(token: token, url: url, noCache: true)
  def json = githubRepoGetUserPermission(token: token, repo: repoName, user: comment?.user?.login)

  return json?.permission == 'admin' || json?.permission == 'write'
}

def isEmptyString(value){
  return value == null || value?.trim() == ""
}

def runE2ETests(String suite) {
  log(level: 'DEBUG', text: "Triggering '${suite}' E2E tests for PR-${env.GT_PR}.")

  // Kibana's maintenance branches follow the 7.11, 7.12 schema.
  def branchName = "${env.GT_BASE_REF}.x"
  def e2eTestsPipeline = "e2e-tests/e2e-testing-mbp/${branchName}"

  def kibanaVersion = "${params.target_tag}"
  if (isEmptyString(params.target_tag)) {
    kibanaVersion = "${env.GT_PR_HEAD_SHA}"
  }

  def parameters = [
    booleanParam(name: 'forceSkipGitChecks', value: true),
    booleanParam(name: 'forceSkipPresubmit', value: true),
    booleanParam(name: 'notifyOnGreenBuilds', value: false),
    booleanParam(name: 'BEATS_USE_CI_SNAPSHOTS', value: true),
    string(name: 'runTestsSuites', value: suite),
    string(name: 'GITHUB_CHECK_NAME', value: env.GITHUB_CHECK_E2E_TESTS_NAME),
    string(name: 'GITHUB_CHECK_REPO', value: env.REPO),
    string(name: 'GITHUB_CHECK_SHA1', value: getCommit()),
    string(name: 'KIBANA_VERSION', value: "${kibanaVersion}"),
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
