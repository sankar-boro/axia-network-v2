// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package propertyfx

import (
	"github.com/sankar-boro/avalanchego/vms/secp256k1fx"
)

type MintOutput struct {
	secp256k1fx.OutputOwners `serialize:"true"`
}
