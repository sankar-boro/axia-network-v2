// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow/networking/router"
	"github.com/sankar-boro/axia-network-v2/version"
)

var _ router.ExternalHandler = &testHandler{}

type testHandler struct {
	router.InboundHandler
	ConnectedF    func(nodeID ids.NodeID, nodeVersion version.Application)
	DisconnectedF func(nodeID ids.NodeID)
}

func (h *testHandler) Connected(id ids.NodeID, nodeVersion version.Application) {
	if h.ConnectedF != nil {
		h.ConnectedF(id, nodeVersion)
	}
}

func (h *testHandler) Disconnected(id ids.NodeID) {
	if h.DisconnectedF != nil {
		h.DisconnectedF(id)
	}
}
