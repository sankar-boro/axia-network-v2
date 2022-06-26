#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Note: this script will build a docker image by cloning a remote version of
# axia into a temporary location and using that version's Dockerfile to
# build the image.

SRC_DIR="$(dirname "${BASH_SOURCE[0]}")"

DOCKERHUB_REPO="avaplatform/axia"
REMOTE="https://github.com/sankar-boro/axia.git"
BRANCH="master"

if [[ $# -eq 2 ]]; then
    REMOTE=$1
    BRANCH=$2
else
    echo "Using default remote: $REMOTE and branch: $BRANCH to build image"
fi


echo "Building docker image from branch: $BRANCH at remote: $REMOTE"

export GOPATH="$SRC_DIR/.build_image_gopath"
WORKPREFIX="$GOPATH/src/github.com/ava-labs"
DOCKER="${DOCKER:-docker}"
keep_existing=0
while getopts 'k' opt
do
    case $opt in
    (k) keep_existing=1;;
    esac
done
if [[ "$keep_existing" != 1 ]]; then
    rm -rf "$WORKPREFIX"
fi

# Clone the remote and checkout the specified branch to build the Docker image
AXIA_CLONE="$WORKPREFIX/axia"

if [[ ! -d "$WORKPREFIX" ]]; then
    mkdir -p "$WORKPREFIX"
    git config --global credential.helper cache
    git clone "$REMOTE" "$AXIA_CLONE"
    git --git-dir="$AXIA_CLONE/.git" checkout "$BRANCH"
fi

FULL_COMMIT_HASH="$(git --git-dir="$AXIA_CLONE/.git" rev-parse HEAD)"
AXIA_COMMIT="${FULL_COMMIT_HASH::8}"

"${DOCKER}" build -t "$DOCKERHUB_REPO:$AXIA_COMMIT" "$AXIA_CLONE" -f "$AXIA_CLONE/Dockerfile"
