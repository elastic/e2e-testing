#!/usr/bin/env bash
set -euxo pipefail
#
# Install the dependencies using in tests.
#

# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
chmod +x ./kubectl
mv ./kubectl /usr/local/bin/kubectl
kubectl version --client

# install minikube
curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
  && chmod +x minikube
mv minikube /usr/local/bin

# install helm
curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
