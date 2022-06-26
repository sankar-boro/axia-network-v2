// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sender

import (
	"errors"
	"testing"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/message"
)

var (
	errSend   = errors.New("unexpectedly called Send")
	errGossip = errors.New("unexpectedly called Gossip")
)

// ExternalSenderTest is a test sender
type ExternalSenderTest struct {
	TB testing.TB

	CantSend, CantGossip bool

	SendF   func(msg message.OutboundMessage, nodeIDs ids.NodeIDSet, allychainID ids.ID, validatorOnly bool) ids.NodeIDSet
	GossipF func(msg message.OutboundMessage, allychainID ids.ID, validatorOnly bool, numValidatorsToSend, numNonValidatorsToSend, numPeersToSend int) ids.NodeIDSet
}

// Default set the default callable value to [cant]
func (s *ExternalSenderTest) Default(cant bool) {
	s.CantSend = cant
	s.CantGossip = cant
}

func (s *ExternalSenderTest) Send(
	msg message.OutboundMessage,
	nodeIDs ids.NodeIDSet,
	allychainID ids.ID,
	validatorOnly bool,
) ids.NodeIDSet {
	if s.SendF != nil {
		return s.SendF(msg, nodeIDs, allychainID, validatorOnly)
	}
	if s.CantSend {
		if s.TB != nil {
			s.TB.Helper()
			s.TB.Fatal(errSend)
		}
	}
	return nil
}

// Given a msg type, the corresponding mock function is called if it was initialized.
// If it wasn't initialized and this function shouldn't be called and testing was
// initialized, then testing will fail.
func (s *ExternalSenderTest) Gossip(
	msg message.OutboundMessage,
	allychainID ids.ID,
	validatorOnly bool,
	numValidatorsToSend int,
	numNonValidatorsToSend int,
	numPeersToSend int,
) ids.NodeIDSet {
	if s.GossipF != nil {
		return s.GossipF(msg, allychainID, validatorOnly, numValidatorsToSend, numNonValidatorsToSend, numPeersToSend)
	}
	if s.CantGossip {
		if s.TB != nil {
			s.TB.Helper()
			s.TB.Fatal(errGossip)
		}
	}
	return nil
}
