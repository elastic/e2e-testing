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
#   - HELM_VERSION - that's the Helm version which will be installed and enabled.
#   - KIND_VERSION - that's the Kind version which will be installed and enabled.
#   - KUBERNETES_VERSION - that's the Kubernetes version which will be installed and enabled.
#

GO_VERSION=${1:?GO_VERSION is not set}
SUITE=${2:?SUITE is not set}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

# execute specific test dependencies if it exists
if [ -f .ci/scripts/install-${SUITE}-dependencies.sh ]
then
    .ci/scripts/install-${SUITE}-dependencies.sh
else
    echo "Not installing test dependencies for ${SUITE}"
fi
