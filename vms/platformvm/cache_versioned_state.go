// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"time"

	"github.com/sankar-boro/axia-network-v2/database"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm/status"
)

var _ VersionedState = &versionedStateImpl{}

type UTXOGetter interface {
	GetUTXO(utxoID ids.ID) (*axc.UTXO, error)
}

type UTXOAdder interface {
	AddUTXO(utxo *axc.UTXO)
}

type UTXODeleter interface {
	DeleteUTXO(utxoID ids.ID)
}

type UTXOState interface {
	UTXOGetter
	UTXOAdder
	UTXODeleter
}

type MutableState interface {
	UTXOState
	ValidatorState

	AddRewardUTXO(txID ids.ID, utxo *axc.UTXO)
	GetRewardUTXOs(txID ids.ID) ([]*axc.UTXO, error)

	GetTimestamp() time.Time
	SetTimestamp(time.Time)

	GetCurrentSupply() uint64
	SetCurrentSupply(uint64)

	GetAllychains() ([]*Tx, error)
	AddAllychain(createAllychainTx *Tx)

	GetChains(allychainID ids.ID) ([]*Tx, error)
	AddChain(createChainTx *Tx)

	GetTx(txID ids.ID) (*Tx, status.Status, error)
	AddTx(tx *Tx, status status.Status)
}

type VersionedState interface {
	MutableState

	SetBase(MutableState)
	Apply(InternalState)
}

type versionedStateImpl struct {
	parentState MutableState

	currentStakerChainState currentStakerChainState
	pendingStakerChainState pendingStakerChainState

	timestamp time.Time

	currentSupply uint64

	addedAllychains  []*Tx
	cachedAllychains []*Tx

	addedChains  map[ids.ID][]*Tx
	cachedChains map[ids.ID][]*Tx

	// map of txID -> []*UTXO
	addedRewardUTXOs map[ids.ID][]*axc.UTXO

	// map of txID -> {*Tx, Status}
	addedTxs map[ids.ID]*txStatusImpl

	// map of modified UTXOID -> *UTXO if the UTXO is nil, it has been removed
	modifiedUTXOs map[ids.ID]*utxoImpl
}

type txStatusImpl struct {
	tx     *Tx
	status status.Status
}

type utxoImpl struct {
	utxoID ids.ID
	utxo   *axc.UTXO
}

func newVersionedState(
	ps MutableState,
	current currentStakerChainState,
	pending pendingStakerChainState,
) VersionedState {
	return &versionedStateImpl{
		parentState:             ps,
		currentStakerChainState: current,
		pendingStakerChainState: pending,
		timestamp:               ps.GetTimestamp(),
		currentSupply:           ps.GetCurrentSupply(),
	}
}

func (vs *versionedStateImpl) GetTimestamp() time.Time {
	return vs.timestamp
}

func (vs *versionedStateImpl) SetTimestamp(timestamp time.Time) {
	vs.timestamp = timestamp
}

func (vs *versionedStateImpl) GetCurrentSupply() uint64 {
	return vs.currentSupply
}

func (vs *versionedStateImpl) SetCurrentSupply(currentSupply uint64) {
	vs.currentSupply = currentSupply
}

func (vs *versionedStateImpl) GetAllychains() ([]*Tx, error) {
	if len(vs.addedAllychains) == 0 {
		return vs.parentState.GetAllychains()
	}
	if len(vs.cachedAllychains) != 0 {
		return vs.cachedAllychains, nil
	}
	allychains, err := vs.parentState.GetAllychains()
	if err != nil {
		return nil, err
	}
	newAllychains := make([]*Tx, len(allychains)+len(vs.addedAllychains))
	copy(newAllychains, allychains)
	for i, allychain := range vs.addedAllychains {
		newAllychains[i+len(allychains)] = allychain
	}
	vs.cachedAllychains = newAllychains
	return newAllychains, nil
}

func (vs *versionedStateImpl) AddAllychain(createAllychainTx *Tx) {
	vs.addedAllychains = append(vs.addedAllychains, createAllychainTx)
	if vs.cachedAllychains != nil {
		vs.cachedAllychains = append(vs.cachedAllychains, createAllychainTx)
	}
}

func (vs *versionedStateImpl) GetChains(allychainID ids.ID) ([]*Tx, error) {
	if len(vs.addedChains) == 0 {
		// No chains have been added
		return vs.parentState.GetChains(allychainID)
	}
	addedChains := vs.addedChains[allychainID]
	if len(addedChains) == 0 {
		// No chains have been added to this allychain
		return vs.parentState.GetChains(allychainID)
	}

	// There have been chains added to the requested allychain

	if vs.cachedChains == nil {
		// This is the first time we are going to be caching the allychain chains
		vs.cachedChains = make(map[ids.ID][]*Tx)
	}

	cachedChains, cached := vs.cachedChains[allychainID]
	if cached {
		return cachedChains, nil
	}

	// This chain wasn't cached yet
	chains, err := vs.parentState.GetChains(allychainID)
	if err != nil {
		return nil, err
	}

	newChains := make([]*Tx, len(chains)+len(addedChains))
	copy(newChains, chains)
	for i, chain := range addedChains {
		newChains[i+len(chains)] = chain
	}
	vs.cachedChains[allychainID] = newChains
	return newChains, nil
}

