// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"github.com/sankar-boro/axia/api"
	"github.com/sankar-boro/axia/pubsub"
	"github.com/sankar-boro/axia/vms/avm/txs"
	"github.com/sankar-boro/axia/vms/components/axc"
)

var _ pubsub.Filterer = &filterer{}

type filterer struct {
	tx *txs.Tx
}

func NewPubSubFilterer(tx *txs.Tx) pubsub.Filterer {
	return &filterer{tx: tx}
}

// Apply the filter on the addresses.
func (f *filterer) Filter(filters []pubsub.Filter) ([]bool, interface{}) {
	resp := make([]bool, len(filters))
	for _, utxo := range f.tx.UTXOs() {
		addressable, ok := utxo.Out.(axc.Addressable)
		if !ok {
			continue
		}

		for _, address := range addressable.Addresses() {
			for i, c := range filters {
				if resp[i] {
					continue
				}
				resp[i] = c.Check(address)
			}
		}
	}
	return resp, api.JSONTxID{
		TxID: f.tx.ID(),
	}
}
