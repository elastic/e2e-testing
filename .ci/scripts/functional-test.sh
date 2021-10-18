#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Run the functional tests using the functional-test make goal and parse the
# output to JUnit format.
#
# Parameters:
#   - SUITE - that's the suite to be tested. Default '' which means all of them.
#   - TAGS - that's the tags to be tested. Default '' which means all of them.
#   - STACK_VERSION - that's the version of the stack to be tested. Default is stored in '.stack-version'.
#   - BEAT_VERSION - that's the version of the Beat to be tested. Default is stored in '.stack-version'.
#

# Set proper path of script
pushd . > /dev/null
SCRIPT_PATH="${BASH_SOURCE[0]}"
if ([ -h "${SCRIPT_PATH}" ]); then
  while([ -h "${SCRIPT_PATH}" ]); do cd `dirname "$SCRIPT_PATH"`;
  SCRIPT_PATH=`readlink "${SCRIPT_PATH}"`; done
fi
cd `dirname ${SCRIPT_PATH}` > /dev/null
SCRIPT_PATH=`pwd`;
popd  > /dev/null

pushd "${SCRIPT_PATH}/../.."

BASE_VERSION="$(cat .stack-version)"

SUITE=${1:-''}
TAGS=${2:-''}
STACK_VERSION=${3:-"${BASE_VERSION}"}
BEAT_VERSION=${4:-"${BASE_VERSION}"}
GOARCH=${GOARCH:-"amd64"}
PATH=$PATH:/usr/local/go/bin

"${SCRIPT_PATH}/install-test-dependencies.sh" "${SUITE}"

rm -rf outputs || true
mkdir -p outputs

REPORT="outputs/TEST-${GOARCH}-${SUITE}"

TAGS="${TAGS}" FORMAT=junit:${REPORT}.xml GOARCH=${GOARCH} STACK_VERSION=${STACK_VERSION} BEAT_VERSION=${BEAT_VERSION} make --no-print-directory -C e2e/_suites/${SUITE} functional-test

popd