func (vs *versionedStateImpl) AddChain(createChainTx *Tx) {
	tx := createChainTx.UnsignedTx.(*UnsignedCreateChainTx)
	if vs.addedChains == nil {
		vs.addedChains = map[ids.ID][]*Tx{
			tx.AllychainID: {createChainTx},
		}
	} else {
		vs.addedChains[tx.AllychainID] = append(vs.addedChains[tx.AllychainID], createChainTx)
	}

	cachedChains, cached := vs.cachedChains[tx.AllychainID]
	if !cached {
		return
	}
	vs.cachedChains[tx.AllychainID] = append(cachedChains, createChainTx)
}

func (vs *versionedStateImpl) GetTx(txID ids.ID) (*Tx, status.Status, error) {
	tx, exists := vs.addedTxs[txID]
	if !exists {
		return vs.parentState.GetTx(txID)
	}
	return tx.tx, tx.status, nil
}

func (vs *versionedStateImpl) AddTx(tx *Tx, status status.Status) {
	txID := tx.ID()
	txStatus := &txStatusImpl{
		tx:     tx,
		status: status,
	}
	if vs.addedTxs == nil {
		vs.addedTxs = map[ids.ID]*txStatusImpl{
			txID: txStatus,
		}
	} else {
		vs.addedTxs[txID] = txStatus
	}
}

func (vs *versionedStateImpl) GetRewardUTXOs(txID ids.ID) ([]*axc.UTXO, error) {
	if utxos, exists := vs.addedRewardUTXOs[txID]; exists {
		return utxos, nil
	}
	return vs.parentState.GetRewardUTXOs(txID)
}

func (vs *versionedStateImpl) AddRewardUTXO(txID ids.ID, utxo *axc.UTXO) {
	if vs.addedRewardUTXOs == nil {
		vs.addedRewardUTXOs = make(map[ids.ID][]*axc.UTXO)
	}
	vs.addedRewardUTXOs[txID] = append(vs.addedRewardUTXOs[txID], utxo)
}

func (vs *versionedStateImpl) GetUTXO(utxoID ids.ID) (*axc.UTXO, error) {
	utxo, modified := vs.modifiedUTXOs[utxoID]
	if !modified {
		return vs.parentState.GetUTXO(utxoID)
	}
	if utxo.utxo == nil {
		return nil, database.ErrNotFound
	}
	return utxo.utxo, nil
}

func (vs *versionedStateImpl) AddUTXO(utxo *axc.UTXO) {
	newUTXO := &utxoImpl{
		utxoID: utxo.InputID(),
		utxo:   utxo,
	}
	if vs.modifiedUTXOs == nil {
		vs.modifiedUTXOs = map[ids.ID]*utxoImpl{
			utxo.InputID(): newUTXO,
		}
	} else {
		vs.modifiedUTXOs[utxo.InputID()] = newUTXO
	}
}

func (vs *versionedStateImpl) DeleteUTXO(utxoID ids.ID) {
	newUTXO := &utxoImpl{
		utxoID: utxoID,
	}
	if vs.modifiedUTXOs == nil {
		vs.modifiedUTXOs = map[ids.ID]*utxoImpl{
			utxoID: newUTXO,
		}
	} else {
		vs.modifiedUTXOs[utxoID] = newUTXO
	}
}

func (vs *versionedStateImpl) CurrentStakerChainState() currentStakerChainState {
	return vs.currentStakerChainState
}

func (vs *versionedStateImpl) PendingStakerChainState() pendingStakerChainState {
	return vs.pendingStakerChainState
}

func (vs *versionedStateImpl) SetBase(parentState MutableState) {
	vs.parentState = parentState
}

func (vs *versionedStateImpl) Apply(is InternalState) {
	is.SetTimestamp(vs.timestamp)
	is.SetCurrentSupply(vs.currentSupply)
	for _, allychain := range vs.addedAllychains {
		is.AddAllychain(allychain)
	}
	for _, chains := range vs.addedChains {
		for _, chain := range chains {
			is.AddChain(chain)
		}
	}
	for _, tx := range vs.addedTxs {
		is.AddTx(tx.tx, tx.status)
	}
	for txID, utxos := range vs.addedRewardUTXOs {
		for _, utxo := range utxos {
			is.AddRewardUTXO(txID, utxo)
		}
	}
	for _, utxo := range vs.modifiedUTXOs {
		if utxo.utxo != nil {
			is.AddUTXO(utxo.utxo)
		} else {
			is.DeleteUTXO(utxo.utxoID)
		}
	}
	vs.currentStakerChainState.Apply(is)
	vs.pendingStakerChainState.Apply(is)
}
