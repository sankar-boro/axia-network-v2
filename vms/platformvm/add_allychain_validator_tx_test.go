// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"testing"
	"time"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/crypto"
	"github.com/sankar-boro/axia-network-v2/utils/hashing"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm/reward"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm/status"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
)

func TestAddAllychainValidatorTxSyntacticVerify(t *testing.T) {
	vm, _, _ := defaultVM()
	vm.ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		vm.ctx.Lock.Unlock()
	}()

	nodeID := ids.NodeID(keys[0].PublicKey().Address())

	// Case: tx is nil
	var unsignedTx *UnsignedAddAllychainValidatorTx
	if err := unsignedTx.SyntacticVerify(vm.ctx); err == nil {
		t.Fatal("should have errored because tx is nil")
	}

	// Case: Wrong network ID
	tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix()),
		nodeID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).NetworkID++
	// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).syntacticallyVerified = false
	if err := tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).SyntacticVerify(vm.ctx); err == nil {
		t.Fatal("should have errored because the wrong network ID was used")
	}

	// Case: Missing Allychain ID
	tx, err = vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix()),
		nodeID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).Validator.Allychain = ids.ID{}
	// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).syntacticallyVerified = false
	if err := tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).SyntacticVerify(vm.ctx); err == nil {
		t.Fatal("should have errored because Allychain ID is empty")
	}

	// Case: No weight
	tx, err = vm.newAddAllychainValidatorTx(
		1,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix()),
		nodeID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).Validator.Wght = 0
	// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).syntacticallyVerified = false
	if err := tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).SyntacticVerify(vm.ctx); err == nil {
		t.Fatal("should have errored because of no weight")
	}

	// Case: Allychain auth indices not unique
	tx, err = vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix())-1,
		nodeID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).AllychainAuth.(*secp256k1fx.Input).SigIndices[0] =
		tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).AllychainAuth.(*secp256k1fx.Input).SigIndices[1]
	// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).syntacticallyVerified = false
	if err = tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).SyntacticVerify(vm.ctx); err == nil {
		t.Fatal("should have errored because sig indices weren't unique")
	}

	// Case: Valid
	if tx, err = vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix()),
		nodeID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if err := tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).SyntacticVerify(vm.ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAddAllychainValidatorTxExecute(t *testing.T) {
	vm, _, _ := defaultVM()
	vm.ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		vm.ctx.Lock.Unlock()
	}()

	nodeID := keys[0].PublicKey().Address()

	// Case: Proposed validator currently validating primary network
	// but stops validating allychain after stops validating primary network
	// (note that keys[0] is a genesis validator)
	if tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix())+1,
		ids.NodeID(nodeID),
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if _, _, err := tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed because validator stops validating primary network earlier than allychain")
	}

	// Case: Proposed validator currently validating primary network
	// and proposed allychain validation period is subset of
	// primary network validation period
	// (note that keys[0] is a genesis validator)
	if tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(defaultValidateStartTime.Unix()+1),
		uint64(defaultValidateEndTime.Unix()),
		ids.NodeID(nodeID),
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if _, _, err = tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err != nil {
		t.Fatal(err)
	}

	// Add a validator to pending validator set of primary network
	key, err := testKeyfactory.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pendingDSValidatorID := ids.NodeID(key.PublicKey().Address())

	// starts validating primary network 10 seconds after genesis
	DSStartTime := defaultGenesisTime.Add(10 * time.Second)
	DSEndTime := DSStartTime.Add(5 * defaultMinStakingDuration)

	addDSTx, err := vm.newAddValidatorTx(
		vm.MinValidatorStake,                    // stake amount
		uint64(DSStartTime.Unix()),              // start time
		uint64(DSEndTime.Unix()),                // end time
		pendingDSValidatorID,                    // node ID
		nodeID,                                  // reward address
		reward.PercentDenominator,               // shares
		[]*crypto.PrivateKeySECP256K1R{keys[0]}, // key
		ids.ShortEmpty,                          // change addr

	)
	if err != nil {
		t.Fatal(err)
	}

	// Case: Proposed validator isn't in pending or current validator sets
	if tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(DSStartTime.Unix()), // start validating allychain before primary network
		uint64(DSEndTime.Unix()),
		pendingDSValidatorID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if _, _, err = tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed because validator not in the current or pending validator sets of the primary network")
	}

	vm.internalState.AddCurrentStaker(addDSTx, 0)
	vm.internalState.AddTx(addDSTx, status.Committed)
	if err := vm.internalState.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := vm.internalState.(*internalStateImpl).loadCurrentValidators(); err != nil {
		t.Fatal(err)
	}

	// Node with ID key.PublicKey().Address() now a pending validator for primary network

	// Case: Proposed validator is pending validator of primary network
	// but starts validating allychain before primary network
	if tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(DSStartTime.Unix())-1, // start validating allychain before primary network
		uint64(DSEndTime.Unix()),
		pendingDSValidatorID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if _, _, err := tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed because validator starts validating primary " +
			"network before starting to validate primary network")
	}

	// Case: Proposed validator is pending validator of primary network
	// but stops validating allychain after primary network
	if tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(DSStartTime.Unix()),
		uint64(DSEndTime.Unix())+1, // stop validating allychain after stopping validating primary network
		pendingDSValidatorID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if _, _, err = tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed because validator stops validating primary " +
			"network after stops validating primary network")
	}

	// Case: Proposed validator is pending validator of primary network
	// and period validating allychain is subset of time validating primary network
	if tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(DSStartTime.Unix()), // same start time as for primary network
		uint64(DSEndTime.Unix()),   // same end time as for primary network
		pendingDSValidatorID,
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if _, _, err := tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err != nil {
		t.Fatalf("should have passed verification")
	}

	// Case: Proposed validator start validating at/before current timestamp
	// First, advance the timestamp
	newTimestamp := defaultGenesisTime.Add(2 * time.Second)
	vm.internalState.SetTimestamp(newTimestamp)

	if tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,               // weight
		uint64(newTimestamp.Unix()), // start time
		uint64(newTimestamp.Add(defaultMinStakingDuration).Unix()), // end time
		ids.NodeID(nodeID), // node ID
		testAllychain1.ID(),   // allychain ID
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	); err != nil {
		t.Fatal(err)
	} else if _, _, err := tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed verification because starts validating at current timestamp")
	}

	// reset the timestamp
	vm.internalState.SetTimestamp(defaultGenesisTime)

	// Case: Proposed validator already validating the allychain
	// First, add validator as validator of allychain
	allychainTx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,                           // weight
		uint64(defaultValidateStartTime.Unix()), // start time
		uint64(defaultValidateEndTime.Unix()),   // end time
		ids.NodeID(nodeID),                      // node ID
		testAllychain1.ID(),                        // allychain ID
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}

	vm.internalState.AddCurrentStaker(allychainTx, 0)
	vm.internalState.AddTx(allychainTx, status.Committed)
	if err := vm.internalState.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := vm.internalState.(*internalStateImpl).loadCurrentValidators(); err != nil {
		t.Fatal(err)
	}

	// Node with ID nodeIDKey.PublicKey().Address() now validating allychain with ID testAllychain1.ID
	duplicateAllychainTx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,                           // weight
		uint64(defaultValidateStartTime.Unix()), // start time
		uint64(defaultValidateEndTime.Unix()),   // end time
		ids.NodeID(nodeID),                      // node ID
		testAllychain1.ID(),                        // allychain ID
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := duplicateAllychainTx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, duplicateAllychainTx); err == nil {
		t.Fatal("should have failed verification because validator already validating the specified allychain")
	}

	vm.internalState.DeleteCurrentStaker(allychainTx)
	if err := vm.internalState.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := vm.internalState.(*internalStateImpl).loadCurrentValidators(); err != nil {
		t.Fatal(err)
	}

	// Case: Too many signatures
	tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,                     // weight
		uint64(defaultGenesisTime.Unix()), // start time
		uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix())+1, // end time
		ids.NodeID(nodeID), // node ID
		testAllychain1.ID(),   // allychain ID
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1], testAllychain1ControlKeys[2]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed verification because tx has 3 signatures but only 2 needed")
	}

	// Case: Too few signatures
	tx, err = vm.newAddAllychainValidatorTx(
		defaultWeight,                     // weight
		uint64(defaultGenesisTime.Unix()), // start time
		uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix()), // end time
		ids.NodeID(nodeID), // node ID
		testAllychain1.ID(),   // allychain ID
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[2]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	// Remove a signature
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).AllychainAuth.(*secp256k1fx.Input).SigIndices =
		tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).AllychainAuth.(*secp256k1fx.Input).SigIndices[1:]
		// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
	tx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).syntacticallyVerified = false
	if _, _, err = tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed verification because not enough control sigs")
	}

	// Case: Control Signature from invalid key (keys[3] is not a control key)
	tx, err = vm.newAddAllychainValidatorTx(
		defaultWeight,                     // weight
		uint64(defaultGenesisTime.Unix()), // start time
		uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix()), // end time
		ids.NodeID(nodeID), // node ID
		testAllychain1.ID(),   // allychain ID
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], keys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	// Replace a valid signature with one from keys[3]
	sig, err := keys[3].SignHash(hashing.ComputeHash256(tx.UnsignedBytes()))
	if err != nil {
		t.Fatal(err)
	}
	copy(tx.Creds[0].(*secp256k1fx.Credential).Sigs[0][:], sig)
	if _, _, err = tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed verification because a control sig is invalid")
	}

	// Case: Proposed validator in pending validator set for allychain
	// First, add validator to pending validator set of allychain
	tx, err = vm.newAddAllychainValidatorTx(
		defaultWeight,                       // weight
		uint64(defaultGenesisTime.Unix())+1, // start time
		uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix())+1, // end time
		ids.NodeID(nodeID), // node ID
		testAllychain1.ID(),   // allychain ID
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
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

	if _, _, err = tx.UnsignedTx.(UnsignedProposalTx).Execute(vm, vm.internalState, tx); err == nil {
		t.Fatal("should have failed verification because validator already in pending validator set of the specified allychain")
	}
}

// Test that marshalling/unmarshalling works
func TestAddAllychainValidatorMarshal(t *testing.T) {
	vm, _, _ := defaultVM()
	vm.ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		vm.ctx.Lock.Unlock()
	}()

	var unmarshaledTx Tx

	// valid tx
	tx, err := vm.newAddAllychainValidatorTx(
		defaultWeight,
		uint64(defaultValidateStartTime.Unix()),
		uint64(defaultValidateEndTime.Unix()),
		ids.NodeID(keys[0].PublicKey().Address()),
		testAllychain1.ID(),
		[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
		ids.ShortEmpty, // change addr
	)
	if err != nil {
		t.Fatal(err)
	}
	txBytes, err := Codec.Marshal(CodecVersion, tx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Codec.Unmarshal(txBytes, &unmarshaledTx); err != nil {
		t.Fatal(err)
	}

	if err := unmarshaledTx.Sign(Codec, nil); err != nil {
		t.Fatal(err)
	}

	if err := unmarshaledTx.UnsignedTx.(*UnsignedAddAllychainValidatorTx).SyntacticVerify(vm.ctx); err != nil {
		t.Fatal(err)
	}
}
