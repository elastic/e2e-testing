#!/usr/bin/env bash
set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#
# Parameters:
#   - HELM_VERSION - that's the Helm version which will be installed and enabled.
#   - KIND_VERSION - that's the Kind version which will be installed and enabled.
#   - K8S_VERSION - that's the Kubernetes version which will be installed and enabled.
#

MSG="parameter missing."
HOME=${HOME:?$MSG}

HELM_VERSION="v2.16.3"
HELM_TAR_GZ_FILE="helm-${HELM_VERSION}-linux-amd64.tar.gz"
KIND_VERSION="v0.7.0"
K8S_VERSION="v1.15.3"

HELM_CMD="${HOME}/bin/helm"
KBC_CMD="${HOME}/bin/kubectl"

# Install kind as a Go binary
GO111MODULE="on" go get sigs.k8s.io/kind@${KIND_VERSION}

mkdir -p "${HOME}/bin" "${HOME}/.kube"
touch "${HOME}/.kube/config"

# Install kubectl
curl -sSLo "${KBC_CMD}" "https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl"
chmod +x "${KBC_CMD}"
${KBC_CMD} version || true

# Install Helm
curl -o ${HELM_TAR_GZ_FILE} "https://get.helm.sh/${HELM_TAR_GZ_FILE}"
tar -xvf ${HELM_TAR_GZ_FILE}
mv linux-amd64/helm ${HELM_CMD}
chmod +x "${HELM_CMD}"
${HELM_CMD} version || true
rm -fr linux-amd64 ${HELM_TAR_GZ_FILE}
