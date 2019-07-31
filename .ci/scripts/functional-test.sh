#!/usr/bin/env bash
set -euxo pipefail

source ./.ci/scripts/install-go.sh

rm -rf outputs || true
mkdir -p outputs
REPORT=outputs/junit-functional-tests

## Generate test report even if make failed.
set +e
if ! make functional-test-ci | tee ${REPORT} ; then
  echo 'ERROR: functional-test-ci failed'
  exit_status=1
fi

sed -e 's/^[ \t]*//' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
exit $exit_status
