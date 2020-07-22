#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#
# Parameters:
#   - GO_VERSION - that's the Go version which will be used for running Go code.
#   - SUITE - that's the name of the test suite to install the dependencies for.
#

GO_VERSION=${1:?GO_VERSION is not set}
SUITE=${2:?SUITE is not set}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

# execute specific test dependencies if it exists
if [ -f .ci/scripts/install-${SUITE}-test-dependencies.sh ]
then
    source .ci/scripts/install-${SUITE}-test-dependencies.sh
else
    echo "Not installing test dependencies for ${SUITE}"
fi
