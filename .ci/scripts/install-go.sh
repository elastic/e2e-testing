#!/usr/bin/env bash
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
