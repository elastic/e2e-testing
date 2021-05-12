#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Run the functional tests for metricbeat using the functional-test wrapper
#
# Parameters:
#   - STACK_VERSION - that's the version of the stack to be tested. Default is stored in '.stack-version'.
#   - BEAT_VERSION - that's the version of the metricbeat to be tested. Default is stored in '.stack-version'.
#

BASE_VERSION="$(cat $(pwd)/.stack-version)"

STACK_VERSION=${1:-"${BASE_VERSION}"}
BEAT_VERSION=${2:-"${BASE_VERSION}"}
SUITE='metricbeat'

.ci/scripts/functional-test.sh "${SUITE}" "" "${STACK_VERSION}" "${BEAT_VERSION}"
