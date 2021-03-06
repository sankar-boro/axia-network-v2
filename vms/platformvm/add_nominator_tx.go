// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"errors"
	"fmt"
	"time"

	"github.com/sankar-boro/axia-network-v2/database"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/utils/crypto"
	"github.com/sankar-boro/axia-network-v2/utils/math"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/components/verify"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm/fx"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"

	coreChainValidator "github.com/sankar-boro/axia-network-v2/vms/platformvm/validator"
)

var (
	errNominatorSubset = errors.New("nominator's time range must be a subset of the validator's time range")
	errInvalidState    = errors.New("generated output isn't valid state")
	errOverDelegated   = errors.New("validator would be over delegated")

	_ UnsignedProposalTx = &UnsignedAddNominatorTx{}
	_ TimedTx            = &UnsignedAddNominatorTx{}
)

// UnsignedAddNominatorTx is an unsigned addNominatorTx
type UnsignedAddNominatorTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// Describes the delegatee
	Validator coreChainValidator.Validator `serialize:"true" json:"validator"`
	// Where to send staked tokens when done validating
	Stake []*axc.TransferableOutput `serialize:"true" json:"stake"`
	// Where to send staking rewards when done validating
	RewardsOwner fx.Owner `serialize:"true" json:"rewardsOwner"`
}

// InitCtx sets the FxID fields in the inputs and outputs of this
// [UnsignedAddNominatorTx]. Also sets the [ctx] to the given [vm.ctx] so that
// the addresses can be json marshalled into human readable format
func (tx *UnsignedAddNominatorTx) InitCtx(ctx *snow.Context) {
	tx.BaseTx.InitCtx(ctx)
	for _, out := range tx.Stake {
		out.FxID = secp256k1fx.ID
		out.InitCtx(ctx)
	}
	tx.RewardsOwner.InitCtx(ctx)
}

// StartTime of this validator
func (tx *UnsignedAddNominatorTx) StartTime() time.Time {
	return tx.Validator.StartTime()
}

// EndTime of this validator
func (tx *UnsignedAddNominatorTx) EndTime() time.Time {
	return tx.Validator.EndTime()
}

// Weight of this validator
func (tx *UnsignedAddNominatorTx) Weight() uint64 {
	return tx.Validator.Weight()
}

