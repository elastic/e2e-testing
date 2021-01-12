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
#   - STACK_VERSION - that's the version of the stack to be tested. Default '7.11.0-SNAPSHOT'.
#   - METRICBEAT_VERSION - that's the version of the metricbeat to be tested. Default '7.11.0-SNAPSHOT'.
#

SUITE=${1:-''}
TAGS=${2:-''}
STACK_VERSION=${3:-'7.11.0-SNAPSHOT'}
METRICBEAT_VERSION=${4:-'7.11.0-SNAPSHOT'}
TARGET_OS=${GOOS:-linux}
TARGET_ARCH=${GOARCH:-amd64}

rm -rf outputs || true
mkdir -p outputs

REPORT="outputs/TEST-${SUITE}"

## Generate test report even if make failed.
set +e
exit_status=0
if ! SUITE=${SUITE} TAGS="${TAGS}" FORMAT=junit STACK_VERSION=${STACK_VERSION} METRICBEAT_VERSION=${METRICBEAT_VERSION} make --no-print-directory -C e2e functional-test | tee ${REPORT}  ; then
  echo 'ERROR: functional-test failed'
  exit_status=1
fi

## Transform report to Junit by parsing the stdout generated previously
sed -e 's/^[ \t]*//; s#>.*failed$#>#g' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
exit $exit_status
