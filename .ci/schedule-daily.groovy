@Library('apm@main') _

pipeline {
  agent { label 'ubuntu-20 && immutable' }
  environment {
    NOTIFY_TO = credentials('notify-to')
    PIPELINE_LOG_LEVEL = 'INFO'
  }
  options {
    timeout(time: 120, unit: 'MINUTES')
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
        runBuilds(quietPeriodFactor: 100, branches: ['main', '8.<minor>', '8.<next-patch>', '8.<next-minor>', '8.<minor-1>', '7.<minor>'])
        runMacosBuilds(branches: ['main', '8.<minor>'])
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
    if (isBranchUnifiedReleaseAvailable(branch)) {
      build(quietPeriod: quietPeriod, job: "e2e-tests/e2e-testing-fleet-daily-mbp/${branch}", wait: false, propagate: false)
      build(quietPeriod: quietPeriod, job: "e2e-tests/e2e-testing-k8s-autodiscovery-daily-mbp/${branch}", wait: false, propagate: false)
      // Increate the quiet period for the next iteration
      quietPeriod += args.quietPeriodFactor
    }
  }
}

def runMacosBuilds(Map args = [:]) {
  def branches = getBranchesFromAliases(aliases: args.branches)

  branches.each { branch ->
    if (isBranchUnifiedReleaseAvailable(branch)) {
      build(quietPeriod: 0, job: "e2e-tests/e2e-testing-macos-daily-mbp/${branch}", wait: false, propagate: false)
    }
  }
}
