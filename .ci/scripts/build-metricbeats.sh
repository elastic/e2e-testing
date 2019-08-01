#!/usr/bin/env bash
set -euxo pipefail

source ./.ci/scripts/install-go.sh

go get -u -d github.com/magefile/mage
pushd "$GOPATH/src/github.com/magefile/mage"
go run bootstrap.go

## For debugging purposes
go env
mage -version

popd
ls -ltrah
find . -name magefile.go
head -n 20 magefile.go || true
mage -debug -l
PLATFORMS=linux/amd64 mage -debug package
docker images
