#!/usr/bin/env bash
set -euxo pipefail

source ./.ci/scripts/install-go.sh

rm -rf outputs || true
mkdir -p outputs
REPORT=outputs/junit-functional-tests

make functional-test-ci | tee ${REPORT}
sed -e 's/^[ \t]*//' ${REPORT} | grep -E '^<.*>$' > ${REPORT}.xml
