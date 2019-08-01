#!/usr/bin/env bash
set -euxo pipefail

LOCATION=${1}

pushd "${LOCATION}"/build/distributions

# shellcheck disable=SC2012
VERSION=$(ls -1 ./*docker.tar.gz | sed 's#.*metricbeat-oss-##g; s#-linux.*##g')

docker tag "docker.elastic.co/beats/metricbeat-oss:${VERSION}" docker.elastic.co/beats/metricbeat-oss:PR
docker inspect docker.elastic.co/beats/metricbeat-oss:PR
