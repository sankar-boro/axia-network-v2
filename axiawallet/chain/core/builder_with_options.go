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

var _ Builder = &builderWithOptions{}

type builderWithOptions struct {
	Builder
	options []common.Option
}

// NewBuilderWithOptions returns a new transaction builder that will use the
// given options by default.
//
// - [builder] is the builder that will be called to perform the underlying
//   opterations.
// - [options] will be provided to the builder in addition to the options
//   provided in the method calls.
func NewBuilderWithOptions(builder Builder, options ...common.Option) Builder {
	return &builderWithOptions{
		Builder: builder,
		options: options,
	}
}

func (b *builderWithOptions) GetBalance(
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return b.Builder.GetBalance(
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) GetImportableBalance(
	chainID ids.ID,
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return b.Builder.GetImportableBalance(
		chainID,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddValidatorTx(
	validator *coreChainValidator.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*platformvm.UnsignedAddValidatorTx, error) {
	return b.Builder.NewAddValidatorTx(
		validator,
		rewardsOwner,
		shares,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddAllychainValidatorTx(
	validator *coreChainValidator.AllychainValidator,
	options ...common.Option,
) (*platformvm.UnsignedAddAllychainValidatorTx, error) {
	return b.Builder.NewAddAllychainValidatorTx(
		validator,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddNominatorTx(
	validator *coreChainValidator.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*platformvm.UnsignedAddNominatorTx, error) {
	return b.Builder.NewAddNominatorTx(
		validator,
		rewardsOwner,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewCreateChainTx(
	allychainID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	chainName string,
	options ...common.Option,
) (*platformvm.UnsignedCreateChainTx, error) {
	return b.Builder.NewCreateChainTx(
		allychainID,
		genesis,
		vmID,
		fxIDs,
		chainName,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewCreateAllychainTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*platformvm.UnsignedCreateAllychainTx, error) {
	return b.Builder.NewCreateAllychainTx(
		owner,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*platformvm.UnsignedImportTx, error) {
	return b.Builder.NewImportTx(
		sourceChainID,
		to,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewExportTx(
	chainID ids.ID,
	outputs []*axc.TransferableOutput,
	options ...common.Option,
) (*platformvm.UnsignedExportTx, error) {
	return b.Builder.NewExportTx(
		chainID,
		outputs,
		common.UnionOptions(b.options, options)...,
	)
}
