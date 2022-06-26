#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Directory above this script
AXIA_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

# Load the versions
source "$AXIA_PATH"/scripts/versions.sh

# Load the constants
source "$AXIA_PATH"/scripts/constants.sh

# WARNING: this will use the most recent commit even if there are un-committed changes present
full_commit_hash="$(git --git-dir="$AXIA_PATH/.git" rev-parse HEAD)"
commit_hash="${full_commit_hash::8}"

echo "Building Docker Image with tags: $axia_dockerhub_repo:$commit_hash , $axia_dockerhub_repo:$current_branch"
docker build -t "$axia_dockerhub_repo:$commit_hash" \
        -t "$axia_dockerhub_repo:$current_branch" "$AXIA_PATH" -f "$AXIA_PATH/Dockerfile"
