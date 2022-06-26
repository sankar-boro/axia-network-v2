// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/sankar-boro/axia/database"
	"github.com/sankar-boro/axia/ids"
)

var _ pendingStakerChainState = &pendingStakerChainStateImpl{}

// pendingStakerChainState manages the set of stakers (both validators and
// nominators) that are slated to start staking in the future.
type pendingStakerChainState interface {
	GetValidatorTx(nodeID ids.NodeID) (addStakerTx *UnsignedAddValidatorTx, err error)
	GetValidator(nodeID ids.NodeID) validator

	AddStaker(addStakerTx *Tx) pendingStakerChainState
	DeleteStakers(numToRemove int) pendingStakerChainState

	// Stakers returns the list of pending validators in order of their removal
	// from the pending staker set
	Stakers() []*Tx

	Apply(InternalState)
}

// pendingStakerChainStateImpl is a copy on write implementation for versioning
// the validator set. None of the slices, maps, or pointers should be modified
// after initialization.
type pendingStakerChainStateImpl struct {
	// nodeID -> validator
	validatorsByNodeID      map[ids.NodeID]*UnsignedAddValidatorTx
	validatorExtrasByNodeID map[ids.NodeID]*validatorImpl

	// list of pending validators in order of their removal from the pending
	// staker set
	validators []*Tx

	addedStakers   []*Tx
	deletedStakers []*Tx
}

func (ps *pendingStakerChainStateImpl) GetValidatorTx(nodeID ids.NodeID) (addStakerTx *UnsignedAddValidatorTx, err error) {
	vdr, exists := ps.validatorsByNodeID[nodeID]
	if !exists {
		return nil, database.ErrNotFound
	}
	return vdr, nil
}

func (ps *pendingStakerChainStateImpl) GetValidator(nodeID ids.NodeID) validator {
	if vdr, exists := ps.validatorExtrasByNodeID[nodeID]; exists {
		return vdr
	}
	return &validatorImpl{}
}

func (ps *pendingStakerChainStateImpl) AddStaker(addStakerTx *Tx) pendingStakerChainState {
	newPS := &pendingStakerChainStateImpl{
		validators:   make([]*Tx, len(ps.validators)+1),
		addedStakers: []*Tx{addStakerTx},
	}
	copy(newPS.validators, ps.validators)
	newPS.validators[len(ps.validators)] = addStakerTx
	sortValidatorsByAddition(newPS.validators)

	switch tx := addStakerTx.UnsignedTx.(type) {
	case *UnsignedAddValidatorTx:
		newPS.validatorExtrasByNodeID = ps.validatorExtrasByNodeID

		newPS.validatorsByNodeID = make(map[ids.NodeID]*UnsignedAddValidatorTx, len(ps.validatorsByNodeID)+1)
		for nodeID, vdr := range ps.validatorsByNodeID {
			newPS.validatorsByNodeID[nodeID] = vdr
		}
		newPS.validatorsByNodeID[tx.Validator.NodeID] = tx
	case *UnsignedAddNominatorTx:
		newPS.validatorsByNodeID = ps.validatorsByNodeID

		newPS.validatorExtrasByNodeID = make(map[ids.NodeID]*validatorImpl, len(ps.validatorExtrasByNodeID)+1)
		for nodeID, vdr := range ps.validatorExtrasByNodeID {
			if nodeID != tx.Validator.NodeID {
				newPS.validatorExtrasByNodeID[nodeID] = vdr
			}
		}
		if vdr, exists := ps.validatorExtrasByNodeID[tx.Validator.NodeID]; exists {
			newNominators := make([]*UnsignedAddNominatorTx, len(vdr.nominators)+1)
			copy(newNominators, vdr.nominators)
			newNominators[len(vdr.nominators)] = tx
			sortNominatorsByAddition(newNominators)

			newPS.validatorExtrasByNodeID[tx.Validator.NodeID] = &validatorImpl{
				nominators: newNominators,
				subnets:    vdr.subnets,
			}
		} else {
			newPS.validatorExtrasByNodeID[tx.Validator.NodeID] = &validatorImpl{
				nominators: []*UnsignedAddNominatorTx{
					tx,
				},
			}
		}
	case *UnsignedAddSubnetValidatorTx:
		newPS.validatorsByNodeID = ps.validatorsByNodeID

		newPS.validatorExtrasByNodeID = make(map[ids.NodeID]*validatorImpl, len(ps.validatorExtrasByNodeID)+1)
		for nodeID, vdr := range ps.validatorExtrasByNodeID {
			if nodeID != tx.Validator.NodeID {
				newPS.validatorExtrasByNodeID[nodeID] = vdr
			}
		}
		if vdr, exists := ps.validatorExtrasByNodeID[tx.Validator.NodeID]; exists {
			newSubnets := make(map[ids.ID]*UnsignedAddSubnetValidatorTx, len(vdr.subnets)+1)
			for subnet, subnetTx := range vdr.subnets {
				newSubnets[subnet] = subnetTx
			}
			newSubnets[tx.Validator.Subnet] = tx

			newPS.validatorExtrasByNodeID[tx.Validator.NodeID] = &validatorImpl{
				nominators: vdr.nominators,
				subnets:    newSubnets,
			}
		} else {
			newPS.validatorExtrasByNodeID[tx.Validator.NodeID] = &validatorImpl{
				subnets: map[ids.ID]*UnsignedAddSubnetValidatorTx{
					tx.Validator.Subnet: tx,
				},
			}
		}
	default:
		panic(fmt.Errorf("expected staker tx type but got %T", addStakerTx.UnsignedTx))
	}

	return newPS
}

