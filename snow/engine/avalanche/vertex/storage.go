// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vertex

import (
	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow/consensus/axia"
)

// Storage defines the persistent storage that is required by the consensus
// engine.
type Storage interface {
	// Get a vertex by its hash from storage.
	GetVtx(vtxID ids.ID) (axia.Vertex, error)
	// Edge returns a list of accepted vertex IDs with no accepted children.
	Edge() (vtxIDs []ids.ID)
}
