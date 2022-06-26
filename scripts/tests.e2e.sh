#!/usr/bin/env bash
set -e

# e.g.,
# ./scripts/build.sh
# ./scripts/tests.e2e.sh ./build/axia
# ENABLE_WHITELIST_VTX_TESTS=true ./scripts/tests.e2e.sh ./build/axia
if ! [[ "$0" =~ scripts/tests.e2e.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

AXIA_PATH=$1
if [[ -z "${AXIA_PATH}" ]]; then
  echo "Missing AXIA_PATH argument!"
  echo "Usage: ${0} [AXIA_PATH]" >> /dev/stderr
  exit 255
fi

ENABLE_WHITELIST_VTX_TESTS=${ENABLE_WHITELIST_VTX_TESTS:-false}

#################################
# download axia-network-runner
# https://github.com/ava-labs/avalanche-network-runner
# TODO: migrate to upstream axia-network-runner
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
NETWORK_RUNNER_VERSION=1.0.6
DOWNLOAD_PATH=/tmp/axia-network-runner.tar.gz
DOWNLOAD_URL=https://github.com/ava-labs/avalanche-network-runner/releases/download/v${NETWORK_RUNNER_VERSION}/axia-network-runner_${NETWORK_RUNNER_VERSION}_linux_amd64.tar.gz
if [[ ${GOOS} == "darwin" ]]; then
  DOWNLOAD_URL=https://github.com/ava-labs/avalanche-network-runner/releases/download/v${NETWORK_RUNNER_VERSION}/axia-network-runner_${NETWORK_RUNNER_VERSION}_darwin_amd64.tar.gz
fi

rm -f ${DOWNLOAD_PATH}
rm -f /tmp/axia-network-runner

echo "downloading axia-network-runner ${NETWORK_RUNNER_VERSION} at ${DOWNLOAD_URL}"
curl -L ${DOWNLOAD_URL} -o ${DOWNLOAD_PATH}

echo "extracting downloaded axia-network-runner"
tar xzvf ${DOWNLOAD_PATH} -C /tmp
/tmp/axia-network-runner -h

#################################
echo "building e2e.test"
# to install the ginkgo binary (required for test build and run)
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.0.0
ACK_GINKGO_RC=true ginkgo build ./tests/e2e
./tests/e2e/e2e.test --help

#################################
# run "axia-network-runner" server
echo "launch axia-network-runner in the background"
/tmp/axia-network-runner \
server \
--log-level debug \
--port=":12342" \
--grpc-gateway-port=":12343" 2> /dev/null &
PID=${!}

#################################
# By default, it runs all e2e test cases!
# Use "--ginkgo.skip" to skip tests.
# Use "--ginkgo.focus" to select tests.
#
# to run only ping tests:
# --ginkgo.focus "\[Local\] \[Ping\]"
#
# to run only Swap-Chain whitelist vtx tests:
# --ginkgo.focus "\[Swap-Chain\] \[WhitelistVtx\]"
#
# to skip all "Local" tests
# --ginkgo.skip "\[Local\]"
#
# set "--enable-whitelist-vtx-tests" to explicitly enable/disable whitelist vtx tests
echo "running e2e tests against the local cluster with ${AXIA_PATH}"
./tests/e2e/e2e.test \
--ginkgo.v \
--ginkgo.skip "\[Local\]" \
--log-level debug \
--network-runner-grpc-endpoint="0.0.0.0:12342" \
--axia-log-level=INFO \
--axia-path=${AXIA_PATH} \
--enable-whitelist-vtx-tests=${ENABLE_WHITELIST_VTX_TESTS} || EXIT_CODE=$?

kill ${PID}

if [[ ${EXIT_CODE} -gt 0 ]]; then
  echo "FAILURE with exit code ${EXIT_CODE}"
  exit ${EXIT_CODE}
else
  echo "ALL SUCCESS!"
fi
