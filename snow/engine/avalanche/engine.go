// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package axia

import (
	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow/consensus/axia"
	"github.com/sankar-boro/axia/snow/engine/common"
)

// Engine describes the events that can occur on a consensus instance
type Engine interface {
	common.Engine

	// GetVtx returns a vertex by its ID.
	// Returns an error if unknown.
	GetVtx(vtxID ids.ID) (axia.Vertex, error)
}
