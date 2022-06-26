// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"errors"

	"github.com/sankar-boro/avalanchego/codec"
	"github.com/sankar-boro/avalanchego/ids"
	"github.com/sankar-boro/avalanchego/snow"
	"github.com/sankar-boro/avalanchego/vms/components/axc"
)

var (
	errOperationsNotSortedUnique = errors.New("operations not sorted and unique")
	errNoOperations              = errors.New("an operationTx must have at least one operation")
	errDoubleSpend               = errors.New("inputs attempt to double spend an input")

	_ UnsignedTx = &OperationTx{}
)

// OperationTx is a transaction with no credentials.
type OperationTx struct {
	BaseTx `serialize:"true"`

	Ops []*Operation `serialize:"true" json:"operations"`
}

func (t *OperationTx) InitCtx(ctx *snow.Context) {
	for _, op := range t.Ops {
		op.Op.InitCtx(ctx)
	}
	t.BaseTx.InitCtx(ctx)
}

// Operations track which ops this transaction is performing. The returned array
// should not be modified.
func (t *OperationTx) Operations() []*Operation { return t.Ops }

func (t *OperationTx) InputUTXOs() []*axc.UTXOID {
	utxos := t.BaseTx.InputUTXOs()
	for _, op := range t.Ops {
		utxos = append(utxos, op.UTXOIDs...)
	}
	return utxos
}

// ConsumedAssetIDs returns the IDs of the assets this transaction consumes
func (t *OperationTx) ConsumedAssetIDs() ids.Set {
	assets := t.AssetIDs()
	for _, op := range t.Ops {
		if len(op.UTXOIDs) > 0 {
			assets.Add(op.AssetID())
		}
	}
	return assets
}

// AssetIDs returns the IDs of the assets this transaction depends on
func (t *OperationTx) AssetIDs() ids.Set {
	assets := t.BaseTx.AssetIDs()
	for _, op := range t.Ops {
		assets.Add(op.AssetID())
	}
	return assets
}

// NumCredentials returns the number of expected credentials
func (t *OperationTx) NumCredentials() int {
	return t.BaseTx.NumCredentials() + len(t.Ops)
}

// UTXOs returns the UTXOs transaction is producing.
func (t *OperationTx) UTXOs() []*axc.UTXO {
	txID := t.ID()
	utxos := t.BaseTx.UTXOs()

	for _, op := range t.Ops {
		asset := op.AssetID()
		for _, out := range op.Op.Outs() {
			utxos = append(utxos, &axc.UTXO{
				UTXOID: axc.UTXOID{
					TxID:        txID,
					OutputIndex: uint32(len(utxos)),
				},
				Asset: axc.Asset{ID: asset},
				Out:   out,
			})
		}
	}

	return utxos
}

// SyntacticVerify that this transaction is well-formed.
func (t *OperationTx) SyntacticVerify(
	ctx *snow.Context,
	c codec.Manager,
	txFeeAssetID ids.ID,
	txFee uint64,
	_ uint64,
	numFxs int,
) error {
	switch {
	case t == nil:
		return errNilTx
	case len(t.Ops) == 0:
		return errNoOperations
	}

	if err := t.BaseTx.SyntacticVerify(ctx, c, txFeeAssetID, txFee, txFee, numFxs); err != nil {
		return err
	}

	inputs := ids.NewSet(len(t.Ins))
	for _, in := range t.Ins {
		inputs.Add(in.InputID())
	}

	for _, op := range t.Ops {
		if err := op.Verify(c); err != nil {
			return err
		}
		for _, utxoID := range op.UTXOIDs {
			inputID := utxoID.InputID()
			if inputs.Contains(inputID) {
				return errDoubleSpend
			}
			inputs.Add(inputID)
		}
	}
	if !IsSortedAndUniqueOperations(t.Ops, c) {
		return errOperationsNotSortedUnique
	}
	return nil
}

func (t *OperationTx) Visit(v Visitor) error {
	return v.OperationTx(t)
}
