// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	"errors"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm/status"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
	"github.com/sankar-boro/axia-network-v2/axiawallet/allychain/primary/common"

	coreChainValidator "github.com/sankar-boro/axia-network-v2/vms/platformvm/validator"
)

var (
	errNotCommitted = errors.New("not committed")

	_ AxiaWallet = &axiawallet{}
)

type AxiaWallet interface {
	Context

	// Builder returns the builder that will be used to create the transactions.
	Builder() Builder

	// Signer returns the signer that will be used to sign the transactions.
	Signer() Signer

	// IssueBaseTx creates, signs, and issues a new simple value transfer.
	// Because the Core-chain doesn't intend for balance transfers to occur, this
	// method is expensive and abuses the creation of allychains.
	//
	// - [outputs] specifies all the recipients and amounts that should be sent
	//   from this transaction.
	IssueBaseTx(
		outputs []*axc.TransferableOutput,
		options ...common.Option,
	) (ids.ID, error)

	// IssueAddValidatorTx creates, signs, and issues a new validator of the
	// primary network.
	//
	// - [validator] specifies all the details of the validation period such as
	//   the startTime, endTime, stake weight, and nodeID.
	// - [rewardsOwner] specifies the owner of all the rewards this validator
	//   may accrue during its validation period.
	// - [shares] specifies the fraction (out of 1,000,000) that this validator
	//   will take from delegation rewards. If 1,000,000 is provided, 100% of
	//   the delegation reward will be sent to the validator's [rewardsOwner].
	IssueAddValidatorTx(
		validator *coreChainValidator.Validator,
		rewardsOwner *secp256k1fx.OutputOwners,
		shares uint32,
		options ...common.Option,
	) (ids.ID, error)

	// IssueAddAllychainValidatorTx creates, signs, and issues a new validator of a
	// allychain.
	//
	// - [validator] specifies all the details of the validation period such as
	//   the startTime, endTime, sampling weight, nodeID, and allychainID.
	IssueAddAllychainValidatorTx(
		validator *coreChainValidator.AllychainValidator,
		options ...common.Option,
	) (ids.ID, error)

	// IssueAddNominatorTx creates, signs, and issues a new nominator to a
	// validator on the primary network.
	//
	// - [validator] specifies all the details of the delegation period such as
	//   the startTime, endTime, stake weight, and validator's nodeID.
	// - [rewardsOwner] specifies the owner of all the rewards this nominator
	//   may accrue at the end of its delegation period.
	IssueAddNominatorTx(
		validator *coreChainValidator.Validator,
		rewardsOwner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (ids.ID, error)

	// IssueCreateChainTx creates, signs, and issues a new chain in the named
	// allychain.
	//
	// - [allychainID] specifies the allychain to launch the chain in.
	// - [genesis] specifies the initial state of the new chain.
	// - [vmID] specifies the vm that the new chain will run.
	// - [fxIDs] specifies all the feature extensions that the vm should be
	//   running with.
	// - [chainName] specifies a human readable name for the chain.
	IssueCreateChainTx(
		allychainID ids.ID,
		genesis []byte,
		vmID ids.ID,
		fxIDs []ids.ID,
		chainName string,
		options ...common.Option,
	) (ids.ID, error)

	// IssueCreateAllychainTx creates, signs, and issues a new allychain with the
	// specified owner.
	//
	// - [owner] specifies who has the ability to create new chains and add new
	//   validators to the allychain.
	IssueCreateAllychainTx(
		owner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (ids.ID, error)

	// IssueImportTx creates, signs, and issues an import transaction that
	// attempts to consume all the available UTXOs and import the funds to [to].
	//
	// - [chainID] specifies the chain to be importing funds from.
	// - [to] specifies where to send the imported funds to.
	IssueImportTx(
		chainID ids.ID,
		to *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (ids.ID, error)

	// IssueExportTx creates, signs, and issues an export transaction that
	// attempts to send all the provided [outputs] to the requested [chainID].
	//
	// - [chainID] specifies the chain to be exporting the funds to.
	// - [outputs] specifies the outputs to send to the [chainID].
	IssueExportTx(
		chainID ids.ID,
		outputs []*axc.TransferableOutput,
		options ...common.Option,
	) (ids.ID, error)

	// IssueUnsignedTx signs and issues the unsigned tx.
	IssueUnsignedTx(
		utx platformvm.UnsignedTx,
		options ...common.Option,
	) (ids.ID, error)

	// IssueTx issues the signed tx.
	IssueTx(
		tx *platformvm.Tx,
		options ...common.Option,
	) (ids.ID, error)
}

func NewAxiaWallet(
	builder Builder,
	signer Signer,
	client platformvm.Client,
	backend Backend,
) AxiaWallet {
	return &axiawallet{
		Backend: backend,
		builder: builder,
		signer:  signer,
		client:  client,
	}
}

type axiawallet struct {
	Backend
	builder Builder
	signer  Signer
	client  platformvm.Client
}

func (w *axiawallet) Builder() Builder { return w.builder }

func (w *axiawallet) Signer() Signer { return w.signer }

func (w *axiawallet) IssueBaseTx(
	outputs []*axc.TransferableOutput,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewBaseTx(outputs, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueAddValidatorTx(
	validator *coreChainValidator.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewAddValidatorTx(validator, rewardsOwner, shares, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueAddAllychainValidatorTx(
	validator *coreChainValidator.AllychainValidator,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewAddAllychainValidatorTx(validator, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueAddNominatorTx(
	validator *coreChainValidator.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewAddNominatorTx(validator, rewardsOwner, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueCreateChainTx(
	allychainID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	chainName string,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewCreateChainTx(allychainID, genesis, vmID, fxIDs, chainName, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueCreateAllychainTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewCreateAllychainTx(owner, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewImportTx(sourceChainID, to, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueExportTx(
	chainID ids.ID,
	outputs []*axc.TransferableOutput,
	options ...common.Option,
) (ids.ID, error) {
	utx, err := w.builder.NewExportTx(chainID, outputs, options...)
	if err != nil {
		return ids.Empty, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *axiawallet) IssueUnsignedTx(
	utx platformvm.UnsignedTx,
	options ...common.Option,
) (ids.ID, error) {
	ops := common.NewOptions(options)
	ctx := ops.Context()
	tx, err := w.signer.SignUnsigned(ctx, utx)
	if err != nil {
		return ids.Empty, err
	}

	return w.IssueTx(tx, options...)
}

func (w *axiawallet) IssueTx(
	tx *platformvm.Tx,
	options ...common.Option,
) (ids.ID, error) {
	ops := common.NewOptions(options)
	ctx := ops.Context()
	txID, err := w.client.IssueTx(ctx, tx.Bytes())
	if err != nil {
		return ids.Empty, err
	}

	if ops.AssumeDecided() {
		return txID, w.Backend.AcceptTx(ctx, tx)
	}

	txStatus, err := w.client.AwaitTxDecided(ctx, txID, ops.PollFrequency())
	if err != nil {
		return txID, err
	}

	if err := w.Backend.AcceptTx(ctx, tx); err != nil {
		return txID, err
	}

	if txStatus.Status != status.Committed {
		return txID, errNotCommitted
	}
	return txID, nil
}
