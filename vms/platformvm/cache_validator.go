// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"github.com/sankar-boro/axia/ids"
)

var _ validator = &validatorImpl{}

type validator interface {
	Nominators() []*UnsignedAddNominatorTx
	SubnetValidators() map[ids.ID]*UnsignedAddSubnetValidatorTx
}

type validatorImpl struct {
	// sorted in order of next operation, either addition or removal.
	nominators []*UnsignedAddNominatorTx
	// maps subnetID to tx
	subnets map[ids.ID]*UnsignedAddSubnetValidatorTx
}

func (v *validatorImpl) Nominators() []*UnsignedAddNominatorTx {
	return v.nominators
}

func (v *validatorImpl) SubnetValidators() map[ids.ID]*UnsignedAddSubnetValidatorTx {
	return v.subnets
}
