// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package axia

import (
	"github.com/sankar-boro/axia/snow"
	"github.com/sankar-boro/axia/snow/consensus/axia"
	"github.com/sankar-boro/axia/snow/engine/axia/vertex"
	"github.com/sankar-boro/axia/snow/engine/common"
	"github.com/sankar-boro/axia/snow/validators"
)

// Config wraps all the parameters needed for an axia engine
type Config struct {
	Ctx *snow.ConsensusContext
	common.AllGetsServer
	VM         vertex.DAGVM
	Manager    vertex.Manager
	Sender     common.Sender
	Validators validators.Set

	Params    axia.Parameters
	Consensus axia.Consensus
}
