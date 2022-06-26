// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow/consensus/snowman"
	"github.com/sankar-boro/axia/utils/crypto"
	"github.com/sankar-boro/axia/vms/platformvm/reward"
	"github.com/sankar-boro/axia/vms/platformvm/status"
)

func TestAddNominatorTxSyntacticVerify(t *testing.T) {
	vm, _, _ := defaultVM()
	vm.ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		vm.ctx.Lock.Unlock()
	}()

	rewardAddress := keys[0].PublicKey().Address()
	nodeID := ids.NodeID(rewardAddress)

	// Case : tx is nil
	var unsignedTx *UnsignedAddNominatorTx
	if err := unsignedTx.SyntacticVerify(vm.ctx); err == nil {
		t.Fatal("should have errored because tx is nil")
	}

	// Case: Wrong network ID
	tx, err := vm.newAddNominatorTx(
		vm.MinNominatorStake,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix()),
		nodeID,
		rewardAddress,
		[]*crypto.PrivateKeySECP256K1R{keys[0]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	tx.UnsignedTx.(*UnsignedAddNominatorTx).NetworkID++
	// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
	tx.UnsignedTx.(*UnsignedAddNominatorTx).syntacticallyVerified = false
	if err := tx.UnsignedTx.(*UnsignedAddNominatorTx).SyntacticVerify(vm.ctx); err == nil {
		t.Fatal("should have errored because the wrong network ID was used")
	}

	// Case: Valid
	if tx, err = vm.newAddNominatorTx(
		vm.MinNominatorStake,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix()),
		nodeID,
		rewardAddress,
		[]*crypto.PrivateKeySECP256K1R{keys[0]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if err := tx.UnsignedTx.(*UnsignedAddNominatorTx).SyntacticVerify(vm.ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAddNominatorTxExecute(t *testing.T) {
	rewardAddress := keys[0].PublicKey().Address()
	nodeID := ids.NodeID(rewardAddress)

	factory := crypto.FactorySECP256K1R{}
	keyIntf, err := factory.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	newValidatorKey := keyIntf.(*crypto.PrivateKeySECP256K1R)
	newValidatorID := ids.NodeID(newValidatorKey.PublicKey().Address())
	newValidatorStartTime := uint64(defaultValidateStartTime.Add(5 * time.Second).Unix())
	newValidatorEndTime := uint64(defaultValidateEndTime.Add(-5 * time.Second).Unix())

	// [addMinStakeValidator] adds a new validator to the primary network's
	// pending validator set with the minimum staking amount
	addMinStakeValidator := func(vm *VM) {
		tx, err := vm.newAddValidatorTx(
			vm.MinValidatorStake,                    // stake amount
			newValidatorStartTime,                   // start time
			newValidatorEndTime,                     // end time
			newValidatorID,                          // node ID
			rewardAddress,                           // Reward Address
			reward.PercentDenominator,               // subnet
			[]*crypto.PrivateKeySECP256K1R{keys[0]}, // key
			ids.ShortEmpty,                          // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		vm.internalState.AddCurrentStaker(tx, 0)
		vm.internalState.AddTx(tx, status.Committed)
		if err := vm.internalState.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := vm.internalState.(*internalStateImpl).loadCurrentValidators(); err != nil {
			t.Fatal(err)
		}
	}

	// [addMaxStakeValidator] adds a new validator to the primary network's
	// pending validator set with the maximum staking amount
	addMaxStakeValidator := func(vm *VM) {
		tx, err := vm.newAddValidatorTx(
			vm.MaxValidatorStake,                    // stake amount
			newValidatorStartTime,                   // start time
			newValidatorEndTime,                     // end time
			newValidatorID,                          // node ID
			rewardAddress,                           // Reward Address
			reward.PercentDenominator,               // subnet
			[]*crypto.PrivateKeySECP256K1R{keys[0]}, // key
			ids.ShortEmpty,                          // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		vm.internalState.AddCurrentStaker(tx, 0)
		vm.internalState.AddTx(tx, status.Committed)
		if err := vm.internalState.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := vm.internalState.(*internalStateImpl).loadCurrentValidators(); err != nil {
			t.Fatal(err)
		}
	}

	freshVM, _, _ := defaultVM()
	currentTimestamp := freshVM.internalState.GetTimestamp()

	type test struct {
		stakeAmount   uint64
		startTime     uint64
		endTime       uint64
		nodeID        ids.NodeID
		rewardAddress ids.ShortID
		feeKeys       []*crypto.PrivateKeySECP256K1R
		setup         func(vm *VM)
		AP3Time       time.Time
		shouldErr     bool
		description   string
	}

	tests := []test{
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     uint64(defaultValidateStartTime.Unix()),
			endTime:       uint64(defaultValidateEndTime.Unix()) + 1,
			nodeID:        nodeID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator stops validating primary network earlier than subnet",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     uint64(currentTimestamp.Add(maxFutureStartTime + time.Second).Unix()),
			endTime:       uint64(currentTimestamp.Add(maxFutureStartTime * 2).Unix()),
			nodeID:        nodeID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   fmt.Sprintf("validator should not be added more than (%s) in the future", maxFutureStartTime),
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     uint64(defaultValidateStartTime.Unix()),
			endTime:       uint64(defaultValidateEndTime.Unix()) + 1,
			nodeID:        nodeID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "end time is after the primary network end time",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     uint64(defaultValidateStartTime.Add(5 * time.Second).Unix()),
			endTime:       uint64(defaultValidateEndTime.Add(-5 * time.Second).Unix()),
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator not in the current or pending validator sets of the subnet",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     newValidatorStartTime - 1, // start validating subnet before primary network
			endTime:       newValidatorEndTime,
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         addMinStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator starts validating subnet before primary network",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     newValidatorStartTime,
			endTime:       newValidatorEndTime + 1, // stop validating subnet after stopping validating primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         addMinStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator stops validating primary network before subnet",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     newValidatorStartTime, // same start time as for primary network
			endTime:       newValidatorEndTime,   // same end time as for primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         addMinStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     false,
			description:   "valid",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake, // weight
			startTime:     uint64(currentTimestamp.Unix()),
			endTime:       uint64(defaultValidateEndTime.Unix()),
			nodeID:        nodeID,                                  // node ID
			rewardAddress: rewardAddress,                           // Reward Address
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]}, // tx fee payer
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "starts validating at current timestamp",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,               // weight
			startTime:     uint64(defaultValidateStartTime.Unix()), // start time
			endTime:       uint64(defaultValidateEndTime.Unix()),   // end time
			nodeID:        nodeID,                                  // node ID
			rewardAddress: rewardAddress,                           // Reward Address
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[1]}, // tx fee payer
			setup: func(vm *VM) { // Remove all UTXOs owned by keys[1]
				utxoIDs, err := vm.internalState.UTXOIDs(keys[1].PublicKey().Address().Bytes(), ids.Empty, math.MaxInt32)
				if err != nil {
					t.Fatal(err)
				}
				for _, utxoID := range utxoIDs {
					vm.internalState.DeleteUTXO(utxoID)
				}
				if err := vm.internalState.Commit(); err != nil {
					t.Fatal(err)
				}
			},
			AP3Time:     defaultGenesisTime,
			shouldErr:   true,
			description: "tx fee paying key has no funds",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     newValidatorStartTime, // same start time as for primary network
			endTime:       newValidatorEndTime,   // same end time as for primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         addMaxStakeValidator,
			AP3Time:       defaultValidateEndTime,
			shouldErr:     false,
			description:   "over delegation before AP3",
		},
		{
			stakeAmount:   freshVM.MinNominatorStake,
			startTime:     newValidatorStartTime, // same start time as for primary network
			endTime:       newValidatorEndTime,   // same end time as for primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{keys[0]},
			setup:         addMaxStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "over delegation after AP3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			vm, _, _ := defaultVM()
			vm.ApricotPhase3Time = tt.AP3Time

			vm.ctx.Lock.Lock()
			defer func() {
				if err := vm.Shutdown(); err != nil {
					t.Fatal(err)
				}
				vm.ctx.Lock.Unlock()
			}()

			tx, err := vm.newAddNominatorTx(
				tt.stakeAmount,
				tt.startTime,
				tt.endTime,
				tt.nodeID,
				tt.rewardAddress,
				tt.feeKeys,
				ids.ShortEmpty, // change addr
			)
			if err != nil {
				t.Fatalf("couldn't build tx: %s", err)
			}
			if tt.setup != nil {
				tt.setup(vm)
			}
			if _, _, err := tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err != nil && !tt.shouldErr {
				t.Fatalf("shouldn't have errored but got %s", err)
			} else if err == nil && tt.shouldErr {
				t.Fatalf("expected test to error but got none")
			}
		})
	}
}

