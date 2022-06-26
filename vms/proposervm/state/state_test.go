// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/database/memdb"
	"github.com/sankar-boro/axia-network-v2/database/versiondb"
)

func TestState(t *testing.T) {
	a := assert.New(t)

	db := memdb.New()
	vdb := versiondb.New(db)
	s := New(vdb)

	testBlockState(a, s)
	testChainState(a, s)
}

func TestMeteredState(t *testing.T) {
	a := assert.New(t)

	db := memdb.New()
	vdb := versiondb.New(db)
	s, err := NewMetered(vdb, "", prometheus.NewRegistry())
	a.NoError(err)

	testBlockState(a, s)
	testChainState(a, s)
}
