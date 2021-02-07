#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Run the functional tests for metricbeat using the functional-test wrapper
#
# Parameters:
#   - STACK_VERSION - that's the version of the stack to be tested. Default '8.0.0-SNAPSHOT'.
#   - METRICBEAT_VERSION - that's the version of the metricbeat to be tested. Default '8.0.0-SNAPSHOT'.
#

STACK_VERSION=${1:-'8.0.0-SNAPSHOT'}
METRICBEAT_VERSION=${2:-'8.0.0-SNAPSHOT'}

.ci/scripts/functional-test.sh "metricbeat" "" "${STACK_VERSION}" "${METRICBEAT_VERSION}"
