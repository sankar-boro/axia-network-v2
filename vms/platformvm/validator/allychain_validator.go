// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validator

import (
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
)

// AllychainValidator validates a allychain on the Axia network.
type AllychainValidator struct {
	Validator `serialize:"true"`

	// ID of the allychain this validator is validating
	Allychain ids.ID `serialize:"true" json:"allychain"`
}

// AllychainID is the ID of the allychain this validator is validating
func (v *AllychainValidator) AllychainID() ids.ID { return v.Allychain }

// Verify this validator is valid
func (v *AllychainValidator) Verify() error {
	switch v.Allychain {
	case constants.PrimaryNetworkID:
		return errBadAllychainID
	default:
		return v.Validator.Verify()
	}
}
