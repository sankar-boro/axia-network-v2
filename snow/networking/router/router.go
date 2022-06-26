// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package router

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sankar-boro/axia-network-v2/api/health"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/message"
	"github.com/sankar-boro/axia-network-v2/snow/networking/benchlist"
	"github.com/sankar-boro/axia-network-v2/snow/networking/handler"
	"github.com/sankar-boro/axia-network-v2/snow/networking/timeout"
	"github.com/sankar-boro/axia-network-v2/utils/logging"
)

// Router routes consensus messages to the Handler of the consensus
// engine that the messages are intended for
type Router interface {
	ExternalHandler
	InternalHandler

	Initialize(
		nodeID ids.NodeID,
		log logging.Logger,
		msgCreator message.Creator,
		timeouts timeout.Manager,
		shutdownTimeout time.Duration,
		criticalChains ids.Set,
		onFatal func(exitCode int),
		healthConfig HealthConfig,
		metricsNamespace string,
		metricsRegisterer prometheus.Registerer,
	) error
	Shutdown()
	AddChain(chain handler.Handler)
	health.Checker
}

// InternalHandler deals with messages internal to this node
type InternalHandler interface {
	benchlist.Benchable

	RegisterRequest(
		nodeID ids.NodeID,
		chainID ids.ID,
		requestID uint32,
		op message.Op,
	)
}
