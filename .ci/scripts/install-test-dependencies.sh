#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#
# Parameters:
#   - SUITE - that's the name of the test suite to install the dependencies for.
#

SUITE=${1:?SUITE is not set}

# execute specific test dependencies if it exists
if [ -f .ci/scripts/install-${SUITE}-test-dependencies.sh ]
then
    source .ci/scripts/install-${SUITE}-test-dependencies.sh
else
    echo "Not installing test dependencies for ${SUITE}"
fi
