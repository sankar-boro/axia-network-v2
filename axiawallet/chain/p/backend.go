// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	"fmt"
	"sync"

	stdcontext "context"

	"github.com/sankar-boro/axia-network-v2/database"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm"
)

var _ Backend = &backend{}

type ChainUTXOs interface {
	AddUTXO(ctx stdcontext.Context, destinationChainID ids.ID, utxo *axc.UTXO) error
	RemoveUTXO(ctx stdcontext.Context, sourceChainID, utxoID ids.ID) error

	UTXOs(ctx stdcontext.Context, sourceChainID ids.ID) ([]*axc.UTXO, error)
	GetUTXO(ctx stdcontext.Context, sourceChainID, utxoID ids.ID) (*axc.UTXO, error)
}

// Backend defines the full interface required to support a Core-chain axiawallet.
type Backend interface {
	ChainUTXOs
	BuilderBackend
	SignerBackend

	AcceptTx(ctx stdcontext.Context, tx *platformvm.Tx) error
}

type backend struct {
	Context
	ChainUTXOs

	txsLock sync.RWMutex
	// txID -> tx
	txs map[ids.ID]*platformvm.Tx
}

func NewBackend(ctx Context, utxos ChainUTXOs, txs map[ids.ID]*platformvm.Tx) Backend {
	return &backend{
		Context:    ctx,
		ChainUTXOs: utxos,
		txs:        txs,
	}
}

func (b *backend) AcceptTx(ctx stdcontext.Context, tx *platformvm.Tx) error {
	var baseTx *platformvm.BaseTx
	txID := tx.ID()
	switch utx := tx.UnsignedTx.(type) {
	case *platformvm.UnsignedAddNominatorTx:
		baseTx = &utx.BaseTx
	case *platformvm.UnsignedAddSubnetValidatorTx:
		baseTx = &utx.BaseTx
	case *platformvm.UnsignedAddValidatorTx:
		baseTx = &utx.BaseTx
	case *platformvm.UnsignedExportTx:
		baseTx = &utx.BaseTx

		for i, out := range utx.ExportedOutputs {
			err := b.AddUTXO(
				ctx,
				utx.DestinationChain,
				&axc.UTXO{
					UTXOID: axc.UTXOID{
						TxID:        txID,
						OutputIndex: uint32(len(utx.Outs) + i),
					},
					Asset: axc.Asset{ID: out.AssetID()},
					Out:   out.Out,
				},
			)
			if err != nil {
				return err
			}
		}
	case *platformvm.UnsignedImportTx:
		baseTx = &utx.BaseTx

		consumedRemoteUTXOIDs := utx.InputUTXOs()
		err := b.removeUTXOs(ctx, utx.SourceChain, consumedRemoteUTXOIDs)
		if err != nil {
			return err
		}
	case *platformvm.UnsignedCreateChainTx:
		baseTx = &utx.BaseTx
	case *platformvm.UnsignedCreateSubnetTx:
		baseTx = &utx.BaseTx
	default:
		return fmt.Errorf("%w: %T", errUnknownTxType, tx.UnsignedTx)
	}

	consumedUTXOIDs := baseTx.InputIDs()
	err := b.removeUTXOs(ctx, constants.PlatformChainID, consumedUTXOIDs)
	if err != nil {
		return err
	}

	producedUTXOSlice := baseTx.UTXOs()
	err = b.addUTXOs(ctx, constants.PlatformChainID, producedUTXOSlice)
	if err != nil {
		return err
	}

	b.txsLock.Lock()
	defer b.txsLock.Unlock()

	b.txs[txID] = tx
	return nil
}

func (b *backend) addUTXOs(ctx stdcontext.Context, destinationChainID ids.ID, utxos []*axc.UTXO) error {
	for _, utxo := range utxos {
		if err := b.AddUTXO(ctx, destinationChainID, utxo); err != nil {
			return err
		}
	}
	return nil
}

func (b *backend) removeUTXOs(ctx stdcontext.Context, sourceChain ids.ID, utxoIDs ids.Set) error {
	for utxoID := range utxoIDs {
		if err := b.RemoveUTXO(ctx, sourceChain, utxoID); err != nil {
			return err
		}
	}
	return nil
}

func (b *backend) GetTx(_ stdcontext.Context, txID ids.ID) (*platformvm.Tx, error) {
	b.txsLock.RLock()
	defer b.txsLock.RUnlock()

	tx, exists := b.txs[txID]
	if !exists {
		return nil, database.ErrNotFound
	}
	return tx, nil
}
