// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package axia

import (
	"errors"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow/consensus/axia"
	"github.com/sankar-boro/axia-network-v2/snow/engine/common"
)

var (
	_ Engine = &EngineTest{}

	errGetVtx = errors.New("unexpectedly called GetVtx")
)

// EngineTest is a test engine
type EngineTest struct {
	common.EngineTest

	CantGetVtx bool
	GetVtxF    func(vtxID ids.ID) (axia.Vertex, error)
}

func (e *EngineTest) Default(cant bool) {
	e.EngineTest.Default(cant)
	e.CantGetVtx = false
}

func (e *EngineTest) GetVtx(vtxID ids.ID) (axia.Vertex, error) {
	if e.GetVtxF != nil {
		return e.GetVtxF(vtxID)
	}
	if e.CantGetVtx && e.T != nil {
		e.T.Fatalf("Unexpectedly called GetVtx")
	}
	return nil, errGetVtx
}
