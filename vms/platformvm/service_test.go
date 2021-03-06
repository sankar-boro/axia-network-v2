// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	stdjson "encoding/json"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/api"
	"github.com/sankar-boro/axia-network-v2/api/keystore"
	"github.com/sankar-boro/axia-network-v2/chains/atomic"
	"github.com/sankar-boro/axia-network-v2/database/manager"
	"github.com/sankar-boro/axia-network-v2/database/prefixdb"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/utils/crypto"
	"github.com/sankar-boro/axia-network-v2/utils/formatting"
	"github.com/sankar-boro/axia-network-v2/utils/json"
	"github.com/sankar-boro/axia-network-v2/utils/logging"
	"github.com/sankar-boro/axia-network-v2/version"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm/status"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"

	vmkeystore "github.com/sankar-boro/axia-network-v2/vms/components/keystore"
)

var (
	// Test user username
	testUsername = "ScoobyUser"

	// Test user password, must meet minimum complexity/length requirements
	testPassword = "ShaggyPassword1Zoinks!"

	// Bytes docoded from CB58 "ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN"
	testPrivateKey = []byte{
		0x56, 0x28, 0x9e, 0x99, 0xc9, 0x4b, 0x69, 0x12,
		0xbf, 0xc1, 0x2a, 0xdc, 0x09, 0x3c, 0x9b, 0x51,
		0x12, 0x4f, 0x0d, 0xc5, 0x4a, 0xc7, 0xa7, 0x66,
		0xb2, 0xbc, 0x5c, 0xcf, 0x55, 0x8d, 0x80, 0x27,
	}

	// 3cb7d3842e8cee6a0ebd09f1fe884f6861e1b29c
	// Platform address resulting from the above private key
	testAddress = "Core-testing18jma8ppw3nhx5r4ap8clazz0dps7rv5umpc36y"

	encodings = []formatting.Encoding{
		formatting.JSON, formatting.Hex, formatting.CB58,
	}
)

func defaultService(t *testing.T) *Service {
	vm, _, _ := defaultVM()
	vm.ctx.Lock.Lock()
	defer vm.ctx.Lock.Unlock()
	ks := keystore.New(logging.NoLog{}, manager.NewMemDB(version.DefaultVersion1_0_0))
	if err := ks.CreateUser(testUsername, testPassword); err != nil {
		t.Fatal(err)
	}
	vm.ctx.Keystore = ks.NewBlockchainKeyStore(vm.ctx.ChainID)
	return &Service{vm: vm}
}

