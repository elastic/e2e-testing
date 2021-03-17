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
MTIME=${1:-'0.5'}
mkdir -p $FOLDER

collectBuilds() {
    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -${MTIME} \
        -not -path "*PR-*" | xargs grep beats-ci-immutable > ${FOLDER}/$FILE_BRANCHES

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -${MTIME} \
        -not -path "*PR-*" | wc -l > ${FOLDER}/$PREFIX_NUMBER$FILE_BRANCHES

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -${MTIME} \
        -path "*PR-*" | xargs grep beats-ci-immutable > ${FOLDER}/$FILE_PRS

    find /var/lib/jenkins/jobs \
        -maxdepth 9 \
        -type f \
        -name log \
        -mtime -${MTIME} \
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
        -e 's# in .*##g' \
        -e 's#.*Running on beats-ci-immutable$##g' \
        -e 's#Running on .*beats-ci-#log: Running on beats-ci-#g' \
        -e 's# 4.*##g' \
        -e 's# 5.*##g' \
        -e 's# docker .*##g' \
        -e 's# immutable .*##g' \
        -e 's#\/var.*##2g' \
        -e 's#C:.*##g' \
        -e 's#: java.lang.InterruptedException.*##g' \
        -e 's# to send.*##g' \
        -e 's# was marked offline.*##g' \
        -e 's# . The agent.*##g' \
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

isReused() {
  if cat "$1" \
        | sort \
        | uniq -c \
        | grep -v '   1' \
        | grep -q $2 ; then
    echo 1
  else
    echo 0
  fi
}

isReusedWindows() {
  if cat "$1" \
        | sort \
        | uniq -c \
        | grep -v '   1' \
        | grep 'beats-ci-immutable' \
        | grep 'windows' \
        | grep -q $2 ; then
    echo 1
  else
    echo 0
  fi
}

isReusedLinux() {
  if cat "$1" \
        | sort \
        | uniq -c \
        | grep -v '   1' \
        | grep 'beats-ci-immutable' \
        | grep -v 'windows' \
        | grep -q $2 ; then
    echo 1
  else
    echo 0
  fi
}

getValue() {
    folder=$(dirname "$1")
    if [ -e "${folder}/build.xml" ] ; then
      grep --text "<$2>" "${folder}/build.xml" \
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
cat > ${FOLDER}/$OUTPUT_FILE <<EOF
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
        -mtime -${MTIME} \
      | xargs grep --text beats-ci-immutable \
      | cut -d":" -f1 | sort -u > builds.tmp

while IFS= read -r line; do
  echo "   processing $line ... "
  result=$(getValue "$line" "result")
  access=$(stat "$line" -c "%z")
  timestamp=$(date --date="$access" -Iseconds)
  if echo "$line" | grep -q 'PR-*' ; then
    reuse=$(isReused ${FOLDER}/$PREFIX_TRANSFORMED$FILE_PRS "$line")
    reuseWindows=$(isReusedWindows ${FOLDER}/$PREFIX_TRANSFORMED$FILE_PRS "$line")
    reuseLinux=$(isReusedLinux ${FOLDER}/$PREFIX_TRANSFORMED$FILE_PRS "$line")
  else
    reuse=$(isReused ${FOLDER}/$PREFIX_TRANSFORMED$FILE_BRANCHES "$line")
    reuseWindows=$(isReusedWindows ${FOLDER}/$PREFIX_TRANSFORMED$FILE_BRANCHES "$line")
    reuseLinux=$(isReusedLinux ${FOLDER}/$PREFIX_TRANSFORMED$FILE_BRANCHES "$line")
  fi
  json="{ \"file\": \"$line\", \"result\": \"${result}\", \"startTime\": \"${timestamp}\", \"reuseWorker\": \"${reuse}\", \"reuseLinux\": \"${reuseLinux}\", \"reuseWindows\": \"${reuseWindows}\" }"
  echo "{ \"index\":{} }" >> "${FOLDER}/${BULK_REPORT}"
  echo "${json}" >> "${FOLDER}/${BULK_REPORT}"
done < builds.tmp
