// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vertex

import (
	"errors"
	"testing"

	"github.com/sankar-boro/axia/snow/consensus/axia"
)

var (
	errParse = errors.New("unexpectedly called Parse")

	_ Parser = &TestParser{}
)

type TestParser struct {
	T            *testing.T
	CantParseVtx bool
	ParseVtxF    func([]byte) (axia.Vertex, error)
}

func (p *TestParser) Default(cant bool) { p.CantParseVtx = cant }

func (p *TestParser) ParseVtx(b []byte) (axia.Vertex, error) {
	if p.ParseVtxF != nil {
		return p.ParseVtxF(b)
	}
	if p.CantParseVtx && p.T != nil {
		p.T.Fatal(errParse)
	}
	return nil, errParse
}
