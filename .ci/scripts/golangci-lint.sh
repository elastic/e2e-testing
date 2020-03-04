#!/usr/bin/env bash

BASEDIR=$(dirname "$0")

error=0

for file in "$@"; do
    echo "golangci-lint run ${file}"
    if [[ $(echo $file |grep "^cli") ]]; then
        parsedFile=$(echo $file |sed "s#cli/##")
        cd cli
        golangci-lint run "${parsedFile}" || error=1
        cd -
    fi

    if [[ $(echo $file |grep "^metricbeat") ]]; then
        parsedFile=$(echo $file |sed "s#metricbeat/##")
        cd metricbeat
        golangci-lint run "${parsedFile}" || error=1
        cd -
    fi 
done

if [[ ${error} -gt 0 ]]; then
    echo "Lint failed!"
    exit ${error}
fi
