#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#
# Parameters:
#   - KIND_VERSION - that's the Kind version which will be installed and enabled.
#   - KUBERNETES_VERSION - that's the Kubernetes version which will be installed and enabled.
#

MSG="parameter missing."
HOME=${HOME:?$MSG}

KIND_VERSION="v${KIND_VERSION:-"0.17.0"}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-"1.26.0"}"

KUBECTL_CMD="${HOME}/bin/kubectl"

# Install kind as a Go binary
GO111MODULE="on" go get sigs.k8s.io/kind@${KIND_VERSION}

mkdir -p "${HOME}/bin"

# Install kubectl
curl -sSLo "${KUBECTL_CMD}" "https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VERSION}/bin/linux/amd64/kubectl"
chmod +x "${KUBECTL_CMD}"
${KUBECTL_CMD} version --client
