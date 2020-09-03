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
  stages {
    stage('Run Tests') {
      steps {
        build(job: "e2e-tests/e2e-testing-mbp/${ env.JOB_BASE_NAME.startsWith('PR-') ? 'none' : env.JOB_BASE_NAME }",
          parameters: [
            booleanParam(name: 'forceSkipGitChecks', value: true),
            booleanParam(name: 'forceSkipPresubmit', value: true),
            string(name: 'runTestsSuite', value: 'helm'),
            string(name: 'SLACK_CHANNEL', value: "integrations"),
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
