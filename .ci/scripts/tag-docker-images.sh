#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail

ARCH="${ARCH:-amd64}"

readonly ELASTIC_REGISTRY="docker.elastic.co"
readonly OBSERVABILITY_CI_REGISTRY="${ELASTIC_REGISTRY}/observability-ci"

main() {
  local image="${1}"
  local tag="${2}"

  _tag_and_push ${image}
}

_tag_and_push() {
  local image="${1}"
  local tag="${2}"

  local dockerImage="${OBSERVABILITY_CI_REGISTRY}/${image}:${tag}"
  local platformSpecificImage="${OBSERVABILITY_CI_REGISTRY}/${image}-${ARCH}:${tag}"

  docker tag ${dockerImage} ${platformSpecificImage}

  docker push ${platformSpecificImage}
}

main "$@"
