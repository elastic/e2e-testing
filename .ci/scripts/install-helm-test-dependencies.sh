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
#   - KIND_VERSION - that's the Kind version which will be installed and enabled.
#   - KUBERNETES_VERSION - that's the Kubernetes version which will be installed and enabled.
#

GOARCH=${GOARCH:-"amd64"}
MSG="parameter missing."
HOME=${HOME:?$MSG}

HELM_VERSION="${HELM_VERSION:-"3.9.0"}"
HELM_TAR_GZ_FILE="helm-v${HELM_VERSION}-linux-${GOARCH}.tar.gz"
KIND_VERSION="v${KIND_VERSION:-"0.20.0"}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-"1.27.3"}"

HELM_CMD="${HOME}/bin/helm"
KBC_CMD="${HOME}/bin/kubectl"

# Install kind as a Go binary
GOARCH=${GOARCH} GO111MODULE="on" go get sigs.k8s.io/kind@${KIND_VERSION}

mkdir -p "${HOME}/bin" "${HOME}/.kube"
touch "${HOME}/.kube/config"

# Install kubectl
curl -sSLo "${KBC_CMD}" "https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VERSION}/bin/linux/${GOARCH}/kubectl"
chmod +x "${KBC_CMD}"
${KBC_CMD} version --client

# Install Helm
curl -o ${HELM_TAR_GZ_FILE} "https://get.helm.sh/${HELM_TAR_GZ_FILE}"
tar -xvf ${HELM_TAR_GZ_FILE}
mv linux-${GOARCH}/helm ${HELM_CMD}
chmod +x "${HELM_CMD}"
${HELM_CMD} version --client
rm -fr linux-${GOARCH} ${HELM_TAR_GZ_FILE}
