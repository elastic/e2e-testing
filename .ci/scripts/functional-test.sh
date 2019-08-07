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

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

rm -rf outputs || true
mkdir -p outputs
REPORT=outputs/junit-functional-tests

## Parse FEATURE if not ALL then enable the flags to be passed to the functional-test wrapper
FLAG=''
if [ "${FEATURE}" != "" ] && [ "${FEATURE}" != "all" ] ; then
  FLAG='-t'
else
  FEATURE=''
fi

## Generate test report even if make failed.
set +e
exit_status=0
if ! FLAG=${FLAG} FEATURE=${FEATURE} FORMAT=junit make functional-test | tee ${REPORT}  ; then
  echo 'ERROR: functional-test failed'
  exit_status=1
fi

## Transform report to Junit by parsing the stdout generated previously
sed -e 's/^[ \t]*//; s#>.*failed$#>#g' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
exit $exit_status