// SyntacticVerify returns nil iff [tx] is valid
func (tx *UnsignedAddNominatorTx) SyntacticVerify(ctx *snow.Context) error {
	switch {
	case tx == nil:
		return errNilTx
	case tx.syntacticallyVerified: // already passed syntactic verification
		return nil
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := verify.All(&tx.Validator, tx.RewardsOwner); err != nil {
		return fmt.Errorf("failed to verify validator or rewards owner: %w", err)
	}

	totalStakeWeight := uint64(0)
	for _, out := range tx.Stake {
		if err := out.Verify(); err != nil {
			return fmt.Errorf("output verification failed: %w", err)
		}
		newWeight, err := math.Add64(totalStakeWeight, out.Output().Amount())
		if err != nil {
			return err
		}
		totalStakeWeight = newWeight
	}

	switch {
	case !axc.IsSortedTransferableOutputs(tx.Stake, Codec):
		return errOutputsNotSorted
	case totalStakeWeight != tx.Validator.Wght:
		return fmt.Errorf("nominator weight %d is not equal to total stake weight %d", tx.Validator.Wght, totalStakeWeight)
	}

	// cache that this is valid
	tx.syntacticallyVerified = true
	return nil
}

// Attempts to verify this transaction with the provided state.
func (tx *UnsignedAddNominatorTx) SemanticVerify(vm *VM, parentState MutableState, stx *Tx) error {
	startTime := tx.StartTime()
	maxLocalStartTime := vm.clock.Time().Add(maxFutureStartTime)
	if startTime.After(maxLocalStartTime) {
		return errFutureStakeTime
	}

	_, _, err := tx.Execute(vm, parentState, stx)
	// We ignore [errFutureStakeTime] here because an advanceTimeTx will be
	// issued before this transaction is issued.
	if errors.Is(err, errFutureStakeTime) {
		return nil
	}
	return err
}

// Execute this transaction.
func (tx *UnsignedAddNominatorTx) Execute(
	vm *VM,
	parentState MutableState,
	stx *Tx,
) (
	VersionedState,
	VersionedState,
	error,
) {
	// Verify the tx is well-formed
	if err := tx.SyntacticVerify(vm.ctx); err != nil {
		return nil, nil, err
	}

	duration := tx.Validator.Duration()
	switch {
	case duration < vm.MinStakeDuration: // Ensure staking length is not too short
		return nil, nil, errStakeTooShort
	case duration > vm.MaxStakeDuration: // Ensure staking length is not too long
		return nil, nil, errStakeTooLong
	case tx.Validator.Wght < vm.MinNominatorStake:
		// Ensure validator is staking at least the minimum amount
		return nil, nil, errWeightTooSmall
	}

	outs := make([]*axc.TransferableOutput, len(tx.Outs)+len(tx.Stake))
	copy(outs, tx.Outs)
	copy(outs[len(tx.Outs):], tx.Stake)

	currentStakers := parentState.CurrentStakerChainState()
	pendingStakers := parentState.PendingStakerChainState()

	if vm.bootstrapped.GetValue() {
		currentTimestamp := parentState.GetTimestamp()
		// Ensure the proposed validator starts after the current timestamp
		validatorStartTime := tx.StartTime()
		if !currentTimestamp.Before(validatorStartTime) {
			return nil, nil, fmt.Errorf(
				"chain timestamp (%s) not before validator's start time (%s)",
				currentTimestamp,
				validatorStartTime,
			)
		}

		currentValidator, err := currentStakers.GetValidator(tx.Validator.NodeID)
		if err != nil && err != database.ErrNotFound {
			return nil, nil, fmt.Errorf(
				"failed to find whether %s is a validator: %w",
				tx.Validator.NodeID,
				err,
			)
		}

		pendingValidator := pendingStakers.GetValidator(tx.Validator.NodeID)
		pendingNominators := pendingValidator.Nominators()

		var (
			vdrTx                  *UnsignedAddValidatorTx
			currentNominatorWeight uint64
			currentNominators      []*UnsignedAddNominatorTx
		)
		if err == nil {
			// This nominator is attempting to delegate to a currently validing
			// node.
			vdrTx = currentValidator.AddValidatorTx()
			currentNominatorWeight = currentValidator.NominatorWeight()
			currentNominators = currentValidator.Nominators()
		} else {
			// This nominator is attempting to delegate to a node that hasn't
			// started validating yet.
			vdrTx, err = pendingStakers.GetValidatorTx(tx.Validator.NodeID)
			if err != nil {
				if err == database.ErrNotFound {
					return nil, nil, errNominatorSubset
				}
				return nil, nil, fmt.Errorf(
					"failed to find whether %s is a validator: %w",
					tx.Validator.NodeID,
					err,
				)
			}
		}

		// Ensure that the period this nominator delegates is a subset of the
		// time the validator validates.
		if !tx.Validator.BoundedBy(vdrTx.StartTime(), vdrTx.EndTime()) {
			return nil, nil, errNominatorSubset
		}

		// Ensure that the period this nominator delegates wouldn't become over
		// delegated.
		vdrWeight := vdrTx.Weight()
		currentWeight, err := math.Add64(vdrWeight, currentNominatorWeight)
		if err != nil {
			return nil, nil, err
		}

		maximumWeight, err := math.Mul64(MaxValidatorWeightFactor, vdrWeight)
		if err != nil {
			return nil, nil, errStakeOverflow
		}

		if !currentTimestamp.Before(vm.ApricotPhase3Time) {
			maximumWeight = math.Min64(maximumWeight, vm.MaxValidatorStake)
		}

		canDelegate, err := CanDelegate(
			currentNominators,
			pendingNominators,
			tx,
			currentWeight,
			maximumWeight,
		)
		if err != nil {
			return nil, nil, err
		}
		if !canDelegate {
			return nil, nil, errOverDelegated
		}

		// Verify the flowcheck
		if err := vm.semanticVerifySpend(parentState, tx, tx.Ins, outs, stx.Creds, vm.AddStakerTxFee, vm.ctx.AXCAssetID); err != nil {
			return nil, nil, fmt.Errorf("failed semanticVerifySpend: %w", err)
		}

		// Make sure the tx doesn't start too far in the future. This is done
		// last to allow SemanticVerification to explicitly check for this
		// error.
		maxStartTime := currentTimestamp.Add(maxFutureStartTime)
		if validatorStartTime.After(maxStartTime) {
			return nil, nil, errFutureStakeTime
		}
	}

	// Set up the state if this tx is committed
	newlyPendingStakers := pendingStakers.AddStaker(stx)
	onCommitState := newVersionedState(parentState, currentStakers, newlyPendingStakers)

	// Consume the UTXOS
	consumeInputs(onCommitState, tx.Ins)
	// Produce the UTXOS
	txID := tx.ID()
	produceOutputs(onCommitState, txID, vm.ctx.AXCAssetID, tx.Outs)

	// Set up the state if this tx is aborted
	onAbortState := newVersionedState(parentState, currentStakers, pendingStakers)
	// Consume the UTXOS
	consumeInputs(onAbortState, tx.Ins)
	// Produce the UTXOS
	produceOutputs(onAbortState, txID, vm.ctx.AXCAssetID, outs)

	return onCommitState, onAbortState, nil
}

// InitiallyPrefersCommit returns true if the proposed validators start time is
// after the current wall clock time,
func (tx *UnsignedAddNominatorTx) InitiallyPrefersCommit(vm *VM) bool {
	return tx.StartTime().After(vm.clock.Time())
}

// Creates a new transaction
func (vm *VM) newAddNominatorTx(
	stakeAmt, // Amount the nominator stakes
	startTime, // Unix time they start delegating
	endTime uint64, // Unix time they stop delegating
	nodeID ids.NodeID, // ID of the node we are delegating to
	rewardAddress ids.ShortID, // Address to send reward to, if applicable
	keys []*crypto.PrivateKeySECP256K1R, // Keys providing the staked tokens
	changeAddr ids.ShortID, // Address to send change to, if there is any
) (*Tx, error) {
	ins, unlockedOuts, lockedOuts, signers, err := vm.stake(keys, stakeAmt, vm.AddStakerTxFee, changeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate tx inputs/outputs: %w", err)
	}
	// Create the tx
	utx := &UnsignedAddNominatorTx{
		BaseTx: BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    vm.ctx.NetworkID,
			BlockchainID: vm.ctx.ChainID,
			Ins:          ins,
			Outs:         unlockedOuts,
		}},
		Validator: coreChainValidator.Validator{
			NodeID: nodeID,
			Start:  startTime,
			End:    endTime,
			Wght:   stakeAmt,
		},
		Stake: lockedOuts,
		RewardsOwner: &secp256k1fx.OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs:     []ids.ShortID{rewardAddress},
		},
	}
	tx := &Tx{UnsignedTx: utx}
	if err := tx.Sign(Codec, signers); err != nil {
		return nil, err
	}
	return tx, utx.SyntacticVerify(vm.ctx)
}

