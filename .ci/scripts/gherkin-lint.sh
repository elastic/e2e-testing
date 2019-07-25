#!/usr/bin/env bash
set -eo pipefail

## Iterate for each file without failing fast.
set +e
for file in "$@"; do
  if ! docker run -ti -v "$(pwd)":/src -w /src gherkin/lint ${file} ; then
    echo "ERROR: gherkin-lint failed for the file '${file}'"
    exit_status=1
  fi
done

exit $exit_status
