// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package proposervm

import (
	"fmt"

	"github.com/sankar-boro/avalanchego/database"
	"github.com/sankar-boro/avalanchego/snow/engine/snowman/block"
	"github.com/sankar-boro/avalanchego/vms/proposervm/summary"
)

func (vm *VM) StateSyncEnabled() (bool, error) {
	if vm.ssVM == nil {
		return false, nil
	}

	// if vm implements Snowman++, a block height index must be available
	// to support state sync
	if vm.VerifyHeightIndex() != nil {
		return false, nil
	}

	return vm.ssVM.StateSyncEnabled()
}

func (vm *VM) GetOngoingSyncStateSummary() (block.StateSummary, error) {
	if vm.ssVM == nil {
		return nil, block.ErrStateSyncableVMNotImplemented
	}

	innerSummary, err := vm.ssVM.GetOngoingSyncStateSummary()
	if err != nil {
		return nil, err // includes database.ErrNotFound case
	}

	return vm.buildStateSummary(innerSummary)
}

func (vm *VM) GetLastStateSummary() (block.StateSummary, error) {
	if vm.ssVM == nil {
		return nil, block.ErrStateSyncableVMNotImplemented
	}

	// Extract inner vm's last state summary
	innerSummary, err := vm.ssVM.GetLastStateSummary()
	if err != nil {
		return nil, err // including database.ErrNotFound case
	}

	return vm.buildStateSummary(innerSummary)
}

// Note: it's important that ParseStateSummary do not use any index or state
// to allow summaries being parsed also by freshly started node with no previous state.
func (vm *VM) ParseStateSummary(summaryBytes []byte) (block.StateSummary, error) {
	if vm.ssVM == nil {
		return nil, block.ErrStateSyncableVMNotImplemented
	}

	statelessSummary, err := summary.Parse(summaryBytes)
	if err != nil {
		// it may be a preFork summary
		return vm.ssVM.ParseStateSummary(summaryBytes)
	}

	innerSummary, err := vm.ssVM.ParseStateSummary(statelessSummary.InnerSummaryBytes())
	if err != nil {
		return nil, fmt.Errorf("could not parse inner summary due to: %w", err)
	}
	block, err := vm.parsePostForkBlock(statelessSummary.BlockBytes())
	if err != nil {
		return nil, fmt.Errorf("could not parse proposervm block bytes from summary due to: %w", err)
	}

	return &stateSummary{
		StateSummary: statelessSummary,
		innerSummary: innerSummary,
		block:        block,
		vm:           vm,
	}, nil
}

func (vm *VM) GetStateSummary(height uint64) (block.StateSummary, error) {
	if vm.ssVM == nil {
		return nil, block.ErrStateSyncableVMNotImplemented
	}

	innerSummary, err := vm.ssVM.GetStateSummary(height)
	if err != nil {
		return nil, err // including database.ErrNotFound case
	}

	return vm.buildStateSummary(innerSummary)
}

// Note: building state summary requires a well formed height index.
func (vm *VM) buildStateSummary(innerSummary block.StateSummary) (block.StateSummary, error) {
	// if vm implements Snowman++, a block height index must be available
	// to support state sync
	if err := vm.VerifyHeightIndex(); err != nil {
		return nil, fmt.Errorf("could not build state summary: %w", err)
	}

	forkHeight, err := vm.GetForkHeight()
	switch err {
	case nil:
		if innerSummary.Height() < forkHeight {
			return innerSummary, nil
		}
	case database.ErrNotFound:
		// fork has not been reached since there is not fork height
		// just return innerSummary
		vm.ctx.Log.Debug(
			"built pre-fork summary, ID: %s, height: %d",
			innerSummary.ID(),
			innerSummary.Height(),
		)
		return innerSummary, nil
	default:
		return nil, err
	}

	height := innerSummary.Height()
	blkID, err := vm.GetBlockIDAtHeight(height)
	if err != nil {
		vm.ctx.Log.Debug("failed to fetch proposervm block ID at height %d with %s", height, err)
		return nil, err
	}
	block, err := vm.getPostForkBlock(blkID)
	if err != nil {
		vm.ctx.Log.Warn("failed to fetch proposervm block %s at height %d with %s", blkID, height, err)
		return nil, err
	}

	statelessSummary, err := summary.Build(forkHeight, block.Bytes(), innerSummary.Bytes())
	if err != nil {
		return nil, err
	}

	vm.ctx.Log.Debug(
		"built post-fork summary, ID: %s, height: %d",
		statelessSummary.ID(),
		forkHeight,
	)
	return &stateSummary{
		StateSummary: statelessSummary,
		innerSummary: innerSummary,
		block:        block,
		vm:           vm,
	}, nil
}