// CanDelegate returns if the [new] nominator can be added to a validator who
// has [current] and [pending] nominators. [currentStake] is the current amount
// of stake on the validator, include the [current] nominators. [maximumStake]
// is the maximum amount of stake that can be on the validator at any given
// time. It is assumed that the validator without adding [new] does not violate
// [maximumStake].
func CanDelegate(
	current,
	pending []*UnsignedAddNominatorTx, // sorted by next start time first
	new *UnsignedAddNominatorTx,
	currentStake,
	maximumStake uint64,
) (bool, error) {
	maxStake, err := maxStakeAmount(current, pending, new.StartTime(), new.EndTime(), currentStake)
	if err != nil {
		return false, err
	}
	newMaxStake, err := math.Add64(maxStake, new.Validator.Wght)
	if err != nil {
		return false, err
	}
	return newMaxStake <= maximumStake, nil
}

// Return the maximum amount of stake on a node (including delegations) at any
// given time between [startTime] and [endTime] given that:
// * The amount of stake on the node right now is [currentStake]
// * The delegations currently on this node are [current]
// * [current] is sorted in order of increasing delegation end time.
// * The stake delegated in [current] are already included in [currentStake]
// * [startTime] is in the future, and [endTime] > [startTime]
// * The delegations that will be on this node in the future are [pending]
// * The start time of all delegations in [pending] are in the future
// * [pending] is sorted in order of increasing delegation start time
func maxStakeAmount(
	current,
	pending []*UnsignedAddNominatorTx, // sorted by next start time first
	startTime time.Time,
	endTime time.Time,
	currentStake uint64,
) (uint64, error) {
	// Keep track of which nominators should be removed next so that we can
	// efficiently remove nominators and keep the current stake updated.
	toRemoveHeap := coreChainValidator.EndTimeHeap{}
	for _, currentNominator := range current {
		toRemoveHeap.Add(&currentNominator.Validator)
	}

	var (
		err error
		// [maxStake] is the max stake at any point between now [starTime] and [endTime]
		maxStake uint64
	)

	// Calculate what the amount staked will be when each pending delegation
	// starts.
	for _, nextPending := range pending { // Iterates in order of increasing start time
		// Calculate what the amount staked will be when this delegation starts.
		nextPendingStartTime := nextPending.StartTime()

		if nextPendingStartTime.After(endTime) {
			// This delegation starts after [endTime].
			// Since we're calculating the max amount staked in
			// [startTime, endTime], we can stop. (Recall that [pending] is
			// sorted in order of increasing end time.)
			break
		}

		// Subtract from [currentStake] all of the current delegations that will
		// have ended by the time that the delegation [nextPending] starts.
		for toRemoveHeap.Len() > 0 {
			// Get the next current delegation that will end.
			toRemove := toRemoveHeap.Peek()
			toRemoveEndTime := toRemove.EndTime()
			if toRemoveEndTime.After(nextPendingStartTime) {
				break
			}
			// This current delegation [toRemove] ends before [nextPending]
			// starts, so its stake should be subtracted from [currentStake].

			// Changed in AP3:
			// If the new nominator has started, then this current nominator
			// should have an end time that is > [startTime].
			newNominatorHasStartedBeforeFinish := toRemoveEndTime.After(startTime)
			if newNominatorHasStartedBeforeFinish && currentStake > maxStake {
				// Only update [maxStake] if it's after [startTime]
				maxStake = currentStake
			}

			currentStake, err = math.Sub64(currentStake, toRemove.Wght)
			if err != nil {
				return 0, err
			}

			// Changed in AP3:
			// Remove the nominator from the heap and update the heap so that
			// the top of the heap is the next nominator to remove.
			toRemoveHeap.Remove()
		}

		// Add to [currentStake] the stake of this pending nominator to
		// calculate what the stake will be when this pending delegation has
		// started.
		currentStake, err = math.Add64(currentStake, nextPending.Validator.Wght)
		if err != nil {
			return 0, err
		}

		// Changed in AP3:
		// If the new nominator has started, then this pending nominator should
		// have a start time that is >= [startTime]. Otherwise, the nominator
		// hasn't started yet and the [currentStake] shouldn't count towards the
		// [maximumStake] during the nominators delegation period.
		newNominatorHasStarted := !nextPendingStartTime.Before(startTime)
		if newNominatorHasStarted && currentStake > maxStake {
			// Only update [maxStake] if it's after [startTime]
			maxStake = currentStake
		}

		// This pending nominator is a current nominator relative
		// when considering later pending nominators that start late
		toRemoveHeap.Add(&nextPending.Validator)
	}

	// [currentStake] is now the amount staked before the next pending nominator
	// whose start time is after [endTime].

	// If there aren't any nominators that will be added before the end of our
	// delegation period, we should advance through time until our delegation
	// period starts.
	for toRemoveHeap.Len() > 0 {
		toRemove := toRemoveHeap.Peek()
		toRemoveEndTime := toRemove.EndTime()
		if toRemoveEndTime.After(startTime) {
			break
		}

		currentStake, err = math.Sub64(currentStake, toRemove.Wght)
		if err != nil {
			return 0, err
		}

		// Changed in AP3:
		// Remove the nominator from the heap and update the heap so that the
		// top of the heap is the next nominator to remove.
		toRemoveHeap.Remove()
	}

	// We have advanced time to be inside the delegation window.
	// Make sure that the max stake is updated accordingly.
	if currentStake > maxStake {
		maxStake = currentStake
	}
	return maxStake, nil
}

