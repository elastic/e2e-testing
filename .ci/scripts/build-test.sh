#!/usr/bin/env bash
set -euxo pipefail

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh

# Prepare junit build context
mkdir -p outputs
export OUT_FILE="outputs/test-report.out"

make install test | tee ${OUT_FILE}

go get -v -u github.com/jstemmer/go-junit-report
go-junit-report > outputs/junit-tests.xml < ${OUT_FILE}
