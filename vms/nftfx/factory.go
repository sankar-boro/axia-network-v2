// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nftfx

import (
	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow"
	"github.com/sankar-boro/axia/vms"
)

var (
	_ vms.Factory = &Factory{}

	// ID that this Fx uses when labeled
	ID = ids.ID{'n', 'f', 't', 'f', 'x'}
)

type Factory struct{}

func (f *Factory) New(*snow.Context) (interface{}, error) { return &Fx{}, nil }
