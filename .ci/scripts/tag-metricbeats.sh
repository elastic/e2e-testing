#!/usr/bin/env bash
set -euxo pipefail

# shellcheck disable=SC2012
DOCKER_IMAGE=$(ls -1 ./*docker.tar.gz)

docker import "${DOCKER_IMAGE}"
