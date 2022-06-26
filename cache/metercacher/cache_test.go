// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metercacher

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sankar-boro/axia-network-v2/cache"
)

func TestInterface(t *testing.T) {
	for _, test := range cache.CacherTests {
		cache := &cache.LRU{Size: test.Size}
		c, err := New("", prometheus.NewRegistry(), cache)
		if err != nil {
			t.Fatal(err)
		}

		test.Func(t, c)
	}
}
