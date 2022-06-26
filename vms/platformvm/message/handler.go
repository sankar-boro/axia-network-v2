// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/utils/logging"
)

var _ Handler = NoopHandler{}

type Handler interface {
	HandleTx(nodeID ids.NodeID, requestID uint32, msg *Tx) error
}

type NoopHandler struct {
	Log logging.Logger
}

func (h NoopHandler) HandleTx(nodeID ids.NodeID, requestID uint32, _ *Tx) error {
	h.Log.Debug(
		"dropping unexpected Tx message from %s with requestID %s",
		nodeID,
		requestID,
	)
	return nil
}
