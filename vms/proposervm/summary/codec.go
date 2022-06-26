// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package summary

import (
	"errors"
	"math"

	"github.com/sankar-boro/axia-network-v2/codec"
	"github.com/sankar-boro/axia-network-v2/codec/linearcodec"
)

const codecVersion = 0

var (
	c codec.Manager

	errWrongCodecVersion = errors.New("wrong codec version")
)

func init() {
	lc := linearcodec.NewCustomMaxLength(math.MaxUint32)
	c = codec.NewManager(math.MaxInt32)
	if err := c.RegisterCodec(codecVersion, lc); err != nil {
		panic(err)
	}
}