func (ps *pendingStakerChainStateImpl) DeleteStakers(numToRemove int) pendingStakerChainState {
	newPS := &pendingStakerChainStateImpl{
		validatorsByNodeID:      make(map[ids.NodeID]*UnsignedAddValidatorTx, len(ps.validatorsByNodeID)),
		validatorExtrasByNodeID: make(map[ids.NodeID]*validatorImpl, len(ps.validatorExtrasByNodeID)),
		validators:              ps.validators[numToRemove:],

		deletedStakers: ps.validators[:numToRemove],
	}

	for nodeID, vdr := range ps.validatorsByNodeID {
		newPS.validatorsByNodeID[nodeID] = vdr
	}
	for nodeID, vdr := range ps.validatorExtrasByNodeID {
		newPS.validatorExtrasByNodeID[nodeID] = vdr
	}

	for _, removedTx := range ps.validators[:numToRemove] {
		switch tx := removedTx.UnsignedTx.(type) {
		case *UnsignedAddValidatorTx:
			delete(newPS.validatorsByNodeID, tx.Validator.NodeID)
		case *UnsignedAddNominatorTx:
			vdr := newPS.validatorExtrasByNodeID[tx.Validator.NodeID]
			if len(vdr.nominators) == 1 && len(vdr.subnets) == 0 {
				delete(newPS.validatorExtrasByNodeID, tx.Validator.NodeID)
				break
			}
			newPS.validatorExtrasByNodeID[tx.Validator.NodeID] = &validatorImpl{
				nominators: vdr.nominators[1:], // sorted in order of removal
				subnets:    vdr.subnets,
			}
		case *UnsignedAddSubnetValidatorTx:
			vdr := newPS.validatorExtrasByNodeID[tx.Validator.NodeID]
			if len(vdr.nominators) == 0 && len(vdr.subnets) == 1 {
				delete(newPS.validatorExtrasByNodeID, tx.Validator.NodeID)
				break
			}
			newSubnets := make(map[ids.ID]*UnsignedAddSubnetValidatorTx, len(vdr.subnets)-1)
			for subnetID, subnetTx := range vdr.subnets {
				if subnetID != tx.Validator.Subnet {
					newSubnets[subnetID] = subnetTx
				}
			}
			newPS.validatorExtrasByNodeID[tx.Validator.NodeID] = &validatorImpl{
				nominators: vdr.nominators,
				subnets:    newSubnets,
			}
		default:
			panic(fmt.Errorf("expected staker tx type but got %T", removedTx.UnsignedTx))
		}
	}

	return newPS
}

