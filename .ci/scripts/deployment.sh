#!/usr/bin/env bash
set -e
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

if [ -d .obs ] ; then
    rm -rf .obs
fi
git clone \
    --branch feature/remove-docker \
    git@github.com:elastic/observability-test-environments.git .obs

cd .obs

export CLUSTER_CONFIG_FILE=tests/environments/elastic_cloud_tf_gcp.yml

if [ $1 == "destroy" ] ; then
    make -C ansible destroy-cluster
else
    make -C ansible create-cluster
fi
