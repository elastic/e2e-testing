#!/bin/bash

function git_branch_name() {
  echo $(git symbolic-ref --short HEAD)
}

readonly BRANCH="$(git_branch_name)"
readonly GO_VERSION="1.12.7"
readonly GO_WORKSPACE="/usr/local/go/src/github.com/elastic/op"
readonly VERSION="$(cat ./VERSION.txt)"

if [[ "$BRANCH" != "master" ]]; then
    echo "It's not possible to build from a branch different to master"
    exit 1
fi


for GOOS in darwin linux windows; do
    goos="${GOOS}"
    extension=""

    if [[ "$GOOS" == "windows" ]]; then
        goos="win"
        extension=".exe"
    fi

    for GOARCH in 386 amd64; do
        arch="${GOARCH}"

        if [[ "$GOARCH" == "amd64" ]]; then
            arch="64"
        fi

        echo ">>> Building for ${GOOS}/${GOARCH}"
        docker run --rm -v "$(pwd)":${GO_WORKSPACE} -w ${GO_WORKSPACE} \
            -e GOOS=${GOOS} -e GOARCH=${GOARCH} golang:${GO_VERSION} \
            go build -v -o ${GO_WORKSPACE}/.github/releases/download/${VERSION}/${goos}${arch}-op${extension}
    done
done