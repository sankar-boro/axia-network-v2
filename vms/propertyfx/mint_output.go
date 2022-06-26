// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package propertyfx

import (
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
)

type MintOutput struct {
	secp256k1fx.OutputOwners `serialize:"true"`
}
