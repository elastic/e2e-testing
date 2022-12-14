#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Run the functional tests using the functional-test make goal and parse the
# output to JUnit format.
#
# Parameters:
#   - SUITE - that's the suite to be tested. Default '' which means all of them.
#   - TAGS - that's the tags to be tested. Default '' which means all of them.
#   - STACK_VERSION - that's the version of the stack to be tested. Default is stored in '.stack-version'.
#   - BEAT_VERSION - that's the version of the Beat to be tested. Default is stored in '.stack-version'.
#   - ELASTIC_AGENT_VERSION - that's the version of the Elastic Agent to be tested. Default is stored in '.stack-version'.
#
# NOTE: this script is replaced in runtime by .ci/ansible/tasks/setup_test_script.yml

BASE_VERSION="$(cat $(pwd)/.stack-version)"

SUITE=${1:-''}
TAGS=${2:-''}
STACK_VERSION=${3:-"${BASE_VERSION}"}
BEAT_VERSION=${4:-"${BASE_VERSION}"}
ELASTIC_AGENT_VERSION=${5:-"${BASE_VERSION}"}
GOARCH=${GOARCH:-"amd64"}
REPORT_PREFIX=${REPORT_PREFIX:-"${SUITE}_${GOARCH}_${TAGS}"}

## Install the required dependencies for the given SUITE
.ci/scripts/install-test-dependencies.sh "$(pwd)" "${SUITE}"

rm -rf outputs || true
mkdir -p outputs

OUTPUT_DIR=$(pwd)/outputs/docker-logs .ci/scripts/run_filebeat.sh

REPORT_PREFIX=$(echo "$REPORT_PREFIX" | sed -r 's/[ @~]+//g')
REPORT="$(pwd)/outputs/TEST-${REPORT_PREFIX}"

TAGS="${TAGS}" FORMAT="pretty,cucumber:${REPORT}.json,junit:${REPORT}.xml" GOARCH="${GOARCH}" STACK_VERSION="${STACK_VERSION}" BEAT_VERSION="${BEAT_VERSION}" ELASTIC_AGENT_VERSION="${ELASTIC_AGENT_VERSION}" make --no-print-directory -C e2e/_suites/${SUITE} functional-test
