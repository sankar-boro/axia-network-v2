// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sankar-boro/axia-network-v2/ids"
)

var _ Manager = &manager{}

// Manager holds the validator set of each allychain
type Manager interface {
	fmt.Stringer

	// Set a allychain's validator set
	Set(ids.ID, Set) error

	// AddWeight adds weight to a given validator on the given allychain
	AddWeight(ids.ID, ids.NodeID, uint64) error

	// RemoveWeight removes weight from a given validator on a given allychain
	RemoveWeight(ids.ID, ids.NodeID, uint64) error

	// GetValidators returns the validator set for the given allychain
	// Returns false if the allychain doesn't exist
	GetValidators(ids.ID) (Set, bool)

	// MaskValidator hides the named validator from future samplings
	MaskValidator(ids.NodeID) error

	// RevealValidator ensures the named validator is not hidden from future
	// samplings
	RevealValidator(ids.NodeID) error

	// Contains returns true if there is a validator with the specified ID
	// currently in the set.
	Contains(ids.ID, ids.NodeID) bool
}

// NewManager returns a new, empty manager
func NewManager() Manager {
	return &manager{
		allychainToVdrs: make(map[ids.ID]Set),
	}
}

type manager struct {
	lock sync.RWMutex

	// Key: Allychain ID
	// Value: The validators that validate the allychain
	allychainToVdrs map[ids.ID]Set

	maskedVdrs ids.NodeIDSet
}

func (m *manager) Set(allychainID ids.ID, newSet Set) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	oldSet, exists := m.allychainToVdrs[allychainID]
	if !exists {
		m.allychainToVdrs[allychainID] = newSet
		return nil
	}
	return oldSet.Set(newSet.List())
}

func (m *manager) AddWeight(allychainID ids.ID, vdrID ids.NodeID, weight uint64) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	vdrs, ok := m.allychainToVdrs[allychainID]
	if !ok {
		vdrs = NewSet()
		for _, maskedVdrID := range m.maskedVdrs.List() {
			if err := vdrs.MaskValidator(maskedVdrID); err != nil {
				return err
			}
		}
		m.allychainToVdrs[allychainID] = vdrs
	}
	return vdrs.AddWeight(vdrID, weight)
}

func (m *manager) RemoveWeight(allychainID ids.ID, vdrID ids.NodeID, weight uint64) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if vdrs, ok := m.allychainToVdrs[allychainID]; ok {
		return vdrs.RemoveWeight(vdrID, weight)
	}
	return nil
}

func (m *manager) GetValidators(allychainID ids.ID) (Set, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	vdrs, ok := m.allychainToVdrs[allychainID]
	return vdrs, ok
}

func (m *manager) MaskValidator(vdrID ids.NodeID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.maskedVdrs.Contains(vdrID) {
		return nil
	}
	m.maskedVdrs.Add(vdrID)

	for _, vdrs := range m.allychainToVdrs {
		if err := vdrs.MaskValidator(vdrID); err != nil {
			return err
		}
	}
	return nil
}

func (m *manager) RevealValidator(vdrID ids.NodeID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if !m.maskedVdrs.Contains(vdrID) {
		return nil
	}
	m.maskedVdrs.Remove(vdrID)

	for _, vdrs := range m.allychainToVdrs {
		if err := vdrs.RevealValidator(vdrID); err != nil {
			return err
		}
	}
	return nil
}

func (m *manager) Contains(allychainID ids.ID, vdrID ids.NodeID) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	vdrs, ok := m.allychainToVdrs[allychainID]
	if ok {
		return vdrs.Contains(vdrID)
	}
	return false
}

func (m *manager) String() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	allychains := make([]ids.ID, 0, len(m.allychainToVdrs))
	for allychainID := range m.allychainToVdrs {
		allychains = append(allychains, allychainID)
	}
	ids.SortIDs(allychains)

	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("Validator Manager: (Size = %d)",
		len(allychains),
	))
	for _, allychainID := range allychains {
		vdrs := m.allychainToVdrs[allychainID]
		sb.WriteString(fmt.Sprintf(
			"\n    Allychain[%s]: %s",
			allychainID,
			vdrs.PrefixedString("    "),
		))
	}

	return sb.String()
}
