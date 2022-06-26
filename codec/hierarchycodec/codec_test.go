// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package hierarchycodec

import (
	"testing"

	"github.com/sankar-boro/axia/codec"
)

func TestVectors(t *testing.T) {
	for _, test := range codec.Tests {
		c := NewDefault()
		test(c, t)
	}
}

func TestMultipleTags(t *testing.T) {
	for _, test := range codec.MultipleTagsTests {
		c := New([]string{"tag1", "tag2"}, defaultMaxSliceLength)
		test(c, t)
	}
}
