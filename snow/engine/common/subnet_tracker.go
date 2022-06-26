// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"github.com/sankar-boro/axia/ids"
)

// SubnetTracker describes the interface for checking if a node is tracking a
// subnet
type SubnetTracker interface {
	// TracksSubnet returns true if [nodeID] tracks [subnetID]
	TracksSubnet(nodeID ids.NodeID, subnetID ids.ID) bool
}
