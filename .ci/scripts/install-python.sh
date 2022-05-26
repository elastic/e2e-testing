#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail

#
# Install python using pyenv manager
#

curl https://pyenv.run | bash

export PATH="${PATH}:${HOME}/.pyenv/bin"

pyenv install "${PYTHON_VERSION}"
pyenv global "${PYTHON_VERSION}"
