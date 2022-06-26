// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package verify

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow"
)

var errMissing = errors.New("missing")

type snLookup struct {
	chainsToAllychain map[ids.ID]ids.ID
}

func (sn *snLookup) AllychainID(chainID ids.ID) (ids.ID, error) {
	allychainID, ok := sn.chainsToAllychain[chainID]
	if !ok {
		return ids.ID{}, errMissing
	}
	return allychainID, nil
}

func TestSameAllychain(t *testing.T) {
	allychain0 := ids.GenerateTestID()
	allychain1 := ids.GenerateTestID()
	chain0 := ids.GenerateTestID()
	chain1 := ids.GenerateTestID()

	tests := []struct {
		name    string
		ctx     *snow.Context
		chainID ids.ID
		result  error
	}{
		{
			name: "same chain",
			ctx: &snow.Context{
				AllychainID: allychain0,
				ChainID:  chain0,
				SNLookup: &snLookup{},
			},
			chainID: chain0,
			result:  errSameChainID,
		},
		{
			name: "unknown chain",
			ctx: &snow.Context{
				AllychainID: allychain0,
				ChainID:  chain0,
				SNLookup: &snLookup{},
			},
			chainID: chain1,
			result:  errMissing,
		},
		{
			name: "wrong allychain",
			ctx: &snow.Context{
				AllychainID: allychain0,
				ChainID:  chain0,
				SNLookup: &snLookup{
					chainsToAllychain: map[ids.ID]ids.ID{
						chain1: allychain1,
					},
				},
			},
			chainID: chain1,
			result:  errMismatchedAllychainIDs,
		},
		{
			name: "same allychain",
			ctx: &snow.Context{
				AllychainID: allychain0,
				ChainID:  chain0,
				SNLookup: &snLookup{
					chainsToAllychain: map[ids.ID]ids.ID{
						chain1: allychain0,
					},
				},
			},
			chainID: chain1,
			result:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SameAllychain(test.ctx, test.chainID)
			assert.ErrorIs(t, result, test.result)
		})
	}
}
