#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail

#
# Generate boilerplate code for a test suite
#

SUITE="$1"

export LOWER_SUITE=$(echo ${SUITE} | tr A-Z a-z)
export CAPITAL_SUITE=$(echo ${LOWER_SUITE^})
export STACK_VERSION=$(cat ../.stack-version)

mkdir -p _suites/${LOWER_SUITE}
echo "include ../../commons-test.mk" > _suites/${LOWER_SUITE}/Makefile
envsubst < templates/suite_test.go.tmpl > _suites/${LOWER_SUITE}/${LOWER_SUITE}_test.go

mkdir -p _suites/${LOWER_SUITE}/features
envsubst < templates/suite.feature.tmpl > _suites/${LOWER_SUITE}/features/${LOWER_SUITE}.feature

mkdir -p ../internal/config/compose/profiles/${LOWER_SUITE}
envsubst < templates/profile.yml.tmpl > ../internal/config/compose/profiles/${LOWER_SUITE}/docker-compose.yml
