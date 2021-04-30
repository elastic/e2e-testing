#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Build and test the app using the install and test make goals.
#

readonly VERSION="7.13.0-SNAPSHOT"

main() {
  # refresh docker images
  cat <<EOF >.tmp_images
docker.elastic.co/beats/elastic-agent:${VERSION}
docker.elastic.co/beats/elastic-agent-ubi8:${VERSION}
docker.elastic.co/elasticsearch/elasticsearch:${VERSION}
docker.elastic.co/kibana/kibana:${VERSION}
docker.elastic.co/observability-ci/elastic-agent:${VERSION}
docker.elastic.co/observability-ci/elastic-agent-ubi8:${VERSION}
docker.elastic.co/observability-ci/elasticsearch:${VERSION}
docker.elastic.co/observability-ci/kibana:${VERSION}
EOF

  pull_images "$(cat .tmp_images)"

  rm .tmp_images
}

pull_images() {
  local desired_images="${1}"
  local image_name

  echo "INFO: Starting to pull images."

  while read -r image_name; do
    _remove_image "${image_name}"
    _pull_image "${image_name}"
  done<<<"$desired_images"
}

_pull_image() {
  local image=$1
  printf "\n\e[1;31m Pulling: [..%s..] \e[0m\n" "${image}"
  docker pull "${image}" || true
}

_remove_image() {
  local image=$1
  printf "\n\e[1;31m Removing: [..%s..] \e[0m\n" "${image_name}"
  docker image remove "${image}" || true
}

main "$@"
