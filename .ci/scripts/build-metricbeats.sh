#!/usr/bin/env bash
set -euxo pipefail
#
# Package the metricbeats project.
#
# Parameters:
#   - GO_VERSION - that's the version which will be installed and enabled.
#   - LOCATION  - that's the location where the metricbeat source code is stored.
#

GO_VERSION=${1:?GO_VERSION is not set}
LOCATION=${2}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

go get -u -d github.com/magefile/mage
pushd "$GOPATH/src/github.com/magefile/mage"
go run bootstrap.go

pushd "${LOCATION}"

## For debugging purposes
go env
mage -version

## Package the metricbeats
mage package
