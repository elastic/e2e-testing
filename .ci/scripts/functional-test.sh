#!/usr/bin/env bash
set -euxo pipefail

GO_VERSION=${1}
FEATURE=${2:-''}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

rm -rf outputs || true
mkdir -p outputs
REPORT=outputs/junit-functional-tests

## Parse FEATURE if ALL
if [ -n "${FEATURE}" ] ; then
  if [ "${FEATURE}" == "all" ] ; then
    FEATURE='*'
  fi
else
  FEATURE='*'
fi

## Generate test report even if make failed.
set +e
exit_status=0
if ! REPORT=${REPORT} FEATURE=${FEATURE} FORMAT=junit make functional-test ; then
  echo 'ERROR: functional-test failed'
  exit_status=1
fi

sed -e 's/^[ \t]*//' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
exit $exit_status
