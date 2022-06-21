#!/usr/bin/env bash
set -e

FOLDER=.obs
if [ -d $FOLDER ] ; then
    rm -rf $FOLDER
fi
git clone \
    git@github.com:elastic/observability-test-environments.git $FOLDER

cd $FOLDER

CONFIG_FILE=$(pwd)/tests/environments/elastic_cloud_tf_gcp.yml

if [ ! -e "$CONFIG_FILE" ] ; then
  echo "$CONFIG_FILE does not exist"
  exit 1
fi

if [ "$1" == "destroy" ] ; then
  CLUSTER_CONFIG_FILE=$CONFIG_FILE \
  make -C ansible destroy-cluster
elif [ "$1" == "create" ] ; then
  CLUSTER_CONFIG_FILE=$CONFIG_FILE \
  make -C ansible create-cluster
else
  echo "Unknown action"
  exit 1
fi
