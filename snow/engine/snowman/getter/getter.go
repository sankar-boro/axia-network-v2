// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package getter

import (
	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow/choices"
	"github.com/sankar-boro/axia/snow/engine/common"
	"github.com/sankar-boro/axia/snow/engine/snowman/block"
	"github.com/sankar-boro/axia/utils/constants"
	"github.com/sankar-boro/axia/utils/logging"
	"github.com/sankar-boro/axia/utils/metric"
)

// Get requests are always served, regardless node state (bootstrapping or normal operations).
var _ common.AllGetsServer = &getter{}

func New(
	vm block.ChainVM,
	commonCfg common.Config,
) (common.AllGetsServer, error) {
	ssVM, _ := vm.(block.StateSyncableVM)
	gh := &getter{
		vm:     vm,
		ssVM:   ssVM,
		sender: commonCfg.Sender,
		cfg:    commonCfg,
		log:    commonCfg.Ctx.Log,
	}

	var err error
	gh.getAncestorsBlks, err = metric.NewAverager(
		"bs",
		"get_ancestors_blks",
		"blocks fetched in a call to GetAncestors",
		commonCfg.Ctx.Registerer,
	)
	return gh, err
}

type getter struct {
	vm     block.ChainVM
	ssVM   block.StateSyncableVM // can be nil
	sender common.Sender
	cfg    common.Config

	log              logging.Logger
	getAncestorsBlks metric.Averager
}

func (gh *getter) GetStateSummaryFrontier(nodeID ids.NodeID, requestID uint32) error {
	// Note: we do not check if gh.ssVM.StateSyncEnabled since we want all
	// nodes, including those disabling state sync to serve state summaries if
	// these are available
	if gh.ssVM == nil {
		gh.log.Debug("state sync not supported. Dropping GetStateSummaryFrontier(%s, %d)", nodeID, requestID)
		return nil
	}

	summary, err := gh.ssVM.GetLastStateSummary()
	if err != nil {
		gh.log.Debug("couldn't get state summary frontier with %s. Dropping GetStateSummaryFrontier(%s, %d)",
			err, nodeID, requestID)
		return nil
	}

	gh.sender.SendStateSummaryFrontier(nodeID, requestID, summary.Bytes())
	return nil
}

func (gh *getter) GetAcceptedStateSummary(nodeID ids.NodeID, requestID uint32, heights []uint64) error {
	// If there are no requested heights, then we can return the result
	// immediately, regardless of if the underlying VM implements state sync.
	if len(heights) == 0 {
		gh.sender.SendAcceptedStateSummary(nodeID, requestID, nil)
		return nil
	}

	// Note: we do not check if gh.ssVM.StateSyncEnabled since we want all
	// nodes, including those disabling state sync to serve state summaries if
	// these are available
	if gh.ssVM == nil {
		gh.log.Debug("state sync not supported. Dropping GetAcceptedStateSummary(%s, %d)",
			nodeID, requestID)
		return nil
	}

	summaryIDs := make([]ids.ID, 0, len(heights))
	for _, height := range heights {
		summary, err := gh.ssVM.GetStateSummary(height)
		if err == block.ErrStateSyncableVMNotImplemented {
			gh.log.Debug("state sync not supported. Dropping GetAcceptedStateSummary(%s, %d)",
				nodeID, requestID)
			return nil
		}
		if err != nil {
			gh.log.Debug("couldn't get state summary with height %d due to %s",
				height, err)
			continue
		}
		summaryIDs = append(summaryIDs, summary.ID())
	}

	gh.sender.SendAcceptedStateSummary(nodeID, requestID, summaryIDs)
	return nil
}

func (gh *getter) GetAcceptedFrontier(nodeID ids.NodeID, requestID uint32) error {
	lastAccepted, err := gh.vm.LastAccepted()
	if err != nil {
		return err
	}
	gh.sender.SendAcceptedFrontier(nodeID, requestID, []ids.ID{lastAccepted})
	return nil
}

func (gh *getter) GetAccepted(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) error {
	acceptedIDs := make([]ids.ID, 0, len(containerIDs))
	for _, blkID := range containerIDs {
		if blk, err := gh.vm.GetBlock(blkID); err == nil && blk.Status() == choices.Accepted {
			acceptedIDs = append(acceptedIDs, blkID)
		}
	}
	gh.sender.SendAccepted(nodeID, requestID, acceptedIDs)
	return nil
}

func (gh *getter) GetAncestors(nodeID ids.NodeID, requestID uint32, blkID ids.ID) error {
	ancestorsBytes, err := block.GetAncestors(
		gh.vm,
		blkID,
		gh.cfg.AncestorsMaxContainersSent,
		constants.MaxContainersLen,
		gh.cfg.MaxTimeGetAncestors,
	)
	if err != nil {
		gh.log.Verbo("couldn't get ancestors with %s. Dropping GetAncestors(%s, %d, %s)",
			err, nodeID, requestID, blkID)
		return nil
	}

	gh.getAncestorsBlks.Observe(float64(len(ancestorsBytes)))
	gh.sender.SendAncestors(nodeID, requestID, ancestorsBytes)
	return nil
}

func (gh *getter) Get(nodeID ids.NodeID, requestID uint32, blkID ids.ID) error {
	blk, err := gh.vm.GetBlock(blkID)
	if err != nil {
		// If we failed to get the block, that means either an unexpected error
		// has occurred, [vdr] is not following the protocol, or the
		// block has been pruned.
		gh.log.Debug("Get(%s, %d, %s) failed with: %s", nodeID, requestID, blkID, err)
		return nil
	}

	// Respond to the validator with the fetched block and the same requestID.
	gh.sender.SendPut(nodeID, requestID, blkID, blk.Bytes())
	return nil
}
