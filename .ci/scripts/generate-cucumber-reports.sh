#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail

## original image: ystia/cucumber-html-reporter:latest
readonly DOCKER_IMAGE="docker.elastic.co/observability-ci/cucumber-html-reporter:latest"

CUCUMBER_REPORTS_PATH="${CUCUMBER_REPORTS_PATH:-""}"
FILES="${CUCUMBER_REPORTS_PATH}/TEST*.json"

main() {
  for f in ${FILES}
  do
    filename="$(basename ${f})"
    echo "parsing ${filename}"
    docker run --rm \
      -v "${CUCUMBER_REPORTS_PATH}:/use/src/app/in" \
      -v "$(pwd)/outputs:/use/src/app/out" \
      -e "CHR_APP_jsonFile=in/${filename}" \
      -e "CHR_APP_output=out/${filename}.html" \
      ${DOCKER_IMAGE}
  done
}

main "$@"
