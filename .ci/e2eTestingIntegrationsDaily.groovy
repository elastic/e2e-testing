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
    JOB_GCS_BUCKET = credentials('gcs-bucket')
    NOTIFY_TO = credentials('notify-to')
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
    string(name: 'METRICBEAT_VERSION', defaultValue: '7.8.0', description: 'SemVer version of the metricbeat to be used.')
    choice(name: 'RETRY_TIMEOUT', choices: ['3', '5', '7', '11'], description: 'Max number of minutes for timeout backoff strategies')
    string(name: 'STACK_VERSION', defaultValue: '7.8.0', description: 'SemVer version of the stack to be used.')
    string(name: 'GO_VERSION', defaultValue: '1.13.4', description: "Go version to use.")
  }
  stages {
    stage('Run Tests') {
      steps {
        build(job: 'e2e-tests/e2e-testing-mbp/master',
          parameters: [
            string(name: 'runTestsSuite', value: 'metricbeat'),
            string(name: 'LOG_LEVEL', value: "${params.LOG_LEVEL.trim()}"),
            string(name: 'forceSkipGitChecks', value: "${params.forceSkipGitChecks}"),
            string(name: 'METRICBEAT_VERSION', value: "${params.METRICBEAT_VERSION.trim()}"),
            string(name: 'RETRY_TIMEOUT', value: "${params.RETRY_TIMEOUT.trim()}"),
            string(name: 'STACK_VERSION', value: "${params.STACK_VERSIONSTACK_VERSION.trim()}"),
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
