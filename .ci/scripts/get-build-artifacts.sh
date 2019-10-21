#!/usr/bin/env bash
# Licensed to Elasticsearch B.V. under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Elasticsearch B.V. licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
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
