#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    REPO = 'beats'
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
        [key: 'action', value: '$.action'],
        [key: 'pr_id', value: '$.number'],
        [key: 'pr_base_ref', value: '$.pull_request.base.ref'],
        [key: 'pr_head_ref', value: '$.pull_request.head.ref'],
        [key: 'pr_head_sha', value: '$.pull_request.head.sha'],
        [key: 'pr_title', value: '$.pull_request.title'],
        [key: 'repo_name', value: '$.repository.name'],
      ],
      genericHeaderVariables: [
        [key: 'x-github-event', regexpFilter: '']
      ],
      causeString: "Triggered on ${repo_name}#${pr_id}: ${pr_title}",
      //Allow to use a credential as a secret to trigger the webhook
      //tokenCredentialId: '',
      printContributedVariables: true,
      printPostContent: false,
      silentResponse: true,
      regexpFilterText: "$pr_head_ref-$x_github_event",
      regexpFilterExpression: '^(refs/tags/current|refs/heads/master/.+)-comment$'
    )
  }
  stages {
    stage('Process GitHub Event') {
      when {
        beforeAgent true
        anyOf {
          expression { env.action == "edited" }
          expression { env.action == "opened" }
          expression { env.action == "reopened" }
          expression { env.action == "synchronize" }
        }
      }
      steps {
        catchError(buildResult: 'UNSTABLE', message: 'Unable to run e2e tests', stageResult: 'FAILURE') {
          runE2ETests('fleet')
        }
      }
    }
  }
}

def runE2ETests(String suite) {
  log(level: 'DEBUG', text: "Triggering '${suite}' E2E tests for PR-${env.pr_id}.")

  // Kibana's maintenance branches follow the 7.11, 7.12 schema.
  def branchName = "${env.pr_base_ref}.x"
  def e2eTestsPipeline = "e2e-tests/e2e-testing-mbp/${branchName}"

  def parameters = [
    booleanParam(name: 'forceSkipGitChecks', value: true),
    booleanParam(name: 'forceSkipPresubmit', value: true),
    booleanParam(name: 'notifyOnGreenBuilds', value: !isPR()),
    booleanParam(name: 'BEATS_USE_CI_SNAPSHOTS', value: true),
    string(name: 'runTestsSuites', value: suite),
    string(name: 'GITHUB_CHECK_NAME', value: env.GITHUB_CHECK_E2E_TESTS_NAME),
    string(name: 'GITHUB_CHECK_REPO', value: env.REPO),
    string(name: 'GITHUB_CHECK_SHA1', value: env.GIT_BASE_COMMIT),
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