func (ps *pendingStakerChainStateImpl) Stakers() []*Tx {
	return ps.validators
}

func (ps *pendingStakerChainStateImpl) Apply(is InternalState) {
	for _, added := range ps.addedStakers {
		is.AddPendingStaker(added)
	}
	for _, deleted := range ps.deletedStakers {
		is.DeletePendingStaker(deleted)
	}
	is.SetPendingStakerChainState(ps)

	// Validator changes should only be applied once.
	ps.addedStakers = nil
	ps.deletedStakers = nil
}

type innerSortValidatorsByAddition []*Tx

func (s innerSortValidatorsByAddition) Less(i, j int) bool {
	iDel := s[i]
	jDel := s[j]

	var (
		iStartTime time.Time
		iPriority  byte
	)
	switch tx := iDel.UnsignedTx.(type) {
	case *UnsignedAddValidatorTx:
		iStartTime = tx.StartTime()
		iPriority = mediumPriority
	case *UnsignedAddNominatorTx:
		iStartTime = tx.StartTime()
		iPriority = topPriority
	case *UnsignedAddSubnetValidatorTx:
		iStartTime = tx.StartTime()
		iPriority = lowPriority
	default:
		panic(fmt.Errorf("expected staker tx type but got %T", iDel.UnsignedTx))
	}

	var (
		jStartTime time.Time
		jPriority  byte
	)
	switch tx := jDel.UnsignedTx.(type) {
	case *UnsignedAddValidatorTx:
		jStartTime = tx.StartTime()
		jPriority = mediumPriority
	case *UnsignedAddNominatorTx:
		jStartTime = tx.StartTime()
		jPriority = topPriority
	case *UnsignedAddSubnetValidatorTx:
		jStartTime = tx.StartTime()
		jPriority = lowPriority
	default:
		panic(fmt.Errorf("expected staker tx type but got %T", jDel.UnsignedTx))
	}

	if iStartTime.Before(jStartTime) {
		return true
	}
	if jStartTime.Before(iStartTime) {
		return false
	}

	// If the end times are the same, then we sort by the tx type. First we
	// add UnsignedAddValidatorTx, then UnsignedAddNominatorTx, then
	// UnsignedAddSubnetValidatorTxs.
	if iPriority > jPriority {
		return true
	}
	if iPriority < jPriority {
		return false
	}

	// If the end times are the same, and the tx types are the same, then we
	// sort by the txID.
	iTxID := iDel.ID()
	jTxID := jDel.ID()
	return bytes.Compare(iTxID[:], jTxID[:]) == -1
}

func (s innerSortValidatorsByAddition) Len() int {
	return len(s)
}

func (s innerSortValidatorsByAddition) Swap(i, j int) {
	s[j], s[i] = s[i], s[j]
}

func sortValidatorsByAddition(s []*Tx) {
	sort.Sort(innerSortValidatorsByAddition(s))
}

type innerSortNominatorsByAddition []*UnsignedAddNominatorTx

func (s innerSortNominatorsByAddition) Less(i, j int) bool {
	iDel := s[i]
	jDel := s[j]

	iStartTime := iDel.StartTime()
	jStartTime := jDel.StartTime()
	if iStartTime.Before(jStartTime) {
		return true
	}
	if jStartTime.Before(iStartTime) {
		return false
	}

	// If the end times are the same, then we sort by the txID
	iTxID := iDel.ID()
	jTxID := jDel.ID()
	return bytes.Compare(iTxID[:], jTxID[:]) == -1
}

func (s innerSortNominatorsByAddition) Len() int {
	return len(s)
}

func (s innerSortNominatorsByAddition) Swap(i, j int) {
	s[j], s[i] = s[i], s[j]
}

func sortNominatorsByAddition(s []*UnsignedAddNominatorTx) {
	sort.Sort(innerSortNominatorsByAddition(s))
}
