@Library('apm@current') _

pipeline {
  agent none
  environment {
    NOTIFY_TO = credentials('notify-to')
    PIPELINE_LOG_LEVEL = 'INFO'
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
  }
  triggers {
    cron('H H(4-5) * * 1-5')
  }
  stages {
    stage('Nighly e2e builds') {
      steps {
        runBuilds(quietPeriodFactor: 100, branches: ['main', '8.<minor>', '8.<next-patch>', '7.<minor>'])
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult(prComment: false)
    }
  }
}

def runBuilds(Map args = [:]) {
  def branches = getBranchesFromAliases(aliases: args.branches)

  def quietPeriod = 0
  branches.each { branch ->
    build(quietPeriod: quietPeriod, job: "e2e-tests/e2e-testing-fleet-daily-mbp/${branch}", wait: false, propagate: false)
		build(quietPeriod: quietPeriod, job: "e2e-tests/e2e-testing-helm-daily-mbp/${branch}", wait: false, propagate: false)
		build(quietPeriod: quietPeriod, job: "e2e-tests/e2e-testing-k8s-autodiscovery-daily-mbp/${branch}", wait: false, propagate: false)
    // Increate the quiet period for the next iteration
    quietPeriod += args.quietPeriodFactor
  }
}
