#!/bin/bash

set -euo pipefail


# IMAGE_NAME and TARGET_IMAGE defined in jenkins environment
# CHANGE_BRANCH is set on PR, BRANCH_NAME is used on master builds
CURRENT_BRANCH=${CHANGE_BRANCH:-$BRANCH_NAME}

# push with commit id tag
docker push "$TARGET_IMAGE"
# push with branch name tag
docker tag "$TARGET_IMAGE" "$IMAGE_NAME":"$CURRENT_BRANCH"
docker push "$IMAGE_NAME":"$CURRENT_BRANCH"

# when building master push with latest tag
if [ "$CURRENT_BRANCH" = "master" ]; then
    docker tag "$TARGET_IMAGE" "$IMAGE_NAME":latest
    docker push "$IMAGE_NAME":latest
fi