func TestAddNominatorTxOverDelegatedRegression(t *testing.T) {
	assert := assert.New(t)

	vm, _, _ := defaultVM()
	vm.ctx.Lock.Lock()
	defer func() {
		err := vm.Shutdown()
		assert.NoError(err)

		vm.ctx.Lock.Unlock()
	}()

	validatorStartTime := defaultGenesisTime.Add(syncBound).Add(1 * time.Second)
	validatorEndTime := validatorStartTime.Add(360 * 24 * time.Hour)

	key, err := testKeyfactory.NewPrivateKey()
	assert.NoError(err)

	id := key.PublicKey().Address()

	// create valid tx
	addValidatorTx, err := vm.newAddValidatorTx(
		vm.MinValidatorStake,
		uint64(validatorStartTime.Unix()),
		uint64(validatorEndTime.Unix()),
		ids.NodeID(id),
		id,
		reward.PercentDenominator,
		[]*crypto.PrivateKeySECP256K1R{keys[0]},
		ids.ShortEmpty, // change addr
	)
	assert.NoError(err)

	// trigger block creation
	err = vm.blockBuilder.AddUnverifiedTx(addValidatorTx)
	assert.NoError(err)

	addValidatorBlock, err := vm.BuildBlock()
	assert.NoError(err)

	verifyAndAcceptProposalCommitment(assert, addValidatorBlock)

	vm.clock.Set(validatorStartTime)

	firstAdvanceTimeBlock, err := vm.BuildBlock()
	assert.NoError(err)

	verifyAndAcceptProposalCommitment(assert, firstAdvanceTimeBlock)

	firstNominatorStartTime := validatorStartTime.Add(syncBound).Add(1 * time.Second)
	firstNominatorEndTime := firstNominatorStartTime.Add(vm.MinStakeDuration)

	// create valid tx
	addFirstNominatorTx, err := vm.newAddNominatorTx(
		4*vm.MinValidatorStake, // maximum amount of stake this nominator can provide
		uint64(firstNominatorStartTime.Unix()),
		uint64(firstNominatorEndTime.Unix()),
		ids.NodeID(id),
		keys[0].PublicKey().Address(),
		[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1]},
		ids.ShortEmpty, // change addr
	)
	assert.NoError(err)

	// trigger block creation
	err = vm.blockBuilder.AddUnverifiedTx(addFirstNominatorTx)
	assert.NoError(err)

	addFirstNominatorBlock, err := vm.BuildBlock()
	assert.NoError(err)

	verifyAndAcceptProposalCommitment(assert, addFirstNominatorBlock)

	vm.clock.Set(firstNominatorStartTime)

	secondAdvanceTimeBlock, err := vm.BuildBlock()
	assert.NoError(err)

	verifyAndAcceptProposalCommitment(assert, secondAdvanceTimeBlock)

	secondNominatorStartTime := firstNominatorEndTime.Add(2 * time.Second)
	secondNominatorEndTime := secondNominatorStartTime.Add(vm.MinStakeDuration)

	vm.clock.Set(secondNominatorStartTime.Add(-10 * syncBound))

	// create valid tx
	addSecondNominatorTx, err := vm.newAddNominatorTx(
		vm.MinNominatorStake,
		uint64(secondNominatorStartTime.Unix()),
		uint64(secondNominatorEndTime.Unix()),
		ids.NodeID(id),
		keys[0].PublicKey().Address(),
		[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1], keys[3]},
		ids.ShortEmpty, // change addr
	)
	assert.NoError(err)

	// trigger block creation
	err = vm.blockBuilder.AddUnverifiedTx(addSecondNominatorTx)
	assert.NoError(err)

	addSecondNominatorBlock, err := vm.BuildBlock()
	assert.NoError(err)

	verifyAndAcceptProposalCommitment(assert, addSecondNominatorBlock)

	thirdNominatorStartTime := firstNominatorEndTime.Add(-time.Second)
	thirdNominatorEndTime := thirdNominatorStartTime.Add(vm.MinStakeDuration)

	// create valid tx
	addThirdNominatorTx, err := vm.newAddNominatorTx(
		vm.MinNominatorStake,
		uint64(thirdNominatorStartTime.Unix()),
		uint64(thirdNominatorEndTime.Unix()),
		ids.NodeID(id),
		keys[0].PublicKey().Address(),
		[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1], keys[4]},
		ids.ShortEmpty, // change addr
	)
	assert.NoError(err)

	// trigger block creation
	err = vm.blockBuilder.AddUnverifiedTx(addThirdNominatorTx)
	assert.Error(err, "should have marked the nominator as being over delegated")
}

