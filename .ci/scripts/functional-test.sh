#!/usr/bin/env bash
set -exo pipefail

source ./.ci/scripts/install-go.sh

rm -rf outputs || true
mkdir -p outputs
REPORT=outputs/junit-functional-tests

## Generate test report even if make failed.
set +e
if ! REPORT=${REPORT} make functional-test-ci ; then
  echo 'ERROR: functional-test-ci failed'
  exit_status=1
fi

sed -e 's/^[ \t]*//' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
exit $exit_status
