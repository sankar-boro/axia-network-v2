// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow/validators"
)

// A Allychain is a set of validators that are validating a set of blockchains
// Each blockchain is validated by one allychain; one allychain may validate many blockchains
type Allychain interface {
	// ID returns this allychain's ID
	ID() ids.ID

	// Validators returns the validators that compose this allychain
	Validators() []validators.Validator
}
