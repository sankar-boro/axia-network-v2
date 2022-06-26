// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"github.com/sankar-boro/axia-network-v2/ids"
)

// AllychainTracker describes the interface for checking if a node is tracking a
// allychain
type AllychainTracker interface {
	// TracksAllychain returns true if [nodeID] tracks [allychainID]
	TracksAllychain(nodeID ids.NodeID, allychainID ids.ID) bool
}
