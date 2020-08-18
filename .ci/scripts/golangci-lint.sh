#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

BASEDIR=$(dirname "$0")

error=0

function run() {
    local module=${1}
    local file=${2}

    cd ${module}
    if [[ $(echo $file |grep "^${module}") ]]; then
        parsedFile=$(echo $file |sed "s#${module}/##")
        golangci-lint run "${parsedFile}" || error=1
    fi
    cd -
}

for file in "$@"; do
    echo "golangci-lint run ${file}"
    run "cli" "${file}"
    run "e2e" "${file}"
done

if [[ ${error} -gt 0 ]]; then
    echo "Lint failed!"
    exit ${error}
fi
