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

CONFIG_FILE=$SCRIPT_DIR/../elastic_cloud_tf_gcp.yml

if [ ! -e "$CONFIG_FILE" ] ; then
  echo "$CONFIG_FILE does not exist"
  exit 1
fi

CALLBACKS_ENABLED="profile_tasks,dense,minimal,yaml"

if [ "$1" == "destroy" ] ; then
    ANSIBLE_CALLBACKS_ENABLED=$CALLBACKS_ENABLED \
    CLUSTER_CONFIG_FILE=$CONFIG_FILE \
    make -C ansible destroy-cluster
elif [ "$1" == "create" ] ; then
    ANSIBLE_CALLBACKS_ENABLED=$CALLBACKS_ENABLED \
    CLUSTER_CONFIG_FILE=$CONFIG_FILE \
    make -C ansible create-cluster
else
  echo "Unknown action"
  exit 1
fi
