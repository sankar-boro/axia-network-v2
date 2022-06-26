// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package peer

import (
	"context"
	"crypto"
	"crypto/x509"
	"net"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/avalanchego/ids"
	"github.com/sankar-boro/avalanchego/message"
	"github.com/sankar-boro/avalanchego/network/throttling"
	"github.com/sankar-boro/avalanchego/snow/networking/router"
	"github.com/sankar-boro/avalanchego/snow/networking/tracker"
	"github.com/sankar-boro/avalanchego/snow/validators"
	"github.com/sankar-boro/avalanchego/staking"
	"github.com/sankar-boro/avalanchego/utils/constants"
	"github.com/sankar-boro/avalanchego/utils/ips"
	"github.com/sankar-boro/avalanchego/utils/logging"
	"github.com/sankar-boro/avalanchego/utils/math/meter"
	"github.com/sankar-boro/avalanchego/utils/resource"
	"github.com/sankar-boro/avalanchego/version"
)

type testPeer struct {
	Peer
	inboundMsgChan <-chan message.InboundMessage
}

type rawTestPeer struct {
	config         *Config
	conn           net.Conn
	cert           *x509.Certificate
	nodeID         ids.NodeID
	inboundMsgChan <-chan message.InboundMessage
}

func newMessageCreator(t *testing.T) message.Creator {
	t.Helper()
	mc, err := message.NewCreator(
		prometheus.NewRegistry(),
		true,
		"",
		10*time.Second,
	)
	assert.NoError(t, err)
	return mc
}

func makeRawTestPeers(t *testing.T) (*rawTestPeer, *rawTestPeer) {
	t.Helper()
	assert := assert.New(t)

	conn0, conn1 := net.Pipe()

	tlsCert0, err := staking.NewTLSCert()
	assert.NoError(err)

	tlsCert1, err := staking.NewTLSCert()
	assert.NoError(err)

	nodeID0 := ids.NodeIDFromCert(tlsCert0.Leaf)
	nodeID1 := ids.NodeIDFromCert(tlsCert1.Leaf)

	mc := newMessageCreator(t)

	pingMessage, err := mc.Ping()
	assert.NoError(err)

	metrics, err := NewMetrics(
		logging.NoLog{},
		"",
		prometheus.NewRegistry(),
	)
	assert.NoError(err)

	resourceTracker, err := tracker.NewResourceTracker(prometheus.NewRegistry(), resource.NoUsage, meter.ContinuousFactory{}, 10*time.Second)
	assert.NoError(err)
	sharedConfig := Config{
		Metrics:              metrics,
		MessageCreator:       mc,
		Log:                  logging.NoLog{},
		InboundMsgThrottler:  throttling.NewNoInboundThrottler(),
		VersionCompatibility: version.GetCompatibility(constants.LocalID),
		VersionParser:        version.DefaultApplicationParser,
		MySubnets:            ids.Set{},
		Beacons:              validators.NewSet(),
		NetworkID:            constants.LocalID,
		PingFrequency:        constants.DefaultPingFrequency,
		PongTimeout:          constants.DefaultPingPongTimeout,
		MaxClockDifference:   time.Minute,
		ResourceTracker:      resourceTracker,
		PingMessage:          pingMessage,
	}
	peerConfig0 := sharedConfig
	peerConfig1 := sharedConfig

	peerConfig0.Network = &testNetwork{
		mc: mc,

		networkID: constants.LocalID,
		ip: ips.IPPort{
			IP:   net.IPv6loopback,
			Port: 0,
		},
		version: version.CurrentApp,
		signer:  tlsCert0.PrivateKey.(crypto.Signer),
		subnets: ids.Set{},

		uptime: 100,
	}
	inboundMsgChan0 := make(chan message.InboundMessage)
	peerConfig0.Router = router.InboundHandlerFunc(func(msg message.InboundMessage) {
		inboundMsgChan0 <- msg
	})

	peerConfig1.Network = &testNetwork{
		mc: mc,

		networkID: constants.LocalID,
		ip: ips.IPPort{
			IP:   net.IPv6loopback,
			Port: 1,
		},
		version: version.CurrentApp,
		signer:  tlsCert1.PrivateKey.(crypto.Signer),
		subnets: ids.Set{},

		uptime: 100,
	}
	inboundMsgChan1 := make(chan message.InboundMessage)
	peerConfig1.Router = router.InboundHandlerFunc(func(msg message.InboundMessage) {
		inboundMsgChan1 <- msg
	})

	peer0 := &rawTestPeer{
		config:         &peerConfig0,
		conn:           conn0,
		cert:           tlsCert0.Leaf,
		nodeID:         nodeID0,
		inboundMsgChan: inboundMsgChan0,
	}
	peer1 := &rawTestPeer{
		config:         &peerConfig1,
		conn:           conn1,
		cert:           tlsCert1.Leaf,
		nodeID:         nodeID1,
		inboundMsgChan: inboundMsgChan1,
	}
	return peer0, peer1
}

