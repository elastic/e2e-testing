#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

# Prepare junit build context
mkdir -p outputs
export OUT_FILE="outputs/test-report.out"

make install test | tee ${OUT_FILE}

go get -v -u github.com/t-yuki/gocover-cobertura
go-junit-report > outputs/junit-tests.xml < ${OUT_FILE}
