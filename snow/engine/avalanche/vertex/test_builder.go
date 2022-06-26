// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vertex

import (
	"errors"
	"testing"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow/consensus/axia"
	"github.com/sankar-boro/axia/snow/consensus/snowstorm"
)

var (
	errBuild = errors.New("unexpectedly called Build")

	_ Builder = &TestBuilder{}
)

type TestBuilder struct {
	T             *testing.T
	CantBuildVtx  bool
	BuildVtxF     func(parentIDs []ids.ID, txs []snowstorm.Tx) (axia.Vertex, error)
	BuildStopVtxF func(parentIDs []ids.ID) (axia.Vertex, error)
}

func (b *TestBuilder) Default(cant bool) { b.CantBuildVtx = cant }

func (b *TestBuilder) BuildVtx(parentIDs []ids.ID, txs []snowstorm.Tx) (axia.Vertex, error) {
	if b.BuildVtxF != nil {
		return b.BuildVtxF(parentIDs, txs)
	}
	if b.CantBuildVtx && b.T != nil {
		b.T.Fatal(errBuild)
	}
	return nil, errBuild
}

func (b *TestBuilder) BuildStopVtx(parentIDs []ids.ID) (axia.Vertex, error) {
	if b.BuildStopVtxF != nil {
		return b.BuildStopVtxF(parentIDs)
	}
	if b.CantBuildVtx && b.T != nil {
		b.T.Fatal(errBuild)
	}
	return nil, errBuild
}
