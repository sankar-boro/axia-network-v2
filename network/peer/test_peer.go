// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package peer

import (
	"context"
	"crypto"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/message"
	"github.com/sankar-boro/axia-network-v2/network/throttling"
	"github.com/sankar-boro/axia-network-v2/snow/networking/router"
	"github.com/sankar-boro/axia-network-v2/snow/networking/tracker"
	"github.com/sankar-boro/axia-network-v2/snow/validators"
	"github.com/sankar-boro/axia-network-v2/staking"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/utils/ips"
	"github.com/sankar-boro/axia-network-v2/utils/logging"
	"github.com/sankar-boro/axia-network-v2/utils/math/meter"
	"github.com/sankar-boro/axia-network-v2/utils/resource"
	"github.com/sankar-boro/axia-network-v2/version"
)

const maxMessageToSend = 1024

// StartTestPeer provides a simple interface to create a peer that has finished
// the p2p handshake.
//
// This function will generate a new TLS key to use when connecting to the peer.
//
// The returned peer will not throttle inbound or outbound messages.
//
// - [ctx] provides a way of canceling the connection request.
// - [ip] is the remote that will be dialed to create the connection.
// - [networkID] will be sent to the peer during the handshake. If the peer is
//   expecting a different [networkID], the handshake will fail and an error
//   will be returned.
// - [router] will be called with all non-handshake messages received by the
//   peer.
func StartTestPeer(
	ctx context.Context,
	ip ips.IPPort,
	networkID uint32,
	router router.InboundHandler,
) (Peer, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, constants.NetworkType, ip.String())
	if err != nil {
		return nil, err
	}

	tlsCert, err := staking.NewTLSCert()
	if err != nil {
		return nil, err
	}

	tlsConfg := TLSConfig(*tlsCert)
	clientUpgrader := NewTLSClientUpgrader(tlsConfg)

	peerID, conn, cert, err := clientUpgrader.Upgrade(conn)
	if err != nil {
		return nil, err
	}

	mc, err := message.NewCreator(
		prometheus.NewRegistry(),
		true,
		"",
		10*time.Second,
	)
	if err != nil {
		return nil, err
	}

	metrics, err := NewMetrics(
		logging.NoLog{},
		"",
		prometheus.NewRegistry(),
	)
	if err != nil {
		return nil, err
	}

	ipPort := ips.IPPort{
		IP:   net.IPv6zero,
		Port: 0,
	}
	resourceTracker, err := tracker.NewResourceTracker(prometheus.NewRegistry(), resource.NoUsage, meter.ContinuousFactory{}, 10*time.Second)
	if err != nil {
		return nil, err
	}

	pingMessage, err := mc.Ping()
	if err != nil {
		return nil, err
	}

	peer := Start(
		&Config{
			Metrics:             metrics,
			MessageCreator:      mc,
			Log:                 logging.NoLog{},
			InboundMsgThrottler: throttling.NewNoInboundThrottler(),
			Network: NewTestNetwork(
				mc,
				networkID,
				ipPort,
				version.CurrentApp,
				tlsCert.PrivateKey.(crypto.Signer),
				ids.Set{},
				100,
			),
			Router:               router,
			VersionCompatibility: version.GetCompatibility(networkID),
			VersionParser:        version.DefaultApplicationParser,
			MySubnets:            ids.Set{},
			Beacons:              validators.NewSet(),
			NetworkID:            networkID,
			PingFrequency:        constants.DefaultPingFrequency,
			PongTimeout:          constants.DefaultPingPongTimeout,
			MaxClockDifference:   time.Minute,
			ResourceTracker:      resourceTracker,
			PingMessage:          pingMessage,
		},
		conn,
		cert,
		peerID,
		NewBlockingMessageQueue(
			metrics,
			logging.NoLog{},
			maxMessageToSend,
		),
	)
	return peer, peer.AwaitReady(ctx)
}
