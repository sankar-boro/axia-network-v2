#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Axia root folder
AXIA_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )
# Load the versions
source "$AXIA_PATH"/scripts/versions.sh
# Load the constants
source "$AXIA_PATH"/scripts/constants.sh

# build_image_from_remote.sh is deprecated
source "$AXIA_PATH"/scripts/build_local_image.sh
