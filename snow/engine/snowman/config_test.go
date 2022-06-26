// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snowman

import (
	"github.com/sankar-boro/avalanchego/snow/consensus/snowball"
	"github.com/sankar-boro/avalanchego/snow/consensus/snowman"
	"github.com/sankar-boro/avalanchego/snow/engine/common"
	"github.com/sankar-boro/avalanchego/snow/engine/snowman/block"
)

func DefaultConfigs() Config {
	commonCfg := common.DefaultConfigTest()
	return Config{
		Ctx:        commonCfg.Ctx,
		Sender:     commonCfg.Sender,
		Validators: commonCfg.Validators,
		VM:         &block.TestVM{},
		Params: snowball.Parameters{
			K:                       1,
			Alpha:                   1,
			BetaVirtuous:            1,
			BetaRogue:               2,
			ConcurrentRepolls:       1,
			OptimalProcessing:       100,
			MaxOutstandingItems:     1,
			MaxItemProcessingTime:   1,
			MixedQueryNumPushNonVdr: 1,
		},
		Consensus: &snowman.Topological{},
	}
}