// Give user [testUsername] control of [testPrivateKey] and keys[0] (which is funded)
func defaultAddress(t *testing.T, service *Service) {
	service.vm.ctx.Lock.Lock()
	defer service.vm.ctx.Lock.Unlock()
	user, err := vmkeystore.NewUserFromKeystore(service.vm.ctx.Keystore, testUsername, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := testKeyfactory.ToPrivateKey(testPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	privKey := pk.(*crypto.PrivateKeySECP256K1R)
	if err := user.PutKeys(privKey, keys[0]); err != nil {
		t.Fatal(err)
	}
}

func TestAddValidator(t *testing.T) {
	expectedJSONString := `{"username":"","password":"","from":null,"changeAddr":"","txID":"11111111111111111111111111111111LpoYY","startTime":"0","endTime":"0","nodeID":"NodeID-111111111111111111116DBWJs","rewardAddress":"","delegationFeeRate":"0.0000"}`
	args := AddValidatorArgs{}
	bytes, err := stdjson.Marshal(&args)
	if err != nil {
		t.Fatal(err)
	}
	jsonString := string(bytes)
	if jsonString != expectedJSONString {
		t.Fatalf("Expected: %s\nResult: %s", expectedJSONString, jsonString)
	}
}

func TestCreateBlockchainArgsParsing(t *testing.T) {
	jsonString := `{"vmID":"lol","fxIDs":["secp256k1"], "name":"awesome", "username":"bob loblaw", "password":"yeet", "genesisData":"SkB92YpWm4Q2iPnLGCuDPZPgUQMxajqQQuz91oi3xD984f8r"}`
	args := CreateBlockchainArgs{}
	err := stdjson.Unmarshal([]byte(jsonString), &args)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = stdjson.Marshal(args.GenesisData); err != nil {
		t.Fatal(err)
	}
}

func TestExportKey(t *testing.T) {
	jsonString := `{"username":"ScoobyUser","password":"ShaggyPassword1Zoinks!","address":"` + testAddress + `"}`
	args := ExportKeyArgs{}
	err := stdjson.Unmarshal([]byte(jsonString), &args)
	if err != nil {
		t.Fatal(err)
	}

	service := defaultService(t)
	defaultAddress(t, service)
	service.vm.ctx.Lock.Lock()
	defer func() {
		if err := service.vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		service.vm.ctx.Lock.Unlock()
	}()

	reply := ExportKeyReply{}
	if err := service.ExportKey(nil, &args, &reply); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(testPrivateKey, reply.PrivateKey.Bytes()) {
		t.Fatalf("Expected %v, got %v", testPrivateKey, reply.PrivateKey.Bytes())
	}
}

func TestImportKey(t *testing.T) {
	jsonString := `{"username":"ScoobyUser","password":"ShaggyPassword1Zoinks!","privateKey":"PrivateKey-ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN"}`
	args := ImportKeyArgs{}
	err := stdjson.Unmarshal([]byte(jsonString), &args)
	if err != nil {
		t.Fatal(err)
	}

	service := defaultService(t)
	service.vm.ctx.Lock.Lock()
	defer func() {
		if err := service.vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		service.vm.ctx.Lock.Unlock()
	}()

	reply := api.JSONAddress{}
	if err := service.ImportKey(nil, &args, &reply); err != nil {
		t.Fatal(err)
	}
	if testAddress != reply.Address {
		t.Fatalf("Expected %q, got %q", testAddress, reply.Address)
	}
}

// Test issuing a tx and accepted
func TestGetTxStatus(t *testing.T) {
	service := defaultService(t)
	defaultAddress(t, service)
	service.vm.ctx.Lock.Lock()
	defer func() {
		if err := service.vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		service.vm.ctx.Lock.Unlock()
	}()

	factory := crypto.FactorySECP256K1R{}
	recipientKeyIntf, err := factory.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	recipientKey := recipientKeyIntf.(*crypto.PrivateKeySECP256K1R)

	m := &atomic.Memory{}
	err = m.Initialize(logging.NoLog{}, prefixdb.New([]byte{}, service.vm.dbManager.Current().Database))
	if err != nil {
		t.Fatal(err)
	}

	sm := m.NewSharedMemory(service.vm.ctx.ChainID)
	peerSharedMemory := m.NewSharedMemory(swapChainID)

	// #nosec G404
	utxo := &axc.UTXO{
		UTXOID: axc.UTXOID{
			TxID:        ids.GenerateTestID(),
			OutputIndex: rand.Uint32(),
		},
		Asset: axc.Asset{ID: axcAssetID},
		Out: &secp256k1fx.TransferOutput{
			Amt: 1234567,
			OutputOwners: secp256k1fx.OutputOwners{
				Locktime:  0,
				Addrs:     []ids.ShortID{recipientKey.PublicKey().Address()},
				Threshold: 1,
			},
		},
	}
	utxoBytes, err := Codec.Marshal(CodecVersion, utxo)
	if err != nil {
		t.Fatal(err)
	}
	inputID := utxo.InputID()
	if err := peerSharedMemory.Apply(map[ids.ID]*atomic.Requests{service.vm.ctx.ChainID: {PutRequests: []*atomic.Element{{
		Key:   inputID[:],
		Value: utxoBytes,
		Traits: [][]byte{
			recipientKey.PublicKey().Address().Bytes(),
		},
	}}}}); err != nil {
		t.Fatal(err)
	}

	oldAtomicUTXOManager := service.vm.AtomicUTXOManager
	newAtomicUTXOManager := axc.NewAtomicUTXOManager(sm, Codec)

	service.vm.AtomicUTXOManager = newAtomicUTXOManager
	tx, err := service.vm.newImportTx(swapChainID, ids.ShortEmpty, []*crypto.PrivateKeySECP256K1R{recipientKey}, ids.ShortEmpty)
	if err != nil {
		t.Fatal(err)
	}
	service.vm.AtomicUTXOManager = oldAtomicUTXOManager

	var (
		arg  = &GetTxStatusArgs{TxID: tx.ID()}
		resp GetTxStatusResponse
	)
	err = service.GetTxStatus(nil, arg, &resp)
	switch {
	case err != nil:
		t.Fatal(err)
	case resp.Status != status.Unknown:
		t.Fatalf("status should be unknown but is %s", resp.Status)
	case resp.Reason != "":
		t.Fatalf("reason should be empty but is %s", resp.Reason)
	}

	// put the chain in existing chain list
	if err := service.vm.blockBuilder.AddUnverifiedTx(tx); err == nil {
		t.Fatal("should have errored because of missing funds")
	}

	service.vm.AtomicUTXOManager = newAtomicUTXOManager
	service.vm.ctx.SharedMemory = sm

	if err := service.vm.blockBuilder.AddUnverifiedTx(tx); err != nil {
		t.Fatal(err)
	} else if block, err := service.vm.BuildBlock(); err != nil {
		t.Fatal(err)
	} else if blk, ok := block.(*StandardBlock); !ok {
		t.Fatalf("should be *StandardBlock but is %T", block)
	} else if err := blk.Verify(); err != nil {
		t.Fatal(err)
	} else if err := blk.Accept(); err != nil {
		t.Fatal(err)
	}

	resp = GetTxStatusResponse{} // reset
	err = service.GetTxStatus(nil, arg, &resp)
	switch {
	case err != nil:
		t.Fatal(err)
	case resp.Status != status.Committed:
		t.Fatalf("status should be Committed but is %s", resp.Status)
	case resp.Reason != "":
		t.Fatalf("reason should be empty but is %s", resp.Reason)
	}
}

// Test issuing and then retrieving a transaction
func TestGetTx(t *testing.T) {
	type test struct {
		description string
		createTx    func(service *Service) (*Tx, error)
	}

	tests := []test{
		{
			"standard block",
			func(service *Service) (*Tx, error) {
				return service.vm.newCreateChainTx( // Test GetTx works for standard blocks
					testAllychain1.ID(),
					nil,
					constants.AVMID,
					nil,
					"chain name",
					[]*crypto.PrivateKeySECP256K1R{testAllychain1ControlKeys[0], testAllychain1ControlKeys[1]},
					keys[0].PublicKey().Address(), // change addr
				)
			},
		},
		{
			"proposal block",
			func(service *Service) (*Tx, error) {
				return service.vm.newAddValidatorTx( // Test GetTx works for proposal blocks
					service.vm.MinValidatorStake,
					uint64(service.vm.clock.Time().Add(syncBound).Unix()),
					uint64(service.vm.clock.Time().Add(syncBound).Add(defaultMinStakingDuration).Unix()),
					ids.GenerateTestNodeID(),
					ids.GenerateTestShortID(),
					0,
					[]*crypto.PrivateKeySECP256K1R{keys[0]},
					keys[0].PublicKey().Address(), // change addr
				)
			},
		},
		{
			"atomic block",
			func(service *Service) (*Tx, error) {
				return service.vm.newExportTx( // Test GetTx works for proposal blocks
					100,
					service.vm.ctx.SwapChainID,
					ids.GenerateTestShortID(),
					[]*crypto.PrivateKeySECP256K1R{keys[0]},
					keys[0].PublicKey().Address(), // change addr
				)
			},
		},
	}

	for _, test := range tests {
		for _, encoding := range encodings {
			service := defaultService(t)
			defaultAddress(t, service)
			service.vm.ctx.Lock.Lock()

			tx, err := test.createTx(service)
			if err != nil {
				t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
			}
			arg := &api.GetTxArgs{
				TxID:     tx.ID(),
				Encoding: encoding,
			}
			var response api.GetTxReply
			if err := service.GetTx(nil, arg, &response); err == nil {
				t.Fatalf("failed test '%s - %s': haven't issued tx yet so shouldn't be able to get it", test.description, encoding.String())
			} else if err := service.vm.blockBuilder.AddUnverifiedTx(tx); err != nil {
				t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
			} else if block, err := service.vm.BuildBlock(); err != nil {
				t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
			} else if err := block.Verify(); err != nil {
				t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
			} else if err := block.Accept(); err != nil {
				t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
			} else if blk, ok := block.(*ProposalBlock); ok { // For proposal blocks, commit them
				if options, err := blk.Options(); err != nil {
					t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
				} else if commit, ok := options[0].(*CommitBlock); !ok {
					t.Fatalf("failed test '%s - %s': should prefer to commit", test.description, encoding.String())
				} else if err := commit.Verify(); err != nil {
					t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
				} else if err := commit.Accept(); err != nil {
					t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
				}
			} else if err := service.GetTx(nil, arg, &response); err != nil {
				t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
			} else {
				switch encoding {
				case formatting.Hex, formatting.CB58:
					// we're always guaranteed a string for hex/cb58 encodings.
					responseTxBytes, err := formatting.Decode(response.Encoding, response.Tx.(string))
					if err != nil {
						t.Fatalf("failed test '%s - %s': %s", test.description, encoding.String(), err)
					}
					if !bytes.Equal(responseTxBytes, tx.Bytes()) {
						t.Fatalf("failed test '%s - %s': byte representation of tx in response is incorrect", test.description, encoding.String())
					}
				case formatting.JSON:
					if response.Tx != tx {
						t.Fatalf("failed test '%s - %s': byte representation of tx in response is incorrect", test.description, encoding.String())
					}
				}
			}

			if err := service.vm.Shutdown(); err != nil {
				t.Fatal(err)
			}
			service.vm.ctx.Lock.Unlock()
		}
	}
}

// Test method GetBalance
func TestGetBalance(t *testing.T) {
	service := defaultService(t)
	defaultAddress(t, service)
	service.vm.ctx.Lock.Lock()
	defer func() {
		if err := service.vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		service.vm.ctx.Lock.Unlock()
	}()

	// Ensure GetStake is correct for each of the genesis validators
	genesis, _ := defaultGenesis()
	for _, utxo := range genesis.UTXOs {
		request := GetBalanceRequest{
			Addresses: []string{
				fmt.Sprintf("Core-%s", utxo.Address),
			},
		}
		reply := GetBalanceResponse{}
		if err := service.GetBalance(nil, &request, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Balance != json.Uint64(defaultBalance) {
			t.Fatalf("Wrong balance. Expected %d ; Returned %d", defaultBalance, reply.Balance)
		}
		if reply.Unlocked != json.Uint64(defaultBalance) {
			t.Fatalf("Wrong unlocked balance. Expected %d ; Returned %d", defaultBalance, reply.Unlocked)
		}
		if reply.LockedStakeable != 0 {
			t.Fatalf("Wrong locked stakeable balance. Expected %d ; Returned %d", reply.LockedStakeable, 0)
		}
		if reply.LockedNotStakeable != 0 {
			t.Fatalf("Wrong locked not stakeable balance. Expected %d ; Returned %d", reply.LockedNotStakeable, 0)
		}
	}
}

// Test method GetStake
func TestGetStake(t *testing.T) {
	assert := assert.New(t)
	service := defaultService(t)
	defaultAddress(t, service)
	service.vm.ctx.Lock.Lock()
	defer func() {
		err := service.vm.Shutdown()
		assert.NoError(err)
		service.vm.ctx.Lock.Unlock()
	}()

	// Ensure GetStake is correct for each of the genesis validators
	genesis, _ := defaultGenesis()
	addrsStrs := []string{}
	for i, validator := range genesis.Validators {
		addr := fmt.Sprintf("Core-%s", validator.RewardOwner.Addresses[0])
		addrsStrs = append(addrsStrs, addr)
		args := GetStakeArgs{
			api.JSONAddresses{
				Addresses: []string{addr},
			},
			formatting.Hex,
		}
		response := GetStakeReply{}
		err := service.GetStake(nil, &args, &response)
		assert.NoError(err)
		assert.EqualValues(uint64(defaultWeight), uint64(response.Staked))
		assert.Len(response.Outputs, 1)
		// Unmarshal into an output
		outputBytes, err := formatting.Decode(args.Encoding, response.Outputs[0])
		assert.NoError(err)
		var output axc.TransferableOutput
		_, err = Codec.Unmarshal(outputBytes, &output)
		assert.NoError(err)
		out, ok := output.Out.(*secp256k1fx.TransferOutput)
		assert.True(ok)
		assert.EqualValues(out.Amount(), defaultWeight)
		assert.EqualValues(out.Threshold, 1)
		assert.Len(out.Addrs, 1)
		assert.Equal(keys[i].PublicKey().Address(), out.Addrs[0])
		assert.EqualValues(out.Locktime, 0)
	}

	// Make sure this works for multiple addresses
	args := GetStakeArgs{
		api.JSONAddresses{
			Addresses: addrsStrs,
		},
		formatting.Hex,
	}
	response := GetStakeReply{}
	err := service.GetStake(nil, &args, &response)
	assert.NoError(err)
	assert.EqualValues(len(genesis.Validators)*defaultWeight, response.Staked)
	assert.Len(response.Outputs, len(genesis.Validators))
	for _, outputStr := range response.Outputs {
		outputBytes, err := formatting.Decode(args.Encoding, outputStr)
		assert.NoError(err)
		var output axc.TransferableOutput
		_, err = Codec.Unmarshal(outputBytes, &output)
		assert.NoError(err)
		out, ok := output.Out.(*secp256k1fx.TransferOutput)
		assert.True(ok)
		assert.EqualValues(defaultWeight, out.Amount())
		assert.EqualValues(out.Threshold, 1)
		assert.EqualValues(out.Locktime, 0)
		assert.Len(out.Addrs, 1)
	}

	oldStake := uint64(defaultWeight)

	// Add a nominator
	stakeAmt := service.vm.MinNominatorStake + 12345
	nominatorNodeID := ids.NodeID(keys[0].PublicKey().Address())
	nominatorEndTime := uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix())
	tx, err := service.vm.newAddNominatorTx(
		stakeAmt,
		uint64(defaultGenesisTime.Unix()),
		nominatorEndTime,
		nominatorNodeID,
		ids.GenerateTestShortID(),
		[]*crypto.PrivateKeySECP256K1R{keys[0]},
		keys[0].PublicKey().Address(), // change addr
	)
	assert.NoError(err)

	service.vm.internalState.AddCurrentStaker(tx, 0)
	service.vm.internalState.AddTx(tx, status.Committed)
	err = service.vm.internalState.Commit()
	assert.NoError(err)
	err = service.vm.internalState.(*internalStateImpl).loadCurrentValidators()
	assert.NoError(err)

	// Make sure the nominator addr has the right stake (old stake + stakeAmt)
	addr, _ := service.vm.FormatLocalAddress(keys[0].PublicKey().Address())
	args.Addresses = []string{addr}
	err = service.GetStake(nil, &args, &response)
	assert.NoError(err)
	assert.EqualValues(oldStake+stakeAmt, uint64(response.Staked))
	assert.Len(response.Outputs, 2)
	// Unmarshal into transferableoutputs
	outputs := make([]axc.TransferableOutput, 2)
	for i := range outputs {
		outputBytes, err := formatting.Decode(args.Encoding, response.Outputs[i])
		assert.NoError(err)
		_, err = Codec.Unmarshal(outputBytes, &outputs[i])
		assert.NoError(err)
	}
	// Make sure the stake amount is as expected
	assert.EqualValues(stakeAmt+oldStake, outputs[0].Out.Amount()+outputs[1].Out.Amount())

	oldStake = uint64(response.Staked)

	// Make sure this works for pending stakers
	// Add a pending staker
	stakeAmt = service.vm.MinValidatorStake + 54321
	pendingStakerNodeID := ids.GenerateTestNodeID()
	pendingStakerEndTime := uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix())
	tx, err = service.vm.newAddValidatorTx(
		stakeAmt,
		uint64(defaultGenesisTime.Unix()),
		pendingStakerEndTime,
		pendingStakerNodeID,
		ids.GenerateTestShortID(),
		0,
		[]*crypto.PrivateKeySECP256K1R{keys[0]},
		keys[0].PublicKey().Address(), // change addr
	)
	assert.NoError(err)

	service.vm.internalState.AddPendingStaker(tx)
	service.vm.internalState.AddTx(tx, status.Committed)
	err = service.vm.internalState.Commit()
	assert.NoError(err)
	err = service.vm.internalState.(*internalStateImpl).loadPendingValidators()
	assert.NoError(err)

	// Make sure the nominator has the right stake (old stake + stakeAmt)
	err = service.GetStake(nil, &args, &response)
	assert.NoError(err)
	assert.EqualValues(oldStake+stakeAmt, response.Staked)
	assert.Len(response.Outputs, 3)
	outputs = make([]axc.TransferableOutput, 3)
	// Unmarshal
	for i := range outputs {
		outputBytes, err := formatting.Decode(args.Encoding, response.Outputs[i])
		assert.NoError(err)
		_, err = Codec.Unmarshal(outputBytes, &outputs[i])
		assert.NoError(err)
	}
	// Make sure the stake amount is as expected
	assert.EqualValues(stakeAmt+oldStake, outputs[0].Out.Amount()+outputs[1].Out.Amount()+outputs[2].Out.Amount())
}

// Test method GetCurrentValidators
func TestGetCurrentValidators(t *testing.T) {
	service := defaultService(t)
	defaultAddress(t, service)
	service.vm.ctx.Lock.Lock()
	defer func() {
		if err := service.vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		service.vm.ctx.Lock.Unlock()
	}()

	genesis, _ := defaultGenesis()

	// Call getValidators
	args := GetCurrentValidatorsArgs{AllychainID: constants.PrimaryNetworkID}
	response := GetCurrentValidatorsReply{}

	err := service.GetCurrentValidators(nil, &args, &response)
	switch {
	case err != nil:
		t.Fatal(err)
	case len(response.Validators) != len(genesis.Validators):
		t.Fatalf("should be %d validators but are %d", len(genesis.Validators), len(response.Validators))
	}

	for _, vdr := range genesis.Validators {
		found := false
		for i := 0; i < len(response.Validators) && !found; i++ {
			gotVdr, ok := response.Validators[i].(APIPrimaryValidator)
			switch {
			case !ok:
				t.Fatal("expected APIPrimaryValidator")
			case gotVdr.NodeID != vdr.NodeID:
			case gotVdr.EndTime != vdr.EndTime:
				t.Fatalf("expected end time of %s to be %v but got %v",
					vdr.NodeID,
					vdr.EndTime,
					gotVdr.EndTime,
				)
			case gotVdr.StartTime != vdr.StartTime:
				t.Fatalf("expected start time of %s to be %v but got %v",
					vdr.NodeID,
					vdr.StartTime,
					gotVdr.StartTime,
				)
			case gotVdr.Weight != vdr.Weight:
				t.Fatalf("expected weight of %s to be %v but got %v",
					vdr.NodeID,
					vdr.Weight,
					gotVdr.Weight,
				)
			default:
				found = true
			}
		}
		if !found {
			t.Fatalf("expected validators to contain %s but didn't", vdr.NodeID)
		}
	}

	// Add a nominator
	stakeAmt := service.vm.MinNominatorStake + 12345
	validatorNodeID := ids.NodeID(keys[1].PublicKey().Address())
	nominatorStartTime := uint64(defaultValidateStartTime.Unix())
	nominatorEndTime := uint64(defaultValidateStartTime.Add(defaultMinStakingDuration).Unix())

	tx, err := service.vm.newAddNominatorTx(
		stakeAmt,
		nominatorStartTime,
		nominatorEndTime,
		validatorNodeID,
		ids.GenerateTestShortID(),
		[]*crypto.PrivateKeySECP256K1R{keys[0]},
		keys[0].PublicKey().Address(), // change addr
	)
	if err != nil {
		t.Fatal(err)
	}

	service.vm.internalState.AddCurrentStaker(tx, 0)
	service.vm.internalState.AddTx(tx, status.Committed)
	err = service.vm.internalState.Commit()
	if err != nil {
		t.Fatal(err)
	}
	err = service.vm.internalState.(*internalStateImpl).loadCurrentValidators()
	if err != nil {
		t.Fatal(err)
	}

	// Call getCurrentValidators
	args = GetCurrentValidatorsArgs{AllychainID: constants.PrimaryNetworkID}
	err = service.GetCurrentValidators(nil, &args, &response)
	switch {
	case err != nil:
		t.Fatal(err)
	case len(response.Validators) != len(genesis.Validators):
		t.Fatalf("should be %d validators but are %d", len(genesis.Validators), len(response.Validators))
	}

	// Make sure the nominator is there
	found := false
	for i := 0; i < len(response.Validators) && !found; i++ {
		vdr := response.Validators[i].(APIPrimaryValidator)
		if vdr.NodeID != validatorNodeID {
			continue
		}
		found = true
		if len(vdr.Nominators) != 1 {
			t.Fatalf("%s should have 1 nominator", vdr.NodeID)
		}
		nominator := vdr.Nominators[0]
		switch {
		case nominator.NodeID != vdr.NodeID:
			t.Fatal("wrong node ID")
		case uint64(nominator.StartTime) != nominatorStartTime:
			t.Fatal("wrong start time")
		case uint64(nominator.EndTime) != nominatorEndTime:
			t.Fatal("wrong end time")
		case nominator.weight() != stakeAmt:
			t.Fatalf("wrong weight")
		}
	}
	if !found {
		t.Fatalf("didnt find nominator")
	}
}

func TestGetTimestamp(t *testing.T) {
	assert := assert.New(t)

	service := defaultService(t)
	service.vm.ctx.Lock.Lock()
	defer func() {
		err := service.vm.Shutdown()
		assert.NoError(err)

		service.vm.ctx.Lock.Unlock()
	}()

	reply := GetTimestampReply{}
	err := service.GetTimestamp(nil, nil, &reply)
	assert.NoError(err)

	assert.Equal(service.vm.internalState.GetTimestamp(), reply.Timestamp)

	newTimestamp := reply.Timestamp.Add(time.Second)
	service.vm.internalState.SetTimestamp(newTimestamp)

	err = service.GetTimestamp(nil, nil, &reply)
	assert.NoError(err)

	assert.Equal(newTimestamp, reply.Timestamp)
}

func TestGetBlock(t *testing.T) {
	tests := []struct {
		name     string
		encoding formatting.Encoding
	}{
		{
			name:     "json",
			encoding: formatting.JSON,
		},
		{
			name:     "cb58",
			encoding: formatting.CB58,
		},
		{
			name:     "hex",
			encoding: formatting.Hex,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := defaultService(t)

			block, err := service.vm.newStandardBlock(ids.GenerateTestID(), 1234, nil)
			if err != nil {
				t.Fatal("couldn't create block: %w", err)
			}
			internalState := NewMockInternalState(ctrl)
			internalState.EXPECT().GetBlock(block.ID()).Times(1).Return(block, nil)

			service.vm.internalState = internalState

			args := api.GetBlockArgs{
				BlockID:  block.ID(),
				Encoding: test.encoding,
			}
			response := api.GetBlockResponse{}
			err = service.GetBlock(nil, &args, &response)
			if err != nil {
				t.Fatal(err)
			}

			switch {
			case test.encoding == formatting.JSON:
				assert.Equal(t, block, response.Block)
			default:
				decoded, _ := formatting.Decode(response.Encoding, response.Block.(string))
				assert.Equal(t, block.Bytes(), decoded)
			}

			assert.Equal(t, test.encoding, response.Encoding)
		})
	}
}
