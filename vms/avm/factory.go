// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"github.com/sankar-boro/axia/snow"
	"github.com/sankar-boro/axia/vms"
)

var _ vms.Factory = &Factory{}

type Factory struct {
	TxFee            uint64
	CreateAssetTxFee uint64
}

func (f *Factory) New(*snow.Context) (interface{}, error) {
	return &VM{Factory: *f}, nil
}
