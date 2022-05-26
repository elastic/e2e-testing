#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail

#
# Install python using pyenv manager
#

PYENV_ROOT="${HOME}/.pyenv"
PYTHON_VERSION=${PYTHON_VERSION:-"3.9.12"}

export PATH="${PATH}:${PYENV_ROOT}/bin"

[[ -d "${PYENV_ROOT}" ]] || curl https://pyenv.run | bash

pyenv install "${PYTHON_VERSION}" -f
pyenv global "${PYTHON_VERSION}"
