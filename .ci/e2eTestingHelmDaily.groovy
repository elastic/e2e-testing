// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

@Library('apm@current') _

pipeline {
  agent none
  environment {
    REPO = 'e2e-testing'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    GOPATH = "${env.WORKSPACE}"
    HOME = "${env.WORKSPACE}"
    JOB_GCS_BUCKET = credentials('gcs-bucket')
    NOTIFY_TO = credentials('notify-to')
    PATH = "${env.GOPATH}/bin:${env.PATH}"
    PIPELINE_LOG_LEVEL='INFO'
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
    cron('H H(4-5) * * 1-5')
  }
  parameters {
    choice(name: 'LOG_LEVEL', choices: ['INFO', 'DEBUG'], description: 'Log level to be used')
    booleanParam(name: "forceSkipGitChecks", defaultValue: true, description: "If it's needed to check for Git changes to filter by modified sources")
    string(name: 'HELM_CHART_VERSION', defaultValue: '7.6.1', description: 'SemVer version of Helm chart to be used.')
    string(name: 'HELM_VERSION', defaultValue: '2.16.3', description: 'SemVer version of Helm to be used.')
    string(name: 'KIND_VERSION', defaultValue: '0.7.0', description: 'SemVer version of Kind to be used.')
    string(name: 'KUBERNETES_VERSION', defaultValue: '1.15.3', description: 'SemVer version of Kubernetes to be used.')
    string(name: 'GO_VERSION', defaultValue: '1.13.4', description: "Go version to use.")
  }
  stages {
    stage('Run Tests') {
      steps {
        build(job: 'stack/e2e-testing-mbp/master',
          parameters: [
            string(name: 'runTestsSuite', value: 'helm'),
            string(name: 'LOG_LEVEL', value: "${params.LOG_LEVEL.trim()}"),
            string(name: 'forceSkipGitChecks', value: "${params.forceSkipGitChecks}"),
            string(name: 'HELM_CHART_VERSION', value: "${params.HELM_CHART_VERSION.trim()}"),
            string(name: 'HELM_VERSION', value: "${params.HELM_VERSION.trim()}"),
            string(name: 'KIND_VERSION', value: "${params.KIND_VERSION.trim()}"),
            string(name: 'KUBERNETES_VERSION', value: "${params.KUBERNETES_VERSION.trim()}"),
            string(name: 'GO_VERSION', value: "${params.GO_VERSION.trim()}")
          ],
          propagate: false,
          wait: false
        )
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult()
    }
  }
}
