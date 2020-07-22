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
#   - GO_VERSION - that's the version which will be installed and enabled.
#   - SUITE - that's the suite to be tested. Default '' which means all of them.
#   - FEATURE - that's the feature to be tested. Default '' which means all of them.
#   - STACK_VERSION - that's the version of the stack to be tested. Default '7.8.0'.
#   - METRICBEAT_VERSION - that's the version of the metricbeat to be tested. Default '7.8.0'.
#

GO_VERSION=${1:?GO_VERSION is not set}
SUITE=${2:-''}
FEATURE=${3:-''}
STACK_VERSION=${4:-'7.8.0'}
METRICBEAT_VERSION=${5:-'7.8.0'}
TARGET_OS=${GOOS:-linux}
TARGET_ARCH=${GOARCH:-amd64}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

rm -rf outputs || true
mkdir -p outputs

## Parse FEATURE if not ALL then enable the flags to be passed to the functional-test wrapper
REPORT=''
if [ "${FEATURE}" != "" ] && [ "${FEATURE}" != "all" ] ; then
  REPORT=outputs/TEST-${SUITE}-${FEATURE}
else
  FEATURE=''
  REPORT=outputs/TEST-functional-tests
fi

## Generate test report even if make failed.
set +e
exit_status=0
if ! SUITE=${SUITE} FEATURE=${FEATURE} FORMAT=junit STACK_VERSION=${STACK_VERSION} METRICBEAT_VERSION=${METRICBEAT_VERSION} make --no-print-directory -C e2e functional-test | tee ${REPORT}  ; then
  echo 'ERROR: functional-test failed'
  exit_status=1
fi

## Transform report to Junit by parsing the stdout generated previously
sed -e 's/^[ \t]*//; s#>.*failed$#>#g' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
exit $exit_status
