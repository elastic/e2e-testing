#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Install the dependencies using the install and test make goals.
#
# Parameters:
#   - GOOS - that's the name of the O.S. used to build the binary
#   - GOARCH - that's the name of the architecture of the O.S. used to build the binary.
#

TARGET_OS=${GOOS:-linux}
TARGET_ARCH=${GOARCH:-amd64}

# Build OP Binary
GOOS=${TARGET_OS} GOARCH=${TARGET_ARCH} make -C e2e fetch-binary

# Sync integrations
make -C e2e sync-integrations