func (vm *VM) maxStakeAmount(
	allychainID ids.ID,
	nodeID ids.NodeID,
	startTime time.Time,
	endTime time.Time,
) (uint64, error) {
	if startTime.After(endTime) {
		return 0, errStartAfterEndTime
	}
	if timestamp := vm.internalState.GetTimestamp(); startTime.Before(timestamp) {
		return 0, errStartTimeTooEarly
	}
	if allychainID == constants.PrimaryNetworkID {
		return vm.maxPrimaryAllychainStakeAmount(nodeID, startTime, endTime)
	}
	return vm.maxAllychainStakeAmount(allychainID, nodeID, startTime, endTime)
}

func (vm *VM) maxAllychainStakeAmount(
	allychainID ids.ID,
	nodeID ids.NodeID,
	startTime time.Time,
	endTime time.Time,
) (uint64, error) {
	var (
		vdrTx  *UnsignedAddAllychainValidatorTx
		exists bool
	)

	pendingStakers := vm.internalState.PendingStakerChainState()
	pendingValidator := pendingStakers.GetValidator(nodeID)

	currentStakers := vm.internalState.CurrentStakerChainState()
	currentValidator, err := currentStakers.GetValidator(nodeID)
	switch err {
	case nil:
		vdrTx, exists = currentValidator.AllychainValidators()[allychainID]
		if !exists {
			vdrTx = pendingValidator.AllychainValidators()[allychainID]
		}
	case database.ErrNotFound:
		vdrTx = pendingValidator.AllychainValidators()[allychainID]
	default:
		return 0, err
	}

	if vdrTx == nil {
		return 0, nil
	}
	if vdrTx.StartTime().After(endTime) {
		return 0, nil
	}
	if vdrTx.EndTime().Before(startTime) {
		return 0, nil
	}
	return vdrTx.Weight(), nil
}

