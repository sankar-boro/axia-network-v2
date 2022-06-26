// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

var _ currentValidator = &currentValidatorImpl{}

type currentValidator interface {
	validator

	AddValidatorTx() *UnsignedAddValidatorTx

	// Weight of delegations to this validator. Doesn't include the stake
	// provided by this validator.
	NominatorWeight() uint64

	PotentialReward() uint64
}

type currentValidatorImpl struct {
	// nominators are sorted in order of removal.
	validatorImpl

	addValidatorTx  *UnsignedAddValidatorTx
	nominatorWeight uint64
	potentialReward uint64
}

func (v *currentValidatorImpl) AddValidatorTx() *UnsignedAddValidatorTx {
	return v.addValidatorTx
}

func (v *currentValidatorImpl) NominatorWeight() uint64 {
	return v.nominatorWeight
}

func (v *currentValidatorImpl) PotentialReward() uint64 {
	return v.potentialReward
}
