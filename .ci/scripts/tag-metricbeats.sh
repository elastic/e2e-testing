#!/usr/bin/env bash
set -euxo pipefail
#
# Import and tag the docker image generated previously one.
#
# Requirements:
# - docker image format to be imported is docker.tar.gz and should be within where this script
#   is triggered.
# - docker image format should match metricbeat-oss-X.Y.Z.docker.tar.gz where X.Y.Z is the version.
#

# shellcheck disable=SC2012
DOCKER_IMAGE=$(ls -1 ./*docker.tar.gz)
VERSION=$(echo "${DOCKER_IMAGE}" | sed 's#.*metricbeat-oss-##g; s#-linux.*##g')

# https://docs.docker.com/engine/reference/commandline/import/
# shellcheck disable=SC2002
cat "${DOCKER_IMAGE}" | docker import - docker.elastic.co/beats/metricbeat-oss:"${VERSION}-SNAPSHOT"
docker inspect docker.elastic.co/beats/metricbeat-oss:"${VERSION}-SNAPSHOT"
docker images docker.elastic.co/beats/metricbeat-oss
