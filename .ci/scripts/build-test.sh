#!/usr/bin/env bash
set -euxo pipefail
#
# Build and test the app using the install and test make goals.
#
# Parameters:
#   - GO_VERSION - that's the version which will be installed and enabled.
#

GO_VERSION=${1:?GO_VERSION is not set}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

# Prepare junit build context
mkdir -p outputs
export OUT_FILE="outputs/test-report.out"

make install test | tee ${OUT_FILE}

go get -v -u github.com/jstemmer/go-junit-report
go-junit-report > outputs/junit-tests.xml < ${OUT_FILE}
