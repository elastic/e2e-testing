#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Build and test the app using the install and test make goals.
#

# Prepare junit build context
mkdir -p $(pwd)/outputs

go get -v -u gotest.tools/gotestsum

# See https://pkg.go.dev/gotest.tools/gotestsum/#readme-junit-xml-output
GOTESTSUM_JUNITFILE="$(pwd)/outputs/TEST-unit-cli.xml" make -C cli install test

# See https://pkg.go.dev/gotest.tools/gotestsum/#readme-junit-xml-output
GOTESTSUM_JUNITFILE="$(pwd)/outputs/TEST-unit-e2e.xml" make -C e2e unit-test
