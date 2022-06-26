// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package peer

import (
	"time"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/message"
	"github.com/sankar-boro/axia/network/throttling"
	"github.com/sankar-boro/axia/snow/networking/router"
	"github.com/sankar-boro/axia/snow/networking/tracker"
	"github.com/sankar-boro/axia/snow/validators"
	"github.com/sankar-boro/axia/utils/logging"
	"github.com/sankar-boro/axia/utils/timer/mockable"
	"github.com/sankar-boro/axia/version"
)

type Config struct {
	// Size, in bytes, of the buffer this peer reads messages into
	ReadBufferSize int
	// Size, in bytes, of the buffer this peer writes messages into
	WriteBufferSize      int
	Clock                mockable.Clock
	Metrics              *Metrics
	MessageCreator       message.Creator
	Log                  logging.Logger
	InboundMsgThrottler  throttling.InboundMsgThrottler
	Network              Network
	Router               router.InboundHandler
	VersionCompatibility version.Compatibility
	VersionParser        version.ApplicationParser
	MySubnets            ids.Set
	Beacons              validators.Set
	NetworkID            uint32
	PingFrequency        time.Duration
	PongTimeout          time.Duration
	MaxClockDifference   time.Duration

	// Unix time of the last message sent and received respectively
	// Must only be accessed atomically
	LastSent, LastReceived int64

	// Tracks CPU/disk usage caused by each peer.
	ResourceTracker tracker.ResourceTracker

	PingMessage message.OutboundMessage
}
