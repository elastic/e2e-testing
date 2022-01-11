#!/usr/bin/env bash
#
# Given the current and next branches, in major.minor format, this script will update
# those versions.
#
# Parameters:
#	$1 -> the version to be searched. Mandatory.
#	$2 -> the version to be updated. Mandatory.
#	$3 -> whether to create a branch where to commit the changes to.
#		  this is required when reusing an existing Pull Request.
#		  Optional. Default true.
#
set -euo pipefail
MSG="parameter missing."
CURRENT=${1:?$MSG}
NEXT=${2:?$MSG}
CREATE_BRANCH=${3:-true}

OS=$(uname -s| tr '[:upper:]' '[:lower:]')

if [ "${OS}" == "darwin" ] ; then
	SED="sed -i .bck"
else
	SED="sed -i"
fi

echo "Update version from ${CURRENT} to ${NEXT}"
for FILE in .backportrc.json .mergify.yml
do
  echo "* Processing $FILE"
  ${SED} -E -e "s#${CURRENT}#${NEXT}#g" $FILE
  git add $FILE
done

## JJBB specific
for FILE in .ci/jobs/*.yml
do
  echo "* Processing JJBB $FILE"
  ESCAPED_CURRENT=$(echo $CURRENT | sed "s#\\.#\\\\\\\\.#g")
  ESCAPED_NEXT=$(echo $NEXT | sed "s#\\.#\\\\\\\\.#g")
  ${SED} -E -e "s#${ESCAPED_CURRENT}#${ESCAPED_NEXT}#g" $FILE
  git add $FILE
done

echo "Commit changes"
if [ "$CREATE_BRANCH" = "true" ]; then
	git checkout -b "update-branch-version-${NEXT}-$(date "+%Y%m%d%H%M%S")"
else
	echo "Branch creation disabled."
fi
git diff --staged --quiet || git commit -m "bump branch version ${VERSION}"
git --no-pager log -1

echo "You can now push and create a Pull Request"
