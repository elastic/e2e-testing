#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -exo pipefail

readonly supportedOSS=("darwin" "linux" "windows")
readonly supportedArchs=("386" "amd64")

arch="${TARGET_ARCH}"
extension=""
goos="${TARGET_OS}"
separator=""

if [[ "${TARGET_OS}" != "" ]]; then
    # GO_OS represents the Golang's supported Operative Systems.
    # Possible values: darwin, linux, windows
    readonly GO_OS="${TARGET_OS:-linux}"
    if [[ ! " ${supportedOSS[@]} " =~ " ${GO_OS} " ]]; then
        echo "It's not possible to build a binary for ${GO_OS}. Supported values: darwin, linux, windows"
        exit 1
    fi

    goos="${GO_OS}"
    if [[ "$GO_OS" == "windows" ]]; then
        goos="win"
        extension=".exe"
    fi

    separator="-"
fi

if [[ "${TARGET_ARCH}" != "" ]]; then
    # GO_ARCH represents the Golang's supported Architectures.
    # Possible values: 386, amd64
    readonly GO_ARCH="${TARGET_ARCH:-amd64}"
    if [[ ! " ${supportedArchs[@]} " =~ " ${GO_ARCH} " ]]; then
        echo "It's not possible to build a binary for ${GO_ARCH}. Supported values: 386, amd64"
        exit 1
    fi

    arch="${GO_ARCH}"
    if [[ "$GO_ARCH" == "amd64" ]]; then
        arch="64"
    fi

    separator="-"
fi

readonly VERSION="$(cat ./VERSION.txt)"

echo ">>> Installing Packr2"
go get github.com/gobuffalo/packr/v2/packr2@v2.7.1

echo ">>> Generating Packr boxes"
packr2

echo ">>> Building CLI"
GOOS=${GO_OS} GOARCH=${GO_ARCH} go build -v -o $(pwd)/.github/releases/download/${VERSION}/${goos}${arch}${separator}op${extension}

echo ">>> Cleaning up Packr boxes"
packr2 clean
