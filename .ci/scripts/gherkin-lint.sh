#!/usr/bin/env bash
set -eo pipefail

IMAGE="gherkin/lint"
docker pull --quiet "${IMAGE}"

## Iterate for each file without failing fast.
set +e
for file in "$@"; do
  if ! docker run -t -v "$(pwd)":/src -w /src "${IMAGE}" ${file} ; then
    echo "ERROR: gherkin-lint failed for the file '${file}'"
    exit_status=1
  fi
done

exit $exit_status
