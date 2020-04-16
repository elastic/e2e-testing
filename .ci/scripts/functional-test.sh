#!/usr/bin/env bash
set -euxo pipefail
#
# Run the functional tests using the functional-test make goal and parse the
# output to JUnit format.
#
# Parameters:
#   - GO_VERSION - that's the version which will be installed and enabled.
#   - FEATURE  - that's the feature to be tested. Default '' which means all of them.
#

GO_VERSION=${1:?GO_VERSION is not set}
FEATURE=${2:-''}
STACK_VERSION=${3:-'7.6.0'}
METRICBEAT_VERSION=${4:-'7.6.0'}
TARGET_OS=${GOOS:-linux}
TARGET_ARCH=${GOARCH:-amd64}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

# Build OP Binary
GOOS=${TARGET_OS} GOARCH=${TARGET_ARCH} make -C e2e fetch-binary

# Build runtime dependencies
STACK_VERSION=${STACK_VERSION} make -C e2e run-elastic-stack

# Sync integrations
make -C e2e sync-integrations

rm -rf outputs || true
mkdir -p outputs

## Parse FEATURE if not ALL then enable the flags to be passed to the functional-test wrapper
REPORT=''
if [ "${FEATURE}" != "" ] && [ "${FEATURE}" != "all" ] ; then
  REPORT=outputs/TEST-${FEATURE}
else
  FEATURE=''
  REPORT=outputs/TEST-functional-tests
fi

## Generate test report even if make failed.
set +e
exit_status=0
if ! FEATURE=${FEATURE} FORMAT=junit STACK_VERSION=${STACK_VERSION} METRICBEAT_VERSION=${METRICBEAT_VERSION} make --no-print-directory -C e2e functional-test | tee ${REPORT}  ; then
  echo 'ERROR: functional-test failed'
  exit_status=1
fi

## Transform report to Junit by parsing the stdout generated previously
sed -i "s/testing: warning: no tests to run//" ${REPORT}
sed -e 's/^[ \t]*//; s#>.*failed$#>#g' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
exit $exit_status
