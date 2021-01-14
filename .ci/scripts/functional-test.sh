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
#   - STACK_VERSION - that's the version of the stack to be tested. Default '7.10-SNAPSHOT'.
#   - METRICBEAT_VERSION - that's the version of the metricbeat to be tested. Default '7.10-SNAPSHOT'.
#

SUITE=${1:-''}
TAGS=${2:-''}
STACK_VERSION=${3:-'7.10-SNAPSHOT'}
METRICBEAT_VERSION=${4:-'7.10-SNAPSHOT'}
TARGET_OS=${GOOS:-linux}
TARGET_ARCH=${GOARCH:-amd64}

rm -rf outputs || true
mkdir -p outputs

REPORT="$(pwd)/outputs/TEST-${SUITE}"

## Generate test report even if make failed.
set +e
exit_status=0
if ! SUITE=${SUITE} TAGS="${TAGS}" FORMAT=junit:${REPORT}.xml STACK_VERSION=${STACK_VERSION} METRICBEAT_VERSION=${METRICBEAT_VERSION} make --no-print-directory -C e2e functional-test ; then
  echo 'ERROR: functional-test failed'
  exit_status=1
fi

exit $exit_status
