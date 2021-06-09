#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail

readonly ELASTIC_REGISTRY="docker.elastic.co"
readonly MANIFEST_TOOL_IMAGE="${ELASTIC_REGISTRY}/infra/manifest-tool:latest"
readonly OBSERVABILITY_CI_REGISTRY="${ELASTIC_REGISTRY}/observability-ci"

main() {
  local image="${1}"

  _push_multiplatform_manifest ${image}
}

_push_multiplatform_manifest() {
  local image="${1}"

  local fqn="${OBSERVABILITY_CI_REGISTRY}/${image}:latest"
  # the '-ARCH' placeholder will be replaced with the values in the '--platforms' argument
  local templateFqn="${OBSERVABILITY_CI_REGISTRY}/${image}-ARCH:latest"

  docker run --rm \
    --mount src=${HOME}/.docker,target=/docker-config,type=bind \
      ${MANIFEST_TOOL_IMAGE} --docker-cfg "/docker-config" \
        push from-args \
          --platforms linux/amd64,linux/arm64 \
          --template ${templateFqn} \
          --target ${fqn}
}

main "$@"
