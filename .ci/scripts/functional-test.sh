#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

# Prepare junit build context
mkdir -p outputs
export OUT_FILE="outputs/functional-test-report.out"

# TODO: How to resolve this within the go dependencies
go get github.com/DATA-DOG/godog/cmd/godog

make functional-test | tee ${OUT_FILE}

go get -v -u github.com/jstemmer/go-junit-report
go-junit-report > outputs/junit-functional-tests.xml < ${OUT_FILE}
