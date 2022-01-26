#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -euxo pipefail

#
# Imports public SSH keys from Github profiles
#

BASEDIR=$(dirname "$0")

input="${BASEDIR}/../ansible/github-ssh-keys"
while IFS= read -r line
do
    ssh-import-id "gh:$line"
done < "$input"
