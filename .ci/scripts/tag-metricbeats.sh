#!/usr/bin/env bash
set -euxo pipefail

# shellcheck disable=SC2012
DOCKER_IMAGE=$(ls -1 ./*docker.tar.gz)
VERSION=$(echo "${DOCKER_IMAGE}" | sed 's#.*metricbeat-oss-##g; s#-linux.*##g')

# https://docs.docker.com/engine/reference/commandline/import/
# shellcheck disable=SC2002
cat "${DOCKER_IMAGE}" | docker import - docker.elastic.co/beats/metricbeat-oss:"${VERSION}"
docker inspect docker.elastic.co/beats/metricbeat-oss:"${VERSION}"
