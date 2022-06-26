// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/avalanchego/database"
	"github.com/sankar-boro/avalanchego/ids"
	"github.com/sankar-boro/avalanchego/snow/engine/common"
	"github.com/sankar-boro/avalanchego/snow/engine/snowman/block"
	"github.com/sankar-boro/avalanchego/snow/engine/snowman/getter"
	"github.com/sankar-boro/avalanchego/snow/validators"
	"github.com/sankar-boro/avalanchego/utils/hashing"
)

const (
	key         uint64 = 2022
	minorityKey uint64 = 2000
)

var (
	_ block.ChainVM         = fullVM{}
	_ block.StateSyncableVM = fullVM{}

	unknownSummaryID = ids.ID{'g', 'a', 'r', 'b', 'a', 'g', 'e'}

	summaryBytes = []byte{'s', 'u', 'm', 'm', 'a', 'r', 'y'}
	summaryID    ids.ID

	minoritySummaryBytes = []byte{'m', 'i', 'n', 'o', 'r', 'i', 't', 'y'}
	minoritySummaryID    ids.ID
)

func init() {
	var err error
	summaryID, err = ids.ToID(hashing.ComputeHash256(summaryBytes))
	if err != nil {
		panic(err)
	}

	minoritySummaryID, err = ids.ToID(hashing.ComputeHash256(minoritySummaryBytes))
	if err != nil {
		panic(err)
	}
}

type fullVM struct {
	*block.TestVM
	*block.TestStateSyncableVM
}

func buildTestPeers(t *testing.T) validators.Set {
	// we consider more than common.MaxOutstandingBroadcastRequests peers
	// so to test the effect of cap on number of requests sent out
	vdrs := validators.NewSet()
	for idx := 0; idx < 2*common.MaxOutstandingBroadcastRequests; idx++ {
		beaconID := ids.GenerateTestNodeID()
		assert.NoError(t, vdrs.AddWeight(beaconID, uint64(1)))
	}
	return vdrs
}

func buildTestsObjects(t *testing.T, commonCfg *common.Config) (
	*stateSyncer,
	*fullVM,
	*common.SenderTest,

) {
	sender := &common.SenderTest{T: t}
	commonCfg.Sender = sender

	fullVM := &fullVM{
		TestVM: &block.TestVM{
			TestVM: common.TestVM{T: t},
		},
		TestStateSyncableVM: &block.TestStateSyncableVM{
			T: t,
		},
	}
	dummyGetter, err := getter.New(fullVM, *commonCfg)
	assert.NoError(t, err)

	cfg, err := NewConfig(*commonCfg, nil, dummyGetter, fullVM)
	assert.NoError(t, err)
	commonSyncer := New(cfg, func(lastReqID uint32) error { return nil })
	syncer, ok := commonSyncer.(*stateSyncer)
	assert.True(t, ok)
	assert.True(t, syncer.stateSyncVM != nil)

	fullVM.GetOngoingSyncStateSummaryF = func() (block.StateSummary, error) {
		return nil, database.ErrNotFound
	}

	return syncer, fullVM, sender
}

func pickRandomFrom(nodes map[ids.NodeID]uint32) ids.NodeID {
	for node := range nodes {
		return node
	}
	return ids.EmptyNodeID
}
