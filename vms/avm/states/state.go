// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package states

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sankar-boro/axia/database"
	"github.com/sankar-boro/axia/database/prefixdb"
	"github.com/sankar-boro/axia/vms/avm/txs"
	"github.com/sankar-boro/axia/vms/components/axc"
)

var (
	utxoPrefix      = []byte("utxo")
	statusPrefix    = []byte("status")
	singletonPrefix = []byte("singleton")
	txPrefix        = []byte("tx")

	_ State = &state{}
)

// State persistently maintains a set of UTXOs, transaction, statuses, and
// singletons.
type State interface {
	axc.UTXOState
	axc.StatusState
	axc.SingletonState
	TxState
}

type state struct {
	axc.UTXOState
	axc.StatusState
	axc.SingletonState
	TxState
}

func New(db database.Database, parser txs.Parser, metrics prometheus.Registerer) (State, error) {
	utxoDB := prefixdb.New(utxoPrefix, db)
	statusDB := prefixdb.New(statusPrefix, db)
	singletonDB := prefixdb.New(singletonPrefix, db)
	txDB := prefixdb.New(txPrefix, db)

	utxoState, err := axc.NewMeteredUTXOState(utxoDB, parser.Codec(), metrics)
	if err != nil {
		return nil, err
	}

	statusState, err := axc.NewMeteredStatusState(statusDB, metrics)
	if err != nil {
		return nil, err
	}

	txState, err := NewTxState(txDB, parser, metrics)
	return &state{
		UTXOState:      utxoState,
		StatusState:    statusState,
		SingletonState: axc.NewSingletonState(singletonDB),
		TxState:        txState,
	}, err
}
