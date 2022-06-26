#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Testing specific variables
axia_testing_repo="avaplatform/axia-testing"
axia_byzantine_repo="avaplatform/axia-byzantine"

# Define axia-testing and axia-byzantine versions to use
axia_testing_image="avaplatform/axia-testing:master"
axia_byzantine_image="avaplatform/axia-byzantine:update-axia-v1.7.0"

# Fetch the images
# If Docker Credentials are not available fail
if [[ -z ${DOCKER_USERNAME} ]]; then
    echo "Skipping Tests because Docker Credentials were not present."
    exit 1
fi

# Axia root directory
AXIA_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd ../.. && pwd )

# Load the versions
source "$AXIA_PATH"/scripts/versions.sh

# Load the constants
source "$AXIA_PATH"/scripts/constants.sh

# Login to docker
echo "$DOCKER_PASS" | docker login --username "$DOCKER_USERNAME" --password-stdin

# Receives params for debug execution
testBatch="${1:-}"
shift 1

echo "Running Test Batch: ${testBatch}"

# pulling the axia-testing image
docker pull $axia_testing_image
docker pull $axia_byzantine_image

# Setting the build ID
git_commit_id=$( git rev-list -1 HEAD )

# Build current axia
source "$AXIA_PATH"/scripts/build_image.sh

# Target built version to use in axia-testing
axia_image="$axia_dockerhub_repo:$current_branch"

echo "Execution Summary:"
echo ""
echo "Running Axia Image: ${axia_image}"
echo "Running Axia Image Tag: $current_branch"
echo "Running Axia Testing Image: ${axia_testing_image}"
echo "Running Axia Byzantine Image: ${axia_byzantine_image}"
echo "Git Commit ID : ${git_commit_id}"
echo ""

# >>>>>>>> axia-testing custom parameters <<<<<<<<<<<<<
custom_params_json="{
    \"isKurtosisCoreDevMode\": false,
    \"axiaImage\":\"${axia_image}\",
    \"axiaByzantineImage\":\"${axia_byzantine_image}\",
    \"testBatch\":\"${testBatch}\"
}"
# >>>>>>>> axia-testing custom parameters <<<<<<<<<<<<<

bash "$AXIA_PATH/.kurtosis/kurtosis.sh" \
    --custom-params "${custom_params_json}" \
    ${1+"${@}"} \
    "${axia_testing_image}" 
