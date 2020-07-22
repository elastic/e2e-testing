#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail
#
# Build and test the app using the install and test make goals.
#

# Prepare junit build context
mkdir -p outputs
export OUT_FILE="outputs/test-report.out"

make -C cli install test | tee ${OUT_FILE}

go get -v -u github.com/jstemmer/go-junit-report
go-junit-report > outputs/TEST-unit.xml < ${OUT_FILE}
