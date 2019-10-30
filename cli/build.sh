#!/usr/bin/env bash
set -exuo pipefail

readonly supportedOSS=("darwin" "linux" "windows")
readonly supportedArchs=("386" "amd64")

# GO_OS represents the Golang's supported Operative Systems.
# Possible values: darwin, linux, windows
readonly GO_OS="${GOOS:-linux}"
if [[ ! " ${supportedOSS[@]} " =~ " ${GOOS} " ]]; then
    echo "It's not possible to build a binary for ${GO_OS}. Supported values: darwin, linux, windows"
    exit 1
fi

# GO_ARCH represents the Golang's supported Architectures.
# Possible values: 386, amd64
readonly GO_ARCH="${GOARCH:-amd64}"
if [[ ! " ${supportedArchs[@]} " =~ " ${GO_ARCH} " ]]; then
    echo "It's not possible to build a binary for ${GO_ARCH}. Supported values: 386, amd64"
    exit 1
fi

readonly GO_VERSION="${GO_VERSION:-1.13.0}"
readonly VERSION="$(cat ./VERSION.txt)"

arch="${GO_ARCH}"
goos="${GO_OS}"
extension=""

if [[ "$GO_ARCH" == "amd64" ]]; then
    arch="64"
fi

if [[ "$GO_OS" == "windows" ]]; then
    goos="win"
    extension=".exe"
fi

echo ">>> Installing Packr2"
go get -u github.com/gobuffalo/packr/v2/packr2@v2.7.1

echo ">>> Generating Packr boxes"
packr2

echo ">>> Building for ${GO_OS}/${GO_ARCH}"
GOOS=${GO_OS} GOARCH=${GO_ARCH} go build -v -o $(pwd)/.github/releases/download/${VERSION}/${goos}${arch}-op${extension}

echo ">>> Cleaning up Packr boxes"
packr2 clean
