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

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Build axia
"$AXIA_PATH"/scripts/build_axia.sh

# Build coreth
"$AXIA_PATH"/scripts/build_coreth.sh

# Exit build successfully if the binaries are created
if [[ -f "$axia_path" && -f "$evm_path" ]]; then
        echo "Build Successful"
        exit 0
else
        echo "Build failure" >&2
        exit 1
fi
