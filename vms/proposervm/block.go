// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package proposervm

import (
	"errors"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/vms/proposervm/block"
	"github.com/ava-labs/avalanchego/vms/proposervm/proposer"
)

const (
	// allowable block issuance in the future
	maxSkew = 10 * time.Second
)

var (
	errUnsignedChild            = errors.New("expected child to be signed")
	errUnexpectedBlockType      = errors.New("unexpected proposer block type")
	errInnerParentMismatch      = errors.New("inner parentID didn't match expected parent")
	errTimeNotMonotonic         = errors.New("time must monotonically increase")
	errCoreChainHeightNotMonotonic = errors.New("non monotonically increasing Core-chain height")
	errCoreChainHeightNotReached   = errors.New("block Core-chain height larger than current Core-chain height")
	errTimeTooAdvanced          = errors.New("time is too far advanced")
	errProposerWindowNotStarted = errors.New("proposer window hasn't started")
	errProposersNotActivated    = errors.New("proposers haven't been activated yet")
	errCoreChainHeightTooLow       = errors.New("block Core-chain height is too low")
)

type Block interface {
	snowman.Block

	getInnerBlk() snowman.Block

	// After a state sync, we may need to update last accepted block data
	// without propagating any changes to the innerVM.
	// acceptOuterBlk and acceptInnerBlk allow controlling acceptance of outer
	// and inner blocks.
	acceptOuterBlk() error
	acceptInnerBlk() error

	verifyPreForkChild(child *preForkBlock) error
	verifyPostForkChild(child *postForkBlock) error
	verifyPostForkOption(child *postForkOption) error

	buildChild() (Block, error)

	coreChainHeight() (uint64, error)
}

type PostForkBlock interface {
	Block

	setStatus(choices.Status)
	getStatelessBlk() block.Block
	setInnerBlk(snowman.Block)
}

// field of postForkBlock and postForkOption
type postForkCommonComponents struct {
	vm       *VM
	innerBlk snowman.Block
	status   choices.Status
}

// Return the inner block's height
func (p *postForkCommonComponents) Height() uint64 {
	return p.innerBlk.Height()
}

// Verify returns nil if:
// 1) [p]'s inner block is not an oracle block
// 2) [child]'s Core-Chain height >= [parentCoreChainHeight]
// 3) [p]'s inner block is the parent of [c]'s inner block
// 4) [child]'s timestamp isn't before [p]'s timestamp
// 5) [child]'s timestamp is within the skew bound
// 6) [childCoreChainHeight] <= the current Core-Chain height
// 7) [child]'s timestamp is within its proposer's window
// 8) [child] has a valid signature from its proposer
// 9) [child]'s inner block is valid
func (p *postForkCommonComponents) Verify(parentTimestamp time.Time, parentCoreChainHeight uint64, child *postForkBlock) error {
	if err := verifyIsNotOracleBlock(p.innerBlk); err != nil {
		return err
	}

	childCoreChainHeight := child.CoreChainHeight()
	if childCoreChainHeight < parentCoreChainHeight {
		return errCoreChainHeightNotMonotonic
	}

	expectedInnerParentID := p.innerBlk.ID()
	innerParentID := child.innerBlk.Parent()
	if innerParentID != expectedInnerParentID {
		return errInnerParentMismatch
	}

	childTimestamp := child.Timestamp()
	if childTimestamp.Before(parentTimestamp) {
		return errTimeNotMonotonic
	}

	maxTimestamp := p.vm.Time().Add(maxSkew)
	if childTimestamp.After(maxTimestamp) {
		return errTimeTooAdvanced
	}

	// If the node is currently syncing - we don't assume that the Core-chain has
	// been synced up to this point yet.
	if p.vm.consensusState == snow.NormalOp {
		childID := child.ID()
		currentCoreChainHeight, err := p.vm.ctx.ValidatorState.GetCurrentHeight()
		if err != nil {
			p.vm.ctx.Log.Error("failed to get current Core-Chain height while processing %s: %s",
				childID, err)
			return err
		}
		if childCoreChainHeight > currentCoreChainHeight {
			return errCoreChainHeightNotReached
		}

		childHeight := child.Height()
		proposerID := child.Proposer()
		minDelay, err := p.vm.Windower.Delay(childHeight, parentCoreChainHeight, proposerID)
		if err != nil {
			return err
		}

		delay := childTimestamp.Sub(parentTimestamp)
		if delay < minDelay {
			return errProposerWindowNotStarted
		}

		// Verify the signature of the node
		shouldHaveProposer := delay < proposer.MaxDelay
		if err := child.SignedBlock.Verify(shouldHaveProposer, p.vm.ctx.ChainID); err != nil {
			return err
		}

		p.vm.ctx.Log.Debug("verified post-fork block %s - parent timestamp %v, expected delay %v, block timestamp %v",
			childID, parentTimestamp, minDelay, childTimestamp)
	}

	return p.vm.verifyAndRecordInnerBlk(child)
}

