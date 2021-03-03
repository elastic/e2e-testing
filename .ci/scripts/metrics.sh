#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -ex

FILE_BRANCHES='branches.txt'
FILE_PRS='prs.txt'
PREFIX_NUMBER='number_build_'
PREFIX_TRANSFORMED='transformed_'
OUTPUT_FILE='document.json'

collectBuilds() {
    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -not -path "*PR-*" | xargs grep beats-ci-immutable > $FILE_BRANCHES

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -not -path "*PR-*" | wc -l > $PREFIX_NUMBER$FILE_BRANCHES

    echo "Collect builds in the last day for PRs"
    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -path "*PR-*" | xargs grep beats-ci-immutable > $FILE_PRS

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -path "*PR-*" | wc -l > $PREFIX_NUMBER$FILE_PRS
}

transform() {
    sed -e 's#.*is offline##g' \
        -e 's#.*have label.*##g' \
        -e 's#log:.*beats-ci-#log: Running on beats-ci-#g' \
        -e 's#.*Cannot contact.*##g' \
        -e 's#.*Also:.*##g' \
        -e 's#log:.*Running#log: Running#g' \
        -e 's#Running on.*beats-ci#Running on beats-ci#g' \
        -e 's# in ##g' \
        -e 's#.c.elastic.*##g' "$1" | grep 'Running on' > "$2"
}

lookForReusedWorkers() {
    cat "$1" \
        | sort \
        | uniq -c \
        | grep -v '   1' \
        | sed "s#^      ##g" \
        | cut -d' ' -f2 \
        | sort -u \
        | wc -l
}

##### MAIN
echo "Collect builds in the last day for branches/tags with ephemeral workers"
collectBuilds
totalBranch=$(cat $PREFIX_NUMBER$FILE_BRANCHES)
totalPR=$(cat $PREFIX_NUMBER$FILE_PRS)

echo "Gather ephemeral workers for PRs"
transform $FILE_BRANCHES $PREFIX_TRANSFORMED$FILE_BRANCHES
transform $FILE_PRS $PREFIX_TRANSFORMED$FILE_PRS

echo 'Number of builds with reused workers'
reusedWorkersBranch=$(lookForReusedWorkers $PREFIX_TRANSFORMED$FILE_BRANCHES)
reusedWorkersPR=$(lookForReusedWorkers $PREFIX_TRANSFORMED$FILE_PRS)

timestamp=$(date '+%Y-%m-%dT%H:%M:%S%z')
cat > $OUTPUT_FILE <<EOF
{
  "branch": {
    "total": "${totalBranch}",
    "reused": "${reusedWorkersBranch}",
  },
  "prs": {
    "total": "${totalPR}",
    "reused": "${reusedWorkersPR}",
  },
  "timestamp": "${timestamp}",
}
EOF

cat ${OUTPUT_FILE}