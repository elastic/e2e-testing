#!/usr/bin/env bash
#
# Given the stack version this script will bump the version.
#
# This script is executed by the automation we are putting in place
# and it requires the git add/commit commands.
#
set -euo pipefail
MSG="parameter missing."
VERSION=${1:?$MSG}

OS=$(uname -s| tr '[:upper:]' '[:lower:]')

if [ "${OS}" == "darwin" ] ; then
	SED="sed -i .bck"
else
	SED="sed -i"
fi

FILES="cli/config/compose/profiles/fleet/docker-compose.yml cli/config/compose/profiles/helm/docker-compose.yml cli/config/compose/profiles/metricbeat/docker-compose.yml cli/config/compose/services/kibana/docker-compose.yml"

echo "Update stack with version ${VERSION}"
for FILE in ${FILES} ; do
	${SED} -E -e "s#(image: docker\.elastic\.co/.*):[0-9]+\.[0-9]+\.[0-9]+(-[a-f0-9]{8})?#\1:${VERSION}#g" $FILE
	## TODO: sed "docker.elastic.co/kibana/kibana:${kibanaTag:-8.0.0-SNAPSHOT}" ??
done

echo "Commit changes"
git config user.email
git checkout -b "update-stack-version-$(date "+%Y%m%d%H%M%S")"
for FILE in ${FILES} ; do
	echo "git add $FILE"
done
git commit -m "bump stack version ${VERSION}"
git --no-pager log -1

echo "You can now push and create a Pull Request"
