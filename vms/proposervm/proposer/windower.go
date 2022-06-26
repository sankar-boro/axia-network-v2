// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package proposer

import (
	"sort"
	"time"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow/validators"
	"github.com/sankar-boro/axia-network-v2/utils/math"
	"github.com/sankar-boro/axia-network-v2/utils/sampler"
	"github.com/sankar-boro/axia-network-v2/utils/wrappers"
)

// Proposer list constants
const (
	MaxWindows     = 6
	WindowDuration = 5 * time.Second
	MaxDelay       = MaxWindows * WindowDuration
)

var _ Windower = &windower{}

type Windower interface {
	Delay(
		chainHeight,
		coreChainHeight uint64,
		validatorID ids.NodeID,
	) (time.Duration, error)
}

// windower interfaces with Core-Chain and it is responsible for calculating the
// delay for the block submission window of a given validator
type windower struct {
	state       validators.State
	allychainID    ids.ID
	chainSource uint64
	sampler     sampler.WeightedWithoutReplacement
}

func New(state validators.State, allychainID, chainID ids.ID) Windower {
	w := wrappers.Packer{Bytes: chainID[:]}
	return &windower{
		state:       state,
		allychainID:    allychainID,
		chainSource: w.UnpackLong(),
		sampler:     sampler.NewDeterministicWeightedWithoutReplacement(),
	}
}

func (w *windower) Delay(chainHeight, coreChainHeight uint64, validatorID ids.NodeID) (time.Duration, error) {
	if validatorID == ids.EmptyNodeID {
		return MaxDelay, nil
	}

	// get the validator set by the core-chain height
	validatorsMap, err := w.state.GetValidatorSet(coreChainHeight, w.allychainID)
	if err != nil {
		return 0, err
	}

	// convert the map of validators to a slice
	validators := make(validatorsSlice, 0, len(validatorsMap))
	weight := uint64(0)
	for k, v := range validatorsMap {
		validators = append(validators, validatorData{
			id:     k,
			weight: v,
		})
		newWeight, err := math.Add64(weight, v)
		if err != nil {
			return 0, err
		}
		weight = newWeight
	}

	// canonically sort validators
	// Note: validators are sorted by ID, sorting by weight would not create a
	// canonically sorted list
	sort.Sort(validators)

	// convert the slice of validators to a slice of weights
	validatorWeights := make([]uint64, len(validators))
	for i, v := range validators {
		validatorWeights[i] = v.weight
	}

	if err := w.sampler.Initialize(validatorWeights); err != nil {
		return 0, err
	}

	numToSample := MaxWindows
	if weight < uint64(numToSample) {
		numToSample = int(weight)
	}

	seed := chainHeight ^ w.chainSource
	w.sampler.Seed(int64(seed))

	indices, err := w.sampler.Sample(numToSample)
	if err != nil {
		return 0, err
	}

	delay := time.Duration(0)
	for _, index := range indices {
		nodeID := validators[index].id
		if nodeID == validatorID {
			return delay, nil
		}
		delay += WindowDuration
	}
	return delay, nil
}
