// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
	"github.com/sankar-boro/axia-network-v2/axiawallet/allychain/primary/common"

	coreChainValidator "github.com/sankar-boro/axia-network-v2/vms/platformvm/validator"
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

func (w *axiawalletWithOptions) IssueAddValidatorTx(
	validator *coreChainValidator.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueAddValidatorTx(
		validator,
		rewardsOwner,
		shares,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueAddAllychainValidatorTx(
	validator *coreChainValidator.AllychainValidator,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueAddAllychainValidatorTx(
		validator,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueAddNominatorTx(
	validator *coreChainValidator.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueAddNominatorTx(
		validator,
		rewardsOwner,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueCreateChainTx(
	allychainID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	chainName string,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueCreateChainTx(
		allychainID,
		genesis,
		vmID,
		fxIDs,
		chainName,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueCreateAllychainTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueCreateAllychainTx(
		owner,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueImportTx(
		sourceChainID,
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
	utx platformvm.UnsignedTx,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueUnsignedTx(
		utx,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *axiawalletWithOptions) IssueTx(
	tx *platformvm.Tx,
	options ...common.Option,
) (ids.ID, error) {
	return w.AxiaWallet.IssueTx(
		tx,
		common.UnionOptions(w.options, options)...,
	)
}
