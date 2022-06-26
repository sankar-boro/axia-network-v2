// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sender

import (
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/message"
)

// ExternalSender sends consensus messages to other validators
// Right now this is implemented in the networking package
type ExternalSender interface {
	// Send a message to a specific set of nodes
	Send(
		msg message.OutboundMessage,
		nodeIDs ids.NodeIDSet,
		allychainID ids.ID,
		validatorOnly bool,
	) ids.NodeIDSet

	// Send a message to a random group of nodes in a allychain.
	// Nodes are sampled based on their validator status.
	Gossip(
		msg message.OutboundMessage,
		allychainID ids.ID,
		validatorOnly bool,
		numValidatorsToSend int,
		numNonValidatorsToSend int,
		numPeersToSend int,
	) ids.NodeIDSet
}
