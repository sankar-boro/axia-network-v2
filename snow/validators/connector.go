// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/version"
)

// Connector represents a handler that is called when a connection is marked as
// connected or disconnected
type Connector interface {
	Connected(id ids.NodeID, nodeVersion version.Application) error
	Disconnected(id ids.NodeID) error
}
