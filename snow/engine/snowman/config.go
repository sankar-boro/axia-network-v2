// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snowman

import (
	"github.com/sankar-boro/axia/snow"
	"github.com/sankar-boro/axia/snow/consensus/snowball"
	"github.com/sankar-boro/axia/snow/consensus/snowman"
	"github.com/sankar-boro/axia/snow/engine/common"
	"github.com/sankar-boro/axia/snow/engine/snowman/block"
	"github.com/sankar-boro/axia/snow/validators"
)

// Config wraps all the parameters needed for a snowman engine
type Config struct {
	common.AllGetsServer

	Ctx        *snow.ConsensusContext
	VM         block.ChainVM
	Sender     common.Sender
	Validators validators.Set
	Params     snowball.Parameters
	Consensus  snowman.Consensus
}
