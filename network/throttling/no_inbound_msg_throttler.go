// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package throttling

import (
	"context"

	"github.com/sankar-boro/axia/ids"
)

var _ InboundMsgThrottler = &noInboundMsgThrottler{}

// Returns an InboundMsgThrottler where Acquire() always returns immediately.
func NewNoInboundThrottler() InboundMsgThrottler {
	return &noInboundMsgThrottler{}
}

// [Acquire] always returns immediately.
type noInboundMsgThrottler struct{}

func (*noInboundMsgThrottler) Acquire(context.Context, uint64, ids.NodeID) {}

func (*noInboundMsgThrottler) Release(uint64, ids.NodeID) {}

func (*noInboundMsgThrottler) AddNode(ids.NodeID) {}

func (*noInboundMsgThrottler) RemoveNode(ids.NodeID) {}
