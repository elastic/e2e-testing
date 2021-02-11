#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#

function installDependenciesForThePreCommit() {
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.34.1
    go get -v golang.org/x/lint/golint
}

function install() {
    CI_UTILS=/usr/local/bin/bash_standard_lib.sh
    if [ -e "${CI_UTILS}" ] ; then
        set +u
        if [ "$sourced" != "1" ] ; then
            # shellcheck disable=SC1090
            source "${CI_UTILS}"
            sourced=1
        fi
        set -u
        # shellcheck disable=SC2086
        retry 3 $1 # It is a function call
    else
        $1
    fi
}

# Install required dependencies with some retry
install installDependenciesForThePreCommit

# Install project dependencies
install make -C cli install
