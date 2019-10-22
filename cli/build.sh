#!/usr/bin/env bash
set -exuo pipefail

readonly supportedOSS=("darwin" "linux" "windows")
readonly supportedArchs=("386" "amd64")

# GOOS represents the Golang's supported Operative Systems.
# Possible values: darwin, linux, windows
readonly GOOS="${GOOS:-linux}"
if [[ ! " ${supportedOSS[@]} " =~ " ${GOOS} " ]]; then
    echo "It's not possible to build a binary for ${GOOS}. Supported values: darwin, linux, windows"
    exit 1
fi

# GOARCH represents the Golang's supported Architectures.
# Possible values: 386, amd64
readonly GOARCH="${GOARCH:-amd64}"
if [[ ! " ${supportedArchs[@]} " =~ " ${GOARCH} " ]]; then
    echo "It's not possible to build a binary for ${GOARCH}. Supported values: 386, amd64"
    exit 1
fi

readonly GO_VERSION="${GO_VERSION:-v1.12.7}"
readonly GO_WORKSPACE="/usr/local/go/src/github.com/elastic/op"
readonly VERSION="$(cat ./VERSION.txt)"

arch="${GOARCH}"
goos="${GOOS}"
extension=""

if [[ "$GOARCH" == "amd64" ]]; then
    arch="64"
fi

if [[ "$GOOS" == "windows" ]]; then
    goos="win"
    extension=".exe"
fi

echo ">>> Building for ${GOOS}/${GOARCH}"
docker run --rm -v "$(pwd)":${GO_WORKSPACE} -w ${GO_WORKSPACE} \
    -e GOOS=${GOOS} -e GOARCH=${GOARCH} registry.hub.docker.com/drud/golang-build-container:v${GO_VERSION} \
    packr2 build -v -o ${GO_WORKSPACE}/.github/releases/download/${VERSION}/${goos}${arch}-op${extension}
