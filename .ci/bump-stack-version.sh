#!/usr/bin/env bash
#
# Given the stack version this script will bump the version.
#
# This script is executed by the automation we are putting in place
# and it requires the git add/commit commands.
#
# Parameters:
#	$1 -> the version to be bumped. Mandatory.
#	$2 -> whether to create a branch where to commit the changes to.
#		  this is required when reusing an existing Pull Request.
#		  Optional. Default true.
#
set -euo pipefail
MSG="parameter missing."
VERSION=${1:?$MSG}
CREATE_BRANCH=${2:-true}

OS=$(uname -s| tr '[:upper:]' '[:lower:]')

if [ "${OS}" == "darwin" ] ; then
	SED="sed -i .bck"
else
	SED="sed -i"
fi

echo "Update stack with version ${VERSION} in docker-compose.yml"
find . -name 'docker-compose.yml' -path './cli/config/compose/profiles/*' -print0 |
	while IFS= read -r -d '' FILE ; do
		${SED} -E -e "s#(image: (\")?docker\.elastic\.co/.*):-[0-9]+\.[0-9]+\.[0-9]+(-[a-f0-9]{8})?#\1:-${VERSION}#g" $FILE
		git add $FILE
	done

echo "Update stack with version ${VERSION} in deployment.yaml"
find . -name 'deployment.yaml' -print0 |
	while IFS= read -r -d '' FILE ; do
		${SED} -E -e "s#(image: docker\.elastic\.co/.*):[0-9]+\.[0-9]+\.[0-9]+(-[a-f0-9]{8})?#\1:${VERSION}#g" $FILE
		git add $FILE
	done

echo "Commit changes"
if [ "$CREATE_BRANCH" = "true" ]; then
	git checkout -b "update-stack-version-$(date "+%Y%m%d%H%M%S")"
else
	echo "Branch creation disabled."
fi
git diff --staged --quiet || git commit -m "bump stack version ${VERSION}"
git --no-pager log -1

echo "You can now push and create a Pull Request"
