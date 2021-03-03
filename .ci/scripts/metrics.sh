#!/usr/bin/env bash

## Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
## or more contributor license agreements. Licensed under the Elastic License;
## you may not use this file except in compliance with the Elastic License.

set -e

FILE_BRANCHES='branches.txt'
FILE_PRS='prs.txt'
PREFIX_NUMBER='metrics_number_build_'
PREFIX_TRANSFORMED='metrics_transformed_'
OUTPUT_FILE='metrics_document.json'
BULK_REPORT='report-bulk.json'
FOLDER='metrics'

mkdir -p $FOLDER

collectBuilds() {
    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -not -path "*PR-*" | xargs grep beats-ci-immutable > ${FOLDER}/$FILE_BRANCHES

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -not -path "*PR-*" | wc -l > ${FOLDER}/$PREFIX_NUMBER$FILE_BRANCHES

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -path "*PR-*" | xargs grep beats-ci-immutable > ${FOLDER}/$FILE_PRS

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
        -path "*PR-*" | wc -l > ${FOLDER}/$PREFIX_NUMBER$FILE_PRS
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

getValue() {
    folder=$(dirname "$1")
    if [ -e "${folder}/build.xml" ] ; then
      grep "<$2>" "${folder}/build.xml" \
                    | head -n1 \
                    | sed -e "s#.*<$2>##g" -e 's#<.*##g'
    fi
}

##### MAIN
echo "1. Collect builds in the last day for branches/tags with ephemeral workers"
collectBuilds
totalBranch=$(cat ${FOLDER}/$PREFIX_NUMBER$FILE_BRANCHES)
totalPR=$(cat ${FOLDER}/$PREFIX_NUMBER$FILE_PRS)

echo "2. Gather ephemeral workers for PRs"
transform ${FOLDER}/$FILE_BRANCHES ${FOLDER}/$PREFIX_TRANSFORMED$FILE_BRANCHES
transform ${FOLDER}/$FILE_PRS ${FOLDER}/$PREFIX_TRANSFORMED$FILE_PRS

echo '3. Number of builds with reused workers'
reusedWorkersBranch=$(lookForReusedWorkers ${FOLDER}/$PREFIX_TRANSFORMED$FILE_BRANCHES)
reusedWorkersPR=$(lookForReusedWorkers ${FOLDER}/$PREFIX_TRANSFORMED$FILE_PRS)

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

echo "4. Look for each build with an immutable worker"
[ -e "${FOLDER}/${BULK_REPORT}" ] && rm "${FOLDER}/${BULK_REPORT}" || true
find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -1 \
      | xargs grep beats-ci-immutable \
      | cut -d":" -f1 | sort -u \
      | while IFS= read -r line; do
        echo "   processing $line ... "
        result=$(getValue "$line" "result")
        startTime=$(getValue "$line" "startTime")
        if [ $(grep -c "$line" "${FOLDER}/$PREFIX_TRANSFORMED$FILE_BRANCHES") -gt 1 ] ; then
          grep "$line" "${FOLDER}/$PREFIX_TRANSFORMED$FILE_BRANCHES"
          reuse=1
        else
          if [ $(grep -c "$line" "${FOLDER}/$PREFIX_TRANSFORMED$FILE_PRS") -gt 1 ] ; then
            grep "$line" "${FOLDER}/$PREFIX_TRANSFORMED$FILE_PRS"
            reuse=1
          else
            reuse=0
          fi
        fi
        json="{ \"file\": \"$line\", \"result\": \"${result}\", \"startTime\": \"${startTime}\", \"reuseWorker\": \"${reuse}\" }"
        echo "{ \"index\":{} }" >> "${FOLDER}/${BULK_REPORT}"
        echo "${json}" >> "${FOLDER}/${BULK_REPORT}"
      done
