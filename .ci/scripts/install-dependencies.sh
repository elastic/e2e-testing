#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#

# Install some other dependencies required for the pre-commit
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.34.1

go get -v golang.org/x/lint/golint
go get -v github.com/go-lintpack/lintpack/...
go get -v github.com/go-critic/go-critic/...
lintpack build -o bin/gocritic -linter.version='v0.3.4' -linter.name='gocritic' github.com/go-critic/go-critic/checkers

# Install project dependencies
make -C cli install
