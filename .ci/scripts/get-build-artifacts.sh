#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -exuo pipefail

if [ $# -ne 3 ]; then
  echo "${0} PRODUCT VERSION COMMIT"
  exit 1
fi

PRODUCT=${1}
VERSION_ID=${2}
COMMIT_ID=${3}

URL="https://staging.elastic.co/${VERSION_ID}-${COMMIT_ID}/docker/${PRODUCT}-${VERSION_ID}-docker-image.tar.gz"

if [[ "$PRODUCT" == *beat ]]; then
    URL="https://staging.elastic.co/${VERSION_ID}-${COMMIT_ID}/docker/${PRODUCT}-${VERSION_ID}-docker-image-linux-amd64.tar.gz"
fi

echo "Downloading ${PRODUCT} - ${URL}"
curl "${URL}"|docker load
