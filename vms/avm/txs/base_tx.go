// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"github.com/sankar-boro/axia-network-v2/codec"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
)

var _ UnsignedTx = &BaseTx{}

// BaseTx is the basis of all transactions.
type BaseTx struct {
	axc.BaseTx `serialize:"true"`
}

func (t *BaseTx) InitCtx(ctx *snow.Context) {
	for _, out := range t.Outs {
		out.InitCtx(ctx)
	}
}

func (t *BaseTx) SyntacticVerify(
	ctx *snow.Context,
	c codec.Manager,
	txFeeAssetID ids.ID,
	txFee uint64,
	_ uint64,
	_ int,
) error {
	if t == nil {
		return errNilTx
	}

	if err := t.MetadataVerify(ctx); err != nil {
		return err
	}

	return axc.VerifyTx(
		txFee,
		txFeeAssetID,
		[][]*axc.TransferableInput{t.Ins},
		[][]*axc.TransferableOutput{t.Outs},
		c,
	)
}

func (t *BaseTx) Visit(v Visitor) error {
	return v.BaseTx(t)
}
