#!/usr/bin/env bash
set -e

# e.g.,
# ./scripts/build.sh
# ./scripts/tests.upgrade.sh 1.7.4 ./build/axia
if ! [[ "$0" =~ scripts/tests.upgrade.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

VERSION=$1
if [[ -z "${VERSION}" ]]; then
  echo "Missing version argument!"
  echo "Usage: ${0} [VERSION] [NEW-BINARY]" >> /dev/stderr
  exit 255
fi

NEW_BINARY=$2
if [[ -z "${NEW_BINARY}" ]]; then
  echo "Missing new binary path argument!"
  echo "Usage: ${0} [VERSION] [NEW-BINARY]" >> /dev/stderr
  exit 255
fi

#################################
# download axia
# https://github.com/sankar-boro/axia/releases
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
DOWNLOAD_URL=https://github.com/sankar-boro/axia/releases/download/v${VERSION}/axia-linux-${GOARCH}-v${VERSION}.tar.gz
DOWNLOAD_PATH=/tmp/axia.tar.gz
if [[ ${GOOS} == "darwin" ]]; then
  DOWNLOAD_URL=https://github.com/sankar-boro/axia/releases/download/v${VERSION}/axia-macos-v${VERSION}.zip
  DOWNLOAD_PATH=/tmp/axia.zip
fi

rm -f ${DOWNLOAD_PATH}
rm -rf /tmp/axia-v${VERSION}
rm -rf /tmp/axia-build

echo "downloading axia ${VERSION} at ${DOWNLOAD_URL}"
curl -L ${DOWNLOAD_URL} -o ${DOWNLOAD_PATH}

echo "extracting downloaded axia"
if [[ ${GOOS} == "linux" ]]; then
  tar xzvf ${DOWNLOAD_PATH} -C /tmp
elif [[ ${GOOS} == "darwin" ]]; then
  unzip ${DOWNLOAD_PATH} -d /tmp/axia-build
  mv /tmp/axia-build/build /tmp/axia-v${VERSION}
fi
find /tmp/axia-v${VERSION}

#################################
# download axia-network-runner
# https://github.com/sankar-boro/axia-network-runner
# TODO: migrate to upstream axia-network-runner
NETWORK_RUNNER_VERSION=1.0.6
DOWNLOAD_PATH=/tmp/axia-network-runner.tar.gz
DOWNLOAD_URL=https://github.com/sankar-boro/axia-network-runner/releases/download/v${NETWORK_RUNNER_VERSION}/axia-network-runner_${NETWORK_RUNNER_VERSION}_linux_amd64.tar.gz
if [[ ${GOOS} == "darwin" ]]; then
  DOWNLOAD_URL=https://github.com/sankar-boro/axia-network-runner/releases/download/v${NETWORK_RUNNER_VERSION}/axia-network-runner_${NETWORK_RUNNER_VERSION}_darwin_amd64.tar.gz
fi

rm -f ${DOWNLOAD_PATH}
rm -f /tmp/axia-network-runner

echo "downloading axia-network-runner ${NETWORK_RUNNER_VERSION} at ${DOWNLOAD_URL}"
curl -L ${DOWNLOAD_URL} -o ${DOWNLOAD_PATH}

echo "extracting downloaded axia-network-runner"
tar xzvf ${DOWNLOAD_PATH} -C /tmp
/tmp/axia-network-runner -h

#################################
echo "building upgrade.test"
# to install the ginkgo binary (required for test build and run)
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.0.0
ACK_GINKGO_RC=true ginkgo build ./tests/upgrade
./tests/upgrade/upgrade.test --help

#################################
# run "axia-network-runner" server
echo "launch axia-network-runner in the background"
/tmp/axia-network-runner \
server \
--log-level debug \
--port=":12340" \
--grpc-gateway-port=":12341" 2> /dev/null &
PID=${!}

#################################
# By default, it runs all upgrade test cases!
echo "running upgrade tests against the local cluster with ${NEW_BINARY}"
./tests/upgrade/upgrade.test \
--ginkgo.v \
--log-level debug \
--network-runner-grpc-endpoint="0.0.0.0:12340" \
--axia-path=/tmp/axia-v${VERSION}/axia \
--axia-path-to-upgrade=${NEW_BINARY} || EXIT_CODE=$?

kill ${PID}

if [[ ${EXIT_CODE} -gt 0 ]]; then
  echo "FAILURE with exit code ${EXIT_CODE}"
  exit ${EXIT_CODE}
else
  echo "ALL SUCCESS!"
fi
