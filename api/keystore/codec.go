// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keystore

import (
	"github.com/sankar-boro/axia/codec"
	"github.com/sankar-boro/axia/codec/linearcodec"
	"github.com/sankar-boro/axia/utils/units"
)

const (
	maxPackerSize  = 1 * units.GiB // max size, in bytes, of something being marshalled by Marshal()
	maxSliceLength = 256 * 1024

	codecVersion = 0
)

var c codec.Manager

func init() {
	lc := linearcodec.NewCustomMaxLength(maxSliceLength)
	c = codec.NewManager(maxPackerSize)
	if err := c.RegisterCodec(codecVersion, lc); err != nil {
		panic(err)
	}
}
