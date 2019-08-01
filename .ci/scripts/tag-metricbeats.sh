#!/usr/bin/env bash
set -euxo pipefail

# shellcheck disable=SC2012
DOCKER_IMAGE=$(ls -1 ./*docker.tar.gz)

# https://docs.docker.com/engine/reference/commandline/import/
# shellcheck disable=SC2002
cat "${DOCKER_IMAGE}" | docker import - docker.elastic.co/beats/metricbeat-oss:PR
docker inspect docker.elastic.co/beats/metricbeat-oss:PR