func TestAddNominatorTxHeapCorruption(t *testing.T) {
	validatorStartTime := defaultGenesisTime.Add(syncBound).Add(1 * time.Second)
	validatorEndTime := validatorStartTime.Add(360 * 24 * time.Hour)
	validatorStake := defaultMaxValidatorStake / 5

	nominator1StartTime := validatorStartTime
	nominator1EndTime := nominator1StartTime.Add(3 * defaultMinStakingDuration)
	nominator1Stake := defaultMinValidatorStake

	nominator2StartTime := validatorStartTime.Add(1 * defaultMinStakingDuration)
	nominator2EndTime := nominator1StartTime.Add(6 * defaultMinStakingDuration)
	nominator2Stake := defaultMinValidatorStake

	nominator3StartTime := validatorStartTime.Add(2 * defaultMinStakingDuration)
	nominator3EndTime := nominator1StartTime.Add(4 * defaultMinStakingDuration)
	nominator3Stake := defaultMaxValidatorStake - validatorStake - 2*defaultMinValidatorStake

	nominator4StartTime := validatorStartTime.Add(5 * defaultMinStakingDuration)
	nominator4EndTime := nominator1StartTime.Add(7 * defaultMinStakingDuration)
	nominator4Stake := defaultMaxValidatorStake - validatorStake - defaultMinValidatorStake

	tests := []struct {
		name       string
		ap3Time    time.Time
		shouldFail bool
	}{
		{
			name:       "pre-upgrade is no longer restrictive",
			ap3Time:    validatorEndTime,
			shouldFail: false,
		},
		{
			name:       "post-upgrade calculate max stake correctly",
			ap3Time:    defaultGenesisTime,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			vm, _, _ := defaultVM()
			vm.ApricotPhase3Time = test.ap3Time

			vm.ctx.Lock.Lock()
			defer func() {
				err := vm.Shutdown()
				assert.NoError(err)

				vm.ctx.Lock.Unlock()
			}()

			key, err := testKeyfactory.NewPrivateKey()
			assert.NoError(err)

			id := key.PublicKey().Address()
			changeAddr := keys[0].PublicKey().Address()

			// create valid tx
			addValidatorTx, err := vm.newAddValidatorTx(
				validatorStake,
				uint64(validatorStartTime.Unix()),
				uint64(validatorEndTime.Unix()),
				ids.NodeID(id),
				id,
				reward.PercentDenominator,
				[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1]},
				changeAddr,
			)
			assert.NoError(err)

			// issue the add validator tx
			err = vm.blockBuilder.AddUnverifiedTx(addValidatorTx)
			assert.NoError(err)

			// trigger block creation for the validator tx
			addValidatorBlock, err := vm.BuildBlock()
			assert.NoError(err)

			verifyAndAcceptProposalCommitment(assert, addValidatorBlock)

			// create valid tx
			addFirstNominatorTx, err := vm.newAddNominatorTx(
				nominator1Stake,
				uint64(nominator1StartTime.Unix()),
				uint64(nominator1EndTime.Unix()),
				ids.NodeID(id),
				keys[0].PublicKey().Address(),
				[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1]},
				changeAddr,
			)
			assert.NoError(err)

			// issue the first add nominator tx
			err = vm.blockBuilder.AddUnverifiedTx(addFirstNominatorTx)
			assert.NoError(err)

			// trigger block creation for the first add nominator tx
			addFirstNominatorBlock, err := vm.BuildBlock()
			assert.NoError(err)

			verifyAndAcceptProposalCommitment(assert, addFirstNominatorBlock)

			// create valid tx
			addSecondNominatorTx, err := vm.newAddNominatorTx(
				nominator2Stake,
				uint64(nominator2StartTime.Unix()),
				uint64(nominator2EndTime.Unix()),
				ids.NodeID(id),
				keys[0].PublicKey().Address(),
				[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1]},
				changeAddr,
			)
			assert.NoError(err)

			// issue the second add nominator tx
			err = vm.blockBuilder.AddUnverifiedTx(addSecondNominatorTx)
			assert.NoError(err)

			// trigger block creation for the second add nominator tx
			addSecondNominatorBlock, err := vm.BuildBlock()
			assert.NoError(err)

			verifyAndAcceptProposalCommitment(assert, addSecondNominatorBlock)

			// create valid tx
			addThirdNominatorTx, err := vm.newAddNominatorTx(
				nominator3Stake,
				uint64(nominator3StartTime.Unix()),
				uint64(nominator3EndTime.Unix()),
				ids.NodeID(id),
				keys[0].PublicKey().Address(),
				[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1]},
				changeAddr,
			)
			assert.NoError(err)

			// issue the third add nominator tx
			err = vm.blockBuilder.AddUnverifiedTx(addThirdNominatorTx)
			assert.NoError(err)

			// trigger block creation for the third add nominator tx
			addThirdNominatorBlock, err := vm.BuildBlock()
			assert.NoError(err)

			verifyAndAcceptProposalCommitment(assert, addThirdNominatorBlock)

			// create valid tx
			addFourthNominatorTx, err := vm.newAddNominatorTx(
				nominator4Stake,
				uint64(nominator4StartTime.Unix()),
				uint64(nominator4EndTime.Unix()),
				ids.NodeID(id),
				keys[0].PublicKey().Address(),
				[]*crypto.PrivateKeySECP256K1R{keys[0], keys[1]},
				changeAddr,
			)
			assert.NoError(err)

			// issue the fourth add nominator tx
			err = vm.blockBuilder.AddUnverifiedTx(addFourthNominatorTx)
			assert.NoError(err)

			// trigger block creation for the fourth add nominator tx
			addFourthNominatorBlock, err := vm.BuildBlock()

			if test.shouldFail {
				assert.Error(err, "should have failed to allow new nominator")
				return
			}

			assert.NoError(err)

			verifyAndAcceptProposalCommitment(assert, addFourthNominatorBlock)
		})
	}
}

func verifyAndAcceptProposalCommitment(assert *assert.Assertions, blk snowman.Block) {
	// Verify the proposed block
	err := blk.Verify()
	assert.NoError(err)

	// Assert preferences are correct
	proposalBlk := blk.(*ProposalBlock)
	options, err := proposalBlk.Options()
	assert.NoError(err)

	// verify the preferences
	commit, ok := options[0].(*CommitBlock)
	assert.True(ok, "expected commit block to be preferred")

	abort, ok := options[1].(*AbortBlock)
	assert.True(ok, "expected abort block to be issued")

	err = commit.Verify()
	assert.NoError(err)

	err = abort.Verify()
	assert.NoError(err)

	// Accept the proposal block and the commit block
	err = proposalBlk.Accept()
	assert.NoError(err)

	err = commit.Accept()
	assert.NoError(err)

	err = abort.Reject()
	assert.NoError(err)
}
