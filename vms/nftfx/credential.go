// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nftfx

import (
	"github.com/sankar-boro/axia/vms/secp256k1fx"
)

type Credential struct {
	secp256k1fx.Credential `serialize:"true"`
}
