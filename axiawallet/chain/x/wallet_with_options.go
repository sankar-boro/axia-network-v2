// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package x

import (
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/vms/avm/txs"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/components/verify"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
	"github.com/sankar-boro/axia-network-v2/axiawallet/subnet/primary/common"
)

var _ AxiaWallet = &axiawalletWithOptions{}

func NewAxiaWalletWithOptions(
	axiawallet AxiaWallet,
	options ...common.Option,
) AxiaWallet {
	return &axiawalletWithOptions{
		AxiaWallet:  axiawallet,
		options: options,
	}
}

type axiawalletWithOptions struct {
	AxiaWallet
	options []common.Option
}

func (w *axiawalletWithOptions) Builder() Builder {
	return NewBuilderWithOptions(
		w.AxiaWallet.Builder(),
		w.options...,
	)
}

func (w *axiawalletWithOptions) IssueBaseTx(
	outputs []*axc.TransferableOutput,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueBaseTx(
		outputs,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueCreateAssetTx(
	name string,
	symbol string,
	denomination byte,
	initialState map[uint32][]verify.State,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueCreateAssetTx(
		name,
		symbol,
		denomination,
		initialState,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueOperationTx(
	operations []*txs.Operation,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueOperationTx(
		operations,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueOperationTxMintFT(
	outputs map[ids.ID]*secp256k1fx.TransferOutput,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueOperationTxMintFT(
		outputs,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueOperationTxMintNFT(
	assetID ids.ID,
	payload []byte,
	owners []*secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueOperationTxMintNFT(
		assetID,
		payload,
		owners,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueOperationTxMintProperty(
	assetID ids.ID,
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueOperationTxMintProperty(
		assetID,
		owner,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueOperationTxBurnProperty(
	assetID ids.ID,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueOperationTxBurnProperty(
		assetID,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueImportTx(
	chainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueImportTx(
		chainID,
		to,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueExportTx(
	chainID ids.ID,
	outputs []*axc.TransferableOutput,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueExportTx(
		chainID,
		outputs,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueUnsignedTx(
	utx txs.UnsignedTx,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueUnsignedTx(
		utx,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueTx(
	tx *txs.Tx,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueTx(
		tx,
		common.UnionOptions(w.options, options)...,
	)
}