func makeTestPeers(t *testing.T) (*testPeer, *testPeer) {
	rawPeer0, rawPeer1 := makeRawTestPeers(t)

	peer0 := &testPeer{
		Peer: Start(
			rawPeer0.config,
			rawPeer0.conn,
			rawPeer1.cert,
			rawPeer1.nodeID,
			NewThrottledMessageQueue(
				rawPeer0.config.Metrics,
				rawPeer1.nodeID,
				logging.NoLog{},
				throttling.NewNoOutboundThrottler(),
			),
		),
		inboundMsgChan: rawPeer0.inboundMsgChan,
	}
	peer1 := &testPeer{
		Peer: Start(
			rawPeer1.config,
			rawPeer1.conn,
			rawPeer0.cert,
			rawPeer0.nodeID,
			NewThrottledMessageQueue(
				rawPeer1.config.Metrics,
				rawPeer0.nodeID,
				logging.NoLog{},
				throttling.NewNoOutboundThrottler(),
			),
		),
		inboundMsgChan: rawPeer1.inboundMsgChan,
	}
	return peer0, peer1
}

func makeReadyTestPeers(t *testing.T) (*testPeer, *testPeer) {
	t.Helper()
	assert := assert.New(t)

	peer0, peer1 := makeTestPeers(t)

	err := peer0.AwaitReady(context.Background())
	assert.NoError(err)
	isReady := peer0.Ready()
	assert.True(isReady)

	err = peer1.AwaitReady(context.Background())
	assert.NoError(err)
	isReady = peer1.Ready()
	assert.True(isReady)

	return peer0, peer1
}

func TestReady(t *testing.T) {
	assert := assert.New(t)

	rawPeer0, rawPeer1 := makeRawTestPeers(t)

	peer0 := Start(
		rawPeer0.config,
		rawPeer0.conn,
		rawPeer1.cert,
		rawPeer1.nodeID,
		NewThrottledMessageQueue(
			rawPeer0.config.Metrics,
			rawPeer1.nodeID,
			logging.NoLog{},
			throttling.NewNoOutboundThrottler(),
		),
	)

	isReady := peer0.Ready()
	assert.False(isReady)

	peer1 := Start(
		rawPeer1.config,
		rawPeer1.conn,
		rawPeer0.cert,
		rawPeer0.nodeID,
		NewThrottledMessageQueue(
			rawPeer1.config.Metrics,
			rawPeer0.nodeID,
			logging.NoLog{},
			throttling.NewNoOutboundThrottler(),
		),
	)

	err := peer0.AwaitReady(context.Background())
	assert.NoError(err)
	isReady = peer0.Ready()
	assert.True(isReady)

	err = peer1.AwaitReady(context.Background())
	assert.NoError(err)
	isReady = peer1.Ready()
	assert.True(isReady)

	peer0.StartClose()
	err = peer0.AwaitClosed(context.Background())
	assert.NoError(err)
	err = peer1.AwaitClosed(context.Background())
	assert.NoError(err)
}

func TestSend(t *testing.T) {
	assert := assert.New(t)

	peer0, peer1 := makeReadyTestPeers(t)
	mc := newMessageCreator(t)

	outboundGetMsg, err := mc.Get(ids.Empty, 1, time.Second, ids.Empty)
	assert.NoError(err)

	sent := peer0.Send(context.Background(), outboundGetMsg)
	assert.True(sent)

	inboundGetMsg := <-peer1.inboundMsgChan
	assert.Equal(message.Get, inboundGetMsg.Op())

	peer1.StartClose()
	err = peer0.AwaitClosed(context.Background())
	assert.NoError(err)
	err = peer1.AwaitClosed(context.Background())
	assert.NoError(err)
}
