#!/usr/bin/env bash
set -e
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
FOLDER=.obs
if [ -d $FOLDER ] ; then
    rm -rf $FOLDER
fi
git clone \
    --branch feature/remove-docker \
    git@github.com:elastic/observability-test-environments.git $FOLDER

cd $FOLDER

CLUSTER_CONFIG_FILE=$SCRIPT_DIR/$FOLDER/tests/environments/elastic_cloud_tf_gcp.yml
export CLUSTER_CONFIG_FILE

if [ $1 == "destroy" ] ; then
    make -C ansible destroy-cluster
else
    make -C ansible create-cluster
fi