// Return the child (a *postForkBlock) of this block
func (p *postForkCommonComponents) buildChild(
	parentID ids.ID,
	parentTimestamp time.Time,
	parentCoreChainHeight uint64,
) (Block, error) {
	// Child's timestamp is the later of now and this block's timestamp
	newTimestamp := p.vm.Time().Truncate(time.Second)
	if newTimestamp.Before(parentTimestamp) {
		newTimestamp = parentTimestamp
	}

	// The child's Core-Chain height is proposed as the optimal Core-Chain height that
	// is at least the parent's Core-Chain height
	coreChainHeight, err := p.vm.optimalCoreChainHeight(parentCoreChainHeight)
	if err != nil {
		return nil, err
	}

	delay := newTimestamp.Sub(parentTimestamp)
	if delay < proposer.MaxDelay {
		parentHeight := p.innerBlk.Height()
		proposerID := p.vm.ctx.NodeID
		minDelay, err := p.vm.Windower.Delay(parentHeight+1, parentCoreChainHeight, proposerID)
		if err != nil {
			return nil, err
		}

		if delay < minDelay {
			// It's not our turn to propose a block yet. This is likely caused
			// by having previously notified the consensus engine to attempt to
			// build a block on top of a block that is no longer the preferred
			// block.
			p.vm.ctx.Log.Debug("build block dropped; parent timestamp %s, expected delay %s, block timestamp %s",
				parentTimestamp, minDelay, newTimestamp)

			// In case the inner VM only issued one pendingTxs message, we
			// should attempt to re-handle that once it is our turn to build the
			// block.
			p.vm.notifyInnerBlockReady()
			return nil, errProposerWindowNotStarted
		}
	}

	innerBlock, err := p.vm.ChainVM.BuildBlock()
	if err != nil {
		return nil, err
	}

	// Build the child
	var statelessChild block.SignedBlock
	if delay >= proposer.MaxDelay {
		statelessChild, err = block.BuildUnsigned(
			parentID,
			newTimestamp,
			coreChainHeight,
			innerBlock.Bytes(),
		)
		if err != nil {
			return nil, err
		}
	} else {
		statelessChild, err = block.Build(
			parentID,
			newTimestamp,
			coreChainHeight,
			p.vm.ctx.StakingCertLeaf,
			innerBlock.Bytes(),
			p.vm.ctx.ChainID,
			p.vm.ctx.StakingLeafSigner,
		)
		if err != nil {
			return nil, err
		}
	}

	child := &postForkBlock{
		SignedBlock: statelessChild,
		postForkCommonComponents: postForkCommonComponents{
			vm:       p.vm,
			innerBlk: innerBlock,
			status:   choices.Processing,
		},
	}

	p.vm.ctx.Log.Info("built block %s - parent timestamp %v, block timestamp %v",
		child.ID(), parentTimestamp, newTimestamp)
	return child, nil
}

func (p *postForkCommonComponents) getInnerBlk() snowman.Block {
	return p.innerBlk
}

func (p *postForkCommonComponents) setInnerBlk(innerBlk snowman.Block) {
	p.innerBlk = innerBlk
}

func verifyIsOracleBlock(b snowman.Block) error {
	oracle, ok := b.(snowman.OracleBlock)
	if !ok {
		return errUnexpectedBlockType
	}
	_, err := oracle.Options()
	return err
}

func verifyIsNotOracleBlock(b snowman.Block) error {
	oracle, ok := b.(snowman.OracleBlock)
	if !ok {
		return nil
	}
	_, err := oracle.Options()
	switch err {
	case nil:
		return errUnexpectedBlockType
	case snowman.ErrNotOracle:
		return nil
	default:
		return err
	}
}
