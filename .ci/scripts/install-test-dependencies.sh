#!/usr/bin/env bash
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

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh "${GO_VERSION}"

MSG="parameter missing."
HOME=${HOME:?$MSG}

HELM_VERSION="${HELM_VERSION:-"2.16.3"}"
HELM_TAR_GZ_FILE="helm-v${HELM_VERSION}-linux-amd64.tar.gz"
KIND_VERSION="v${KIND_VERSION:-"0.7.0"}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-"1.15.3"}"

HELM_CMD="${HOME}/bin/helm"
KBC_CMD="${HOME}/bin/kubectl"

# Install kind as a Go binary
GO111MODULE="on" go get sigs.k8s.io/kind@${KIND_VERSION}

mkdir -p "${HOME}/bin" "${HOME}/.kube"
touch "${HOME}/.kube/config"

# Install kubectl
curl -sSLo "${KBC_CMD}" "https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VERSION}/bin/linux/amd64/kubectl"
chmod +x "${KBC_CMD}"
${KBC_CMD} version --client

# Install Helm
curl -o ${HELM_TAR_GZ_FILE} "https://get.helm.sh/${HELM_TAR_GZ_FILE}"
tar -xvf ${HELM_TAR_GZ_FILE}
mv linux-amd64/helm ${HELM_CMD}
chmod +x "${HELM_CMD}"
${HELM_CMD} version --client
rm -fr linux-amd64 ${HELM_TAR_GZ_FILE}
