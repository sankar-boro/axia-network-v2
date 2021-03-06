// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package x

import (
	"fmt"

	stdcontext "context"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/vms/avm/txs"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
)

var _ Backend = &backend{}

type ChainUTXOs interface {
	AddUTXO(ctx stdcontext.Context, destinationChainID ids.ID, utxo *axc.UTXO) error
	RemoveUTXO(ctx stdcontext.Context, sourceChainID, utxoID ids.ID) error

	UTXOs(ctx stdcontext.Context, sourceChainID ids.ID) ([]*axc.UTXO, error)
	GetUTXO(ctx stdcontext.Context, sourceChainID, utxoID ids.ID) (*axc.UTXO, error)
}

// Backend defines the full interface required to support an Swap-chain axiawallet.
type Backend interface {
	ChainUTXOs
	BuilderBackend
	SignerBackend

	AcceptTx(ctx stdcontext.Context, tx *txs.Tx) error
}

type backend struct {
	Context
	ChainUTXOs

	chainID ids.ID
}

func NewBackend(ctx Context, chainID ids.ID, utxos ChainUTXOs) Backend {
	return &backend{
		Context:    ctx,
		ChainUTXOs: utxos,

		chainID: chainID,
	}
}

// TODO: implement txs.Visitor here
func (b *backend) AcceptTx(ctx stdcontext.Context, tx *txs.Tx) error {
	switch utx := tx.UnsignedTx.(type) {
	case *txs.BaseTx, *txs.CreateAssetTx, *txs.OperationTx:
	case *txs.ImportTx:
		for _, input := range utx.ImportedIns {
			utxoID := input.UTXOID.InputID()
			if err := b.RemoveUTXO(ctx, utx.SourceChain, utxoID); err != nil {
				return err
			}
		}
	case *txs.ExportTx:
		txID := tx.ID()
		for i, out := range utx.ExportedOuts {
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
	default:
		return fmt.Errorf("%w: %T", errUnknownTxType, tx.UnsignedTx)
	}

	inputUTXOs := tx.UnsignedTx.InputUTXOs()
	for _, utxoID := range inputUTXOs {
		if utxoID.Symbol {
			continue
		}
		if err := b.RemoveUTXO(ctx, b.chainID, utxoID.InputID()); err != nil {
			return err
		}
	}

	outputUTXOs := tx.UnsignedTx.UTXOs()
	for _, utxo := range outputUTXOs {
		if err := b.AddUTXO(ctx, b.chainID, utxo); err != nil {
			return err
		}
	}
	return nil
}
