// Copyright (C) 2019-2022, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package e2e

import (
	ginkgo "github.com/onsi/ginkgo/v2"
)

// DescribeLocal annotates the tests that requires local network-runner.
// Can only run with local cluster.
func DescribeLocal(text string, body func()) bool {
	return ginkgo.Describe("[Local] "+text, body)
}

// DescribeSwapChain annotates the tests for Swap-Chain.
// Can run with any type of cluster (e.g., local, test, mainnet).
func DescribeSwapChain(text string, body func()) bool {
	return ginkgo.Describe("[Swap-Chain] "+text, body)
}
