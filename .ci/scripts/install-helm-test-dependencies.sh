#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#
# Parameters:
#   - HELM_VERSION - that's the Helm version which will be installed and enabled.
#   - HELM_KIND_VERSION - that's the Kind version which will be installed and enabled.
#   - HELM_KUBERNETES_VERSION - that's the Kubernetes version which will be installed and enabled.
#

MSG="parameter missing."
HOME=${HOME:?$MSG}

HELM_VERSION="${HELM_VERSION:-"3.5.2"}"
HELM_TAR_GZ_FILE="helm-v${HELM_VERSION}-linux-amd64.tar.gz"
HELM_KIND_VERSION="v${HELM_KIND_VERSION:-"0.10.0"}"
HELM_KUBERNETES_VERSION="${HELM_KUBERNETES_VERSION:-"1.18.2"}"

HELM_CMD="${HOME}/bin/helm"
KBC_CMD="${HOME}/bin/kubectl"

# Install kind as a Go binary
GO111MODULE="on" go get sigs.k8s.io/kind@${HELM_KIND_VERSION}

mkdir -p "${HOME}/bin" "${HOME}/.kube"
touch "${HOME}/.kube/config"

# Install kubectl
curl -sSLo "${KBC_CMD}" "https://storage.googleapis.com/kubernetes-release/release/v${HELM_KUBERNETES_VERSION}/bin/linux/amd64/kubectl"
chmod +x "${KBC_CMD}"
${KBC_CMD} version --client

# Install Helm
curl -o ${HELM_TAR_GZ_FILE} "https://get.helm.sh/${HELM_TAR_GZ_FILE}"
tar -xvf ${HELM_TAR_GZ_FILE}
mv linux-amd64/helm ${HELM_CMD}
chmod +x "${HELM_CMD}"
${HELM_CMD} version --client
rm -fr linux-amd64 ${HELM_TAR_GZ_FILE}
