#!/usr/bin/env bash
set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#
# Parameters:
#   - GO_VERSION - that's the version which will be installed and enabled.
#

GO_VERSION=${1:?GO_VERSION is not set}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

# disable GOPROXY temporarily
GOPROXY_CONTEXT=${GOPROXY}
GOPROXY=''

# Install some other dependencies required for the pre-commit
go get -v -u golang.org/x/lint/golint
go get -v -u github.com/golangci/golangci-lint/cmd/golangci-lint@v1.18.0
go get -v -u github.com/go-lintpack/lintpack/...
go get -v -u github.com/go-critic/go-critic/...
lintpack build -o bin/gocritic -linter.version='v0.3.4' -linter.name='gocritic' github.com/go-critic/go-critic/checkers

# enable GOPROXY
GOPROXY=${GOPROXY_CONTEXT}

# Install project dependencies
make -C cli install
