// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package block

import (
	"github.com/sankar-boro/axia-network-v2/codec"
	"github.com/sankar-boro/axia-network-v2/codec/linearcodec"
	"github.com/sankar-boro/axia-network-v2/utils/wrappers"
)

const version = 0

var c codec.Manager

func init() {
	lc := linearcodec.NewDefault()
	c = codec.NewDefaultManager()

	errs := wrappers.Errs{}
	errs.Add(
		lc.RegisterType(&statelessBlock{}),
		lc.RegisterType(&option{}),

		c.RegisterCodec(version, lc),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
