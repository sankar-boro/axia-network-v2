// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/pubsub"
	"github.com/sankar-boro/axia/vms/avm/txs"
	"github.com/sankar-boro/axia/vms/components/axc"
	"github.com/sankar-boro/axia/vms/secp256k1fx"
)

type mockFilter struct {
	addr []byte
}

func (f *mockFilter) Check(addr []byte) bool {
	return bytes.Equal(addr, f.addr)
}

func TestFilter(t *testing.T) {
	assert := assert.New(t)

	addrID := ids.ShortID{1}
	tx := txs.Tx{UnsignedTx: &txs.BaseTx{BaseTx: axc.BaseTx{
		Outs: []*axc.TransferableOutput{
			{
				Out: &secp256k1fx.TransferOutput{
					OutputOwners: secp256k1fx.OutputOwners{
						Addrs: []ids.ShortID{addrID},
					},
				},
			},
		},
	}}}
	addrBytes := addrID[:]

	fp := pubsub.NewFilterParam()
	err := fp.Add(addrBytes)
	assert.NoError(err)

	parser := NewPubSubFilterer(&tx)
	fr, _ := parser.Filter([]pubsub.Filter{&mockFilter{addr: addrBytes}})
	assert.Equal([]bool{true}, fr)
}