func (vm *VM) maxPrimaryAllychainStakeAmount(
	nodeID ids.NodeID,
	startTime time.Time,
	endTime time.Time,
) (uint64, error) {
	currentStakers := vm.internalState.CurrentStakerChainState()
	pendingStakers := vm.internalState.PendingStakerChainState()

	pendingValidator := pendingStakers.GetValidator(nodeID)
	currentValidator, err := currentStakers.GetValidator(nodeID)

	switch err {
	case nil:
		vdrTx := currentValidator.AddValidatorTx()
		if vdrTx.StartTime().After(endTime) {
			return 0, nil
		}
		if vdrTx.EndTime().Before(startTime) {
			return 0, nil
		}

		currentWeight := vdrTx.Weight()
		currentWeight, err = math.Add64(currentWeight, currentValidator.NominatorWeight())
		if err != nil {
			return 0, err
		}
		return maxStakeAmount(
			currentValidator.Nominators(),
			pendingValidator.Nominators(),
			startTime,
			endTime,
			currentWeight,
		)
	case database.ErrNotFound:
		futureValidator, err := pendingStakers.GetValidatorTx(nodeID)
		if err == database.ErrNotFound {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		if futureValidator.StartTime().After(endTime) {
			return 0, nil
		}
		if futureValidator.EndTime().Before(startTime) {
			return 0, nil
		}

		return maxStakeAmount(
			nil,
			pendingValidator.Nominators(),
			startTime,
			endTime,
			futureValidator.Weight(),
		)
	default:
		return 0, err
	}
}
