#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euo pipefail

BASEDIR=$(dirname "$0")
YQ_IMAGE="mikefarah/yq:4"

PLATFORM="${1:-}"
FILE_ENV=".node-${PLATFORM}-env"
ENV_PREFIX="NODE"

if [ "${PLATFORM}" == "stack" ]; then
  FILE_ENV=".stack-env"
  ENV_PREFIX="STACK"
fi

docker run --rm  -w "/workdir" -v "${PWD}":/workdir ${YQ_IMAGE} ".PLATFORMS | with_entries( select( .key | (. == \"${PLATFORM}\") ) )" .e2e-platforms.yaml > ${PWD}/yml.tmp

SHELL_TYPE=$(docker run --rm -i  -w "/workdir" -v "${PWD}":/workdir ${YQ_IMAGE} ".${PLATFORM}.shell_type" < yml.tmp)
if [ "${SHELL_TYPE}" == "null" ]; then
	SHELL_TYPE="sh"
fi

echo "export ${ENV_PREFIX}_IMAGE="$(docker run --rm -i  -w "/workdir" -v "${PWD}":/workdir ${YQ_IMAGE} ".${PLATFORM}.image" < yml.tmp) > "${FILE_ENV}"
echo "export ${ENV_PREFIX}_INSTANCE_TYPE="$(docker run --rm -i  -w "/workdir" -v "${PWD}":/workdir ${YQ_IMAGE} ".${PLATFORM}.instance_type" < yml.tmp) >> "${FILE_ENV}"
echo "export ${ENV_PREFIX}_LABEL=${PLATFORM}" >> "${FILE_ENV}"
echo "export ${ENV_PREFIX}_SHELL_TYPE="${SHELL_TYPE} >> "${FILE_ENV}"
echo "export ${ENV_PREFIX}_USER="$(docker run --rm -i  -w "/workdir" -v "${PWD}":/workdir ${YQ_IMAGE} ".${PLATFORM}.username" < yml.tmp) >> "${FILE_ENV}"

echo ">>> Please source the generated file in order to use it while running the tests:"
echo "source ${PWD}/${FILE_ENV}"

rm -f ${PWD}/yml.tmp
