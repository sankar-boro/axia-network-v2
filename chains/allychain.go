// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chains

import (
	"sync"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow/consensus/axia"
	"github.com/sankar-boro/axia-network-v2/snow/engine/common"
	"github.com/sankar-boro/axia-network-v2/snow/networking/sender"
)

var _ Allychain = &allychain{}

// Allychain keeps track of the currently bootstrapping chains in a allychain. If no
// chains in the allychain are currently bootstrapping, the allychain is considered
// bootstrapped.
type Allychain interface {
	common.Allychain

	afterBootstrapped() chan struct{}

	addChain(chainID ids.ID)
	removeChain(chainID ids.ID)
}

type AllychainConfig struct {
	sender.GossipConfig

	// ValidatorOnly indicates that this Allychain's Chains are available to only allychain validators.
	ValidatorOnly       bool                 `json:"validatorOnly"`
	ConsensusParameters axia.Parameters `json:"consensusParameters"`
}

type allychain struct {
	lock             sync.RWMutex
	bootstrapping    ids.Set
	once             sync.Once
	bootstrappedSema chan struct{}
}

func newAllychain() Allychain {
	return &allychain{
		bootstrappedSema: make(chan struct{}),
	}
}

func (s *allychain) IsBootstrapped() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.bootstrapping.Len() == 0
}

func (s *allychain) Bootstrapped(chainID ids.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.bootstrapping.Remove(chainID)
	if s.bootstrapping.Len() > 0 {
		return
	}

	s.once.Do(func() {
		close(s.bootstrappedSema)
	})
}

func (s *allychain) afterBootstrapped() chan struct{} {
	return s.bootstrappedSema
}

func (s *allychain) addChain(chainID ids.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.bootstrapping.Add(chainID)
}

func (s *allychain) removeChain(chainID ids.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.bootstrapping.Remove(chainID)
}
