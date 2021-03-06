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
	"github.com/sankar-boro/axia-network-v2/utils/crypto"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/components/verify"

	coreChainValidator "github.com/sankar-boro/axia-network-v2/vms/platformvm/validator"
)

var (
	errDSValidatorSubset = errors.New("all allychains' staking period must be a subset of the primary network")

	_ UnsignedProposalTx = &UnsignedAddAllychainValidatorTx{}
	_ TimedTx            = &UnsignedAddAllychainValidatorTx{}
)

// UnsignedAddAllychainValidatorTx is an unsigned addAllychainValidatorTx
type UnsignedAddAllychainValidatorTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// The validator
	Validator coreChainValidator.AllychainValidator `serialize:"true" json:"validator"`
	// Auth that will be allowing this validator into the network
	AllychainAuth verify.Verifiable `serialize:"true" json:"allychainAuthorization"`
}

// StartTime of this validator
func (tx *UnsignedAddAllychainValidatorTx) StartTime() time.Time {
	return tx.Validator.StartTime()
}

// EndTime of this validator
func (tx *UnsignedAddAllychainValidatorTx) EndTime() time.Time {
	return tx.Validator.EndTime()
}

// Weight of this validator
func (tx *UnsignedAddAllychainValidatorTx) Weight() uint64 {
	return tx.Validator.Weight()
}

// SyntacticVerify returns nil iff [tx] is valid
func (tx *UnsignedAddAllychainValidatorTx) SyntacticVerify(ctx *snow.Context) error {
	switch {
	case tx == nil:
		return errNilTx
	case tx.syntacticallyVerified: // already passed syntactic verification
		return nil
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := verify.All(&tx.Validator, tx.AllychainAuth); err != nil {
		return err
	}

	// cache that this is valid
	tx.syntacticallyVerified = true
	return nil
}

// Attempts to verify this transaction with the provided state.
func (tx *UnsignedAddAllychainValidatorTx) SemanticVerify(vm *VM, parentState MutableState, stx *Tx) error {
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
func (tx *UnsignedAddAllychainValidatorTx) Execute(
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
	case len(stx.Creds) == 0:
		return nil, nil, errWrongNumberOfCredentials
	}

	currentStakers := parentState.CurrentStakerChainState()
	pendingStakers := parentState.PendingStakerChainState()

	if vm.bootstrapped.GetValue() {
		currentTimestamp := parentState.GetTimestamp()
		// Ensure the proposed validator starts after the current timestamp
		validatorStartTime := tx.StartTime()
		if !currentTimestamp.Before(validatorStartTime) {
			return nil, nil, fmt.Errorf(
				"validator's start time (%s) is at or after current chain timestamp (%s)",
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

		var vdrTx *UnsignedAddValidatorTx
		if err == nil {
			// This validator is attempting to validate with a currently
			// validing node.
			vdrTx = currentValidator.AddValidatorTx()

			// Ensure that this transaction isn't a duplicate add validator tx.
			allychains := currentValidator.AllychainValidators()
			if _, validates := allychains[tx.Validator.Allychain]; validates {
				return nil, nil, fmt.Errorf(
					"already validating allychain %s",
					tx.Validator.Allychain,
				)
			}
		} else {
			// This validator is attempting to validate with a node that hasn't
			// started validating yet.
			vdrTx, err = pendingStakers.GetValidatorTx(tx.Validator.NodeID)
			if err != nil {
				if err == database.ErrNotFound {
					return nil, nil, errDSValidatorSubset
				}
				return nil, nil, fmt.Errorf(
					"failed to find whether %s is a validator: %w",
					tx.Validator.NodeID,
					err,
				)
			}
		}

		// Ensure that the period this validator validates the specified allychain
		// is a subset of the time they validate the primary network.
		if !tx.Validator.BoundedBy(vdrTx.StartTime(), vdrTx.EndTime()) {
			return nil, nil, errDSValidatorSubset
		}

		// Ensure that this transaction isn't a duplicate add validator tx.
		pendingValidator := pendingStakers.GetValidator(tx.Validator.NodeID)
		allychains := pendingValidator.AllychainValidators()
		if _, validates := allychains[tx.Validator.Allychain]; validates {
			return nil, nil, fmt.Errorf(
				"already validating allychain %s",
				tx.Validator.Allychain,
			)
		}

		baseTxCredsLen := len(stx.Creds) - 1
		baseTxCreds := stx.Creds[:baseTxCredsLen]
		allychainCred := stx.Creds[baseTxCredsLen]

		allychainIntf, _, err := parentState.GetTx(tx.Validator.Allychain)
		if err != nil {
			if err == database.ErrNotFound {
				return nil, nil, errDSValidatorSubset
			}
			return nil, nil, fmt.Errorf(
				"couldn't find allychain %s with %w",
				tx.Validator.Allychain,
				err,
			)
		}

		allychain, ok := allychainIntf.UnsignedTx.(*UnsignedCreateAllychainTx)
		if !ok {
			return nil, nil, fmt.Errorf(
				"%s is not a allychain",
				tx.Validator.Allychain,
			)
		}

		if err := vm.fx.VerifyPermission(tx, tx.AllychainAuth, allychainCred, allychain.Owner); err != nil {
			return nil, nil, err
		}

		// Verify the flowcheck
		if err := vm.semanticVerifySpend(parentState, tx, tx.Ins, tx.Outs, baseTxCreds, vm.TxFee, vm.ctx.AXCAssetID); err != nil {
			return nil, nil, err
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
	produceOutputs(onAbortState, txID, vm.ctx.AXCAssetID, tx.Outs)

	return onCommitState, onAbortState, nil
}

// InitiallyPrefersCommit returns true if the proposed validators start time is
// after the current wall clock time,
func (tx *UnsignedAddAllychainValidatorTx) InitiallyPrefersCommit(vm *VM) bool {
	return tx.StartTime().After(vm.clock.Time())
}

// Create a new transaction
func (vm *VM) newAddAllychainValidatorTx(
	weight, // Sampling weight of the new validator
	startTime, // Unix time they start delegating
	endTime uint64, // Unix time they top delegating
	nodeID ids.NodeID, // ID of the node validating
	allychainID ids.ID, // ID of the allychain the validator will validate
	keys []*crypto.PrivateKeySECP256K1R, // Keys to use for adding the validator
	changeAddr ids.ShortID, // Address to send change to, if there is any
) (*Tx, error) {
	ins, outs, _, signers, err := vm.stake(keys, 0, vm.TxFee, changeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate tx inputs/outputs: %w", err)
	}

	allychainAuth, allychainSigners, err := vm.authorize(vm.internalState, allychainID, keys)
	if err != nil {
		return nil, fmt.Errorf("couldn't authorize tx's allychain restrictions: %w", err)
	}
	signers = append(signers, allychainSigners)

	// Create the tx
	utx := &UnsignedAddAllychainValidatorTx{
		BaseTx: BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    vm.ctx.NetworkID,
			BlockchainID: vm.ctx.ChainID,
			Ins:          ins,
			Outs:         outs,
		}},
		Validator: coreChainValidator.AllychainValidator{
			Validator: coreChainValidator.Validator{
				NodeID: nodeID,
				Start:  startTime,
				End:    endTime,
				Wght:   weight,
			},
			Allychain: allychainID,
		},
		AllychainAuth: allychainAuth,
	}
	tx := &Tx{UnsignedTx: utx}
	if err := tx.Sign(Codec, signers); err != nil {
		return nil, err
	}
	return tx, utx.SyntacticVerify(vm.ctx)
}
