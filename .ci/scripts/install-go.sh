#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the given go version using the gimme script.
#
# Parameters:
#   - GO_VERSION - that's the version which will be installed and enabled.
#

GO_VERSION=${1:?GO_VERSION is not set}

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"
