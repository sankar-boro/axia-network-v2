// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"github.com/sankar-boro/axia-network-v2/snow"
	"github.com/sankar-boro/axia-network-v2/vms"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm/config"
)

var _ vms.Factory = &Factory{}

// Factory can create new instances of the Platform Chain
type Factory struct {
	config.Config
}

// New returns a new instance of the Platform Chain
func (f *Factory) New(*snow.Context) (interface{}, error) {
	return &VM{Factory: *f}, nil
}
