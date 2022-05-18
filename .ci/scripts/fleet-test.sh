#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Run the functional tests for fleets using the functional-test wrapper
#
# Parameters:
#   - STACK_VERSION - that's the version of the stack to be tested. Default is stored in '.stack-version'.
#

STACK_VERSION=${1:-"$(cat $(pwd)/.stack-version)"}
SUITE='fleet'

# Run all scenarios in the default profile
TAG=""

.ci/scripts/functional-test.sh "${SUITE}" "${TAG}" "${STACK_VERSION}"
