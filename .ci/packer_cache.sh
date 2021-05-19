#!/usr/bin/env bash

# shellcheck disable=SC1091
source /usr/local/bin/bash_standard_lib.sh

readonly GO_VERSION=$(cat .go-version)

DOCKER_IMAGES="docker.elastic.co/observability-ci/centos-systemd:latest
docker.elastic.co/observability-ci/debian-systemd:latest
docker.elastic.co/beats/filebeat:7.13.0-SNAPSHOT
docker.elastic.co/observability-ci/picklesdoc:2.20.1
golang:${GO_VERSION}-stretch
"

if [ -x "$(command -v docker)" ]; then
  for di in ${DOCKER_IMAGES}
  do
  (retry 2 docker pull "${di}") || echo "Error pulling ${di} Docker image, we continue"
  done
fi
