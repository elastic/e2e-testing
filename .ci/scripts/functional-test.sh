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
#   - STACK_VERSION - that's the version of the stack to be tested. Default '8.0.0-SNAPSHOT'.
#   - BEAT_VERSION - that's the version of the metricbeat to be tested. Default '8.0.0-SNAPSHOT'.
#

SUITE=${1:-''}
TAGS=${2:-''}
STACK_VERSION=${3:-'8.0.0-SNAPSHOT'}
BEAT_VERSION=${4:-'8.0.0-SNAPSHOT'}

## Install the required dependencies for the given SUITE
.ci/scripts/install-test-dependencies.sh "${SUITE}"

rm -rf outputs || true
mkdir -p outputs

REPORT="$(pwd)/outputs/TEST-${SUITE}"

SUITE=${SUITE} TAGS="${TAGS}" FORMAT=junit:${REPORT}.xml STACK_VERSION=${STACK_VERSION} BEAT_VERSION=${BEAT_VERSION} make --no-print-directory -C e2e functional-test
