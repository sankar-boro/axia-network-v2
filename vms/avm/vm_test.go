// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"bytes"
	"errors"
	"math"
	"testing"

	stdjson "encoding/json"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/api/keystore"
	"github.com/sankar-boro/axia-network-v2/chains/atomic"
	"github.com/sankar-boro/axia-network-v2/database/manager"
	"github.com/sankar-boro/axia-network-v2/database/mockdb"
	"github.com/sankar-boro/axia-network-v2/database/prefixdb"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow"
	"github.com/sankar-boro/axia-network-v2/snow/engine/common"
	"github.com/sankar-boro/axia-network-v2/utils/crypto"
	"github.com/sankar-boro/axia-network-v2/utils/formatting"
	"github.com/sankar-boro/axia-network-v2/utils/formatting/address"
	"github.com/sankar-boro/axia-network-v2/utils/json"
	"github.com/sankar-boro/axia-network-v2/utils/logging"
	"github.com/sankar-boro/axia-network-v2/utils/wrappers"
	"github.com/sankar-boro/axia-network-v2/version"
	"github.com/sankar-boro/axia-network-v2/vms/avm/fxs"
	"github.com/sankar-boro/axia-network-v2/vms/avm/states"
	"github.com/sankar-boro/axia-network-v2/vms/avm/txs"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/components/verify"
	"github.com/sankar-boro/axia-network-v2/vms/nftfx"
	"github.com/sankar-boro/axia-network-v2/vms/propertyfx"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
)

var (
	networkID       uint32 = 10
	chainID                = ids.ID{5, 4, 3, 2, 1}
	platformChainID        = ids.Empty.Prefix(0)
	testTxFee              = uint64(1000)
	startBalance           = uint64(50000)

	keys  []*crypto.PrivateKeySECP256K1R
	addrs []ids.ShortID // addrs[i] corresponds to keys[i]

	assetID        = ids.ID{1, 2, 3}
	username       = "bobby"
	password       = "StrnasfqewiurPasswdn56d" // #nosec G101
	feeAssetName   = "TEST"
	otherAssetName = "OTHER"
)

func init() {
	factory := crypto.FactorySECP256K1R{}

	for _, key := range []string{
		"24jUJ9vZexUM6expyMcT48LBx27k1m7xpraoV62oSQAHdziao5",
		"2MMvUMsxx6zsHSNXJdFD8yc5XkancvwyKPwpw4xUK3TCGDuNBY",
		"cxb7KpGWhDMALTjNNSJ7UQkkomPesyWAPUaWRGdyeBNzR6f35",
	} {
		keyBytes, _ := formatting.Decode(formatting.CB58, key)
		pk, _ := factory.ToPrivateKey(keyBytes)
		keys = append(keys, pk.(*crypto.PrivateKeySECP256K1R))
		addrs = append(addrs, pk.PublicKey().Address())
	}
}

type snLookup struct {
	chainsToSubnet map[ids.ID]ids.ID
}

func (sn *snLookup) SubnetID(chainID ids.ID) (ids.ID, error) {
	subnetID, ok := sn.chainsToSubnet[chainID]
	if !ok {
		return ids.ID{}, errors.New("")
	}
	return subnetID, nil
}

func NewContext(tb testing.TB) *snow.Context {
	genesisBytes := BuildGenesisTest(tb)

	tx := GetAXCTxFromGenesisTest(genesisBytes, tb)

	ctx := snow.DefaultContextTest()
	ctx.NetworkID = networkID
	ctx.ChainID = chainID
	ctx.AXCAssetID = tx.ID()
	ctx.SwapChainID = ids.Empty.Prefix(0)
	aliaser := ctx.BCLookup.(ids.Aliaser)

	errs := wrappers.Errs{}
	errs.Add(
		aliaser.Alias(chainID, "Swap"),
		aliaser.Alias(chainID, chainID.String()),
		aliaser.Alias(platformChainID, "Core"),
		aliaser.Alias(platformChainID, platformChainID.String()),
	)
	if errs.Errored() {
		tb.Fatal(errs.Err)
	}

	sn := &snLookup{
		chainsToSubnet: make(map[ids.ID]ids.ID),
	}
	sn.chainsToSubnet[chainID] = ctx.SubnetID
	sn.chainsToSubnet[platformChainID] = ctx.SubnetID
	ctx.SNLookup = sn
	return ctx
}

// Returns:
//   1) tx in genesis that creates asset
//   2) the index of the output
func GetCreateTxFromGenesisTest(tb testing.TB, genesisBytes []byte, assetName string) *txs.Tx {
	parser, err := txs.NewParser([]fxs.Fx{
		&secp256k1fx.Fx{},
	})
	if err != nil {
		tb.Fatal(err)
	}

	cm := parser.GenesisCodec()
	genesis := Genesis{}
	if _, err := cm.Unmarshal(genesisBytes, &genesis); err != nil {
		tb.Fatal(err)
	}

	if len(genesis.Txs) == 0 {
		tb.Fatal("genesis tx didn't have any txs")
	}

	var assetTx *GenesisAsset
	for _, tx := range genesis.Txs {
		if tx.Name == assetName {
			assetTx = tx
			break
		}
	}
	if assetTx == nil {
		tb.Fatal("there is no create tx")
		return nil
	}

	tx := &txs.Tx{
		UnsignedTx: &assetTx.CreateAssetTx,
	}
	if err := parser.InitializeGenesisTx(tx); err != nil {
		tb.Fatal(err)
	}
	return tx
}

func GetAXCTxFromGenesisTest(genesisBytes []byte, tb testing.TB) *txs.Tx {
	return GetCreateTxFromGenesisTest(tb, genesisBytes, "AXC")
}

// BuildGenesisTest is the common Genesis builder for most tests
func BuildGenesisTest(tb testing.TB) []byte {
	addr0Str, _ := address.FormatBech32(testHRP, addrs[0].Bytes())
	addr1Str, _ := address.FormatBech32(testHRP, addrs[1].Bytes())
	addr2Str, _ := address.FormatBech32(testHRP, addrs[2].Bytes())

	defaultArgs := &BuildGenesisArgs{
		Encoding: formatting.Hex,
		GenesisData: map[string]AssetDefinition{
			"asset1": {
				Name:   "AXC",
				Symbol: "SYMB",
				InitialState: map[string][]interface{}{
					"fixedCap": {
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr0Str,
						},
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr1Str,
						},
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr2Str,
						},
					},
				},
			},
			"asset2": {
				Name:   "myVarCapAsset",
				Symbol: "MVCA",
				InitialState: map[string][]interface{}{
					"variableCap": {
						Owners{
							Threshold: 1,
							Minters: []string{
								addr0Str,
								addr1Str,
							},
						},
						Owners{
							Threshold: 2,
							Minters: []string{
								addr0Str,
								addr1Str,
								addr2Str,
							},
						},
					},
				},
			},
			"asset3": {
				Name: "myOtherVarCapAsset",
				InitialState: map[string][]interface{}{
					"variableCap": {
						Owners{
							Threshold: 1,
							Minters: []string{
								addr0Str,
							},
						},
					},
				},
			},
		},
	}

	return BuildGenesisTestWithArgs(tb, defaultArgs)
}

// BuildGenesisTestWithArgs allows building the genesis while injecting different starting points (args)
func BuildGenesisTestWithArgs(tb testing.TB, args *BuildGenesisArgs) []byte {
	ss := CreateStaticService()

	reply := BuildGenesisReply{}
	err := ss.BuildGenesis(nil, args, &reply)
	if err != nil {
		tb.Fatal(err)
	}

	b, err := formatting.Decode(reply.Encoding, reply.Bytes)
	if err != nil {
		tb.Fatal(err)
	}

	return b
}

func GenesisVM(tb testing.TB) ([]byte, chan common.Message, *VM, *atomic.Memory) {
	return GenesisVMWithArgs(tb, nil, nil)
}

func GenesisVMWithArgs(tb testing.TB, additionalFxs []*common.Fx, args *BuildGenesisArgs) ([]byte, chan common.Message, *VM, *atomic.Memory) {
	var genesisBytes []byte

	if args != nil {
		genesisBytes = BuildGenesisTestWithArgs(tb, args)
	} else {
		genesisBytes = BuildGenesisTest(tb)
	}

	ctx := NewContext(tb)

	baseDBManager := manager.NewMemDB(version.DefaultVersion1_0_0)

	m := &atomic.Memory{}
	err := m.Initialize(logging.NoLog{}, prefixdb.New([]byte{0}, baseDBManager.Current().Database))
	if err != nil {
		tb.Fatal(err)
	}
	ctx.SharedMemory = m.NewSharedMemory(ctx.ChainID)

	// NB: this lock is intentionally left locked when this function returns.
	// The caller of this function is responsible for unlocking.
	ctx.Lock.Lock()

	userKeystore, err := keystore.CreateTestKeystore()
	if err != nil {
		tb.Fatal(err)
	}
	if err := userKeystore.CreateUser(username, password); err != nil {
		tb.Fatal(err)
	}
	ctx.Keystore = userKeystore.NewBlockchainKeyStore(ctx.ChainID)

	issuer := make(chan common.Message, 1)
	vm := &VM{Factory: Factory{
		TxFee:            testTxFee,
		CreateAssetTxFee: testTxFee,
	}}
	configBytes, err := stdjson.Marshal(Config{IndexTransactions: true})
	if err != nil {
		tb.Fatal("should not have caused error in creating avm config bytes")
	}
	err = vm.Initialize(
		ctx,
		baseDBManager.NewPrefixDBManager([]byte{1}),
		genesisBytes,
		nil,
		configBytes,
		issuer,
		append(
			[]*common.Fx{
				{
					ID: ids.Empty,
					Fx: &secp256k1fx.Fx{},
				},
				{
					ID: nftfx.ID,
					Fx: &nftfx.Fx{},
				},
			},
			additionalFxs...,
		),
		nil,
	)
	if err != nil {
		tb.Fatal(err)
	}
	vm.batchTimeout = 0

	if err := vm.SetState(snow.Bootstrapping); err != nil {
		tb.Fatal(err)
	}

	if err := vm.SetState(snow.NormalOp); err != nil {
		tb.Fatal(err)
	}

	return genesisBytes, issuer, vm, m
}

func NewTx(t *testing.T, genesisBytes []byte, vm *VM) *txs.Tx {
	return NewTxWithAsset(t, genesisBytes, vm, "AXC")
}

func NewTxWithAsset(t *testing.T, genesisBytes []byte, vm *VM, assetName string) *txs.Tx {
	createTx := GetCreateTxFromGenesisTest(t, genesisBytes, assetName)

	newTx := &txs.Tx{
		UnsignedTx: &txs.BaseTx{
			BaseTx: axc.BaseTx{
				NetworkID:    networkID,
				BlockchainID: chainID,
				Ins: []*axc.TransferableInput{{
					UTXOID: axc.UTXOID{
						TxID:        createTx.ID(),
						OutputIndex: 2,
					},
					Asset: axc.Asset{ID: createTx.ID()},
					In: &secp256k1fx.TransferInput{
						Amt: startBalance,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				}},
			},
		},
	}
	if err := newTx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{keys[0]}}); err != nil {
		t.Fatal(err)
	}
	return newTx
}

func setupIssueTx(t testing.TB) (chan common.Message, *VM, *snow.Context, []*txs.Tx) {
	genesisBytes, issuer, vm, _ := GenesisVM(t)
	ctx := vm.ctx

	axcTx := GetAXCTxFromGenesisTest(genesisBytes, t)
	key := keys[0]
	firstTx := &txs.Tx{
		UnsignedTx: &txs.BaseTx{
			BaseTx: axc.BaseTx{
				NetworkID:    networkID,
				BlockchainID: chainID,
				Ins: []*axc.TransferableInput{{
					UTXOID: axc.UTXOID{
						TxID:        axcTx.ID(),
						OutputIndex: 2,
					},
					Asset: axc.Asset{ID: axcTx.ID()},
					In: &secp256k1fx.TransferInput{
						Amt: startBalance,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				}},
				Outs: []*axc.TransferableOutput{{
					Asset: axc.Asset{ID: axcTx.ID()},
					Out: &secp256k1fx.TransferOutput{
						Amt: startBalance - vm.TxFee,
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{key.PublicKey().Address()},
						},
					},
				}},
			},
		},
	}
	if err := firstTx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{key}}); err != nil {
		t.Fatal(err)
	}

	secondTx := &txs.Tx{
		UnsignedTx: &txs.BaseTx{
			BaseTx: axc.BaseTx{
				NetworkID:    networkID,
				BlockchainID: chainID,
				Ins: []*axc.TransferableInput{{
					UTXOID: axc.UTXOID{
						TxID:        axcTx.ID(),
						OutputIndex: 2,
					},
					Asset: axc.Asset{ID: axcTx.ID()},
					In: &secp256k1fx.TransferInput{
						Amt: startBalance,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				}},
				Outs: []*axc.TransferableOutput{{
					Asset: axc.Asset{ID: axcTx.ID()},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{key.PublicKey().Address()},
						},
					},
				}},
			},
		},
	}
	if err := secondTx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{key}}); err != nil {
		t.Fatal(err)
	}
	return issuer, vm, ctx, []*txs.Tx{axcTx, firstTx, secondTx}
}

func TestInvalidGenesis(t *testing.T) {
	vm := &VM{}
	ctx := NewContext(t)
	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	err := vm.Initialize(
		ctx, // context
		manager.NewMemDB(version.DefaultVersion1_0_0), // dbManager
		nil,                          // genesisState
		nil,                          // upgradeBytes
		nil,                          // configBytes
		make(chan common.Message, 1), // engineMessenger
		nil,                          // fxs
		nil,                          // AppSender
	)
	if err == nil {
		t.Fatalf("Should have erred due to an invalid genesis")
	}
}

func TestInvalidFx(t *testing.T) {
	vm := &VM{}
	ctx := NewContext(t)
	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	genesisBytes := BuildGenesisTest(t)
	err := vm.Initialize(
		ctx, // context
		manager.NewMemDB(version.DefaultVersion1_0_0), // dbManager
		genesisBytes,                 // genesisState
		nil,                          // upgradeBytes
		nil,                          // configBytes
		make(chan common.Message, 1), // engineMessenger
		[]*common.Fx{ // fxs
			nil,
		},
		nil,
	)
	if err == nil {
		t.Fatalf("Should have erred due to an invalid interface")
	}
}

func TestFxInitializationFailure(t *testing.T) {
	vm := &VM{}
	ctx := NewContext(t)
	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	genesisBytes := BuildGenesisTest(t)
	err := vm.Initialize(
		ctx, // context
		manager.NewMemDB(version.DefaultVersion1_0_0), // dbManager
		genesisBytes,                 // genesisState
		nil,                          // upgradeBytes
		nil,                          // configBytes
		make(chan common.Message, 1), // engineMessenger
		[]*common.Fx{{ // fxs
			ID: ids.Empty,
			Fx: &FxTest{
				InitializeF: func(interface{}) error {
					return errUnknownFx
				},
			},
		}},
		nil,
	)
	if err == nil {
		t.Fatalf("Should have erred due to an invalid fx initialization")
	}
}

func TestIssueTx(t *testing.T) {
	genesisBytes, issuer, vm, _ := GenesisVM(t)
	ctx := vm.ctx
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	newTx := NewTx(t, genesisBytes, vm)

	txID, err := vm.IssueTx(newTx.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if txID != newTx.ID() {
		t.Fatalf("Issue Tx returned wrong TxID")
	}
	ctx.Lock.Unlock()

	msg := <-issuer
	if msg != common.PendingTxs {
		t.Fatalf("Wrong message")
	}
	ctx.Lock.Lock()

	if txs := vm.PendingTxs(); len(txs) != 1 {
		t.Fatalf("Should have returned %d tx(s)", 1)
	}
}

// Test issuing a transaction that consumes a currently pending UTXO. The
// transaction should be issued successfully.
func TestIssueDependentTx(t *testing.T) {
	issuer, vm, ctx, txs := setupIssueTx(t)
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	firstTx := txs[1]
	secondTx := txs[2]

	if _, err := vm.IssueTx(firstTx.Bytes()); err != nil {
		t.Fatal(err)
	}

	if _, err := vm.IssueTx(secondTx.Bytes()); err != nil {
		t.Fatal(err)
	}
	ctx.Lock.Unlock()

	msg := <-issuer
	if msg != common.PendingTxs {
		t.Fatalf("Wrong message")
	}
	ctx.Lock.Lock()

	if txs := vm.PendingTxs(); len(txs) != 2 {
		t.Fatalf("Should have returned %d tx(s)", 2)
	}
}

// Test issuing a transaction that creates an NFT family
func TestIssueNFT(t *testing.T) {
	vm := &VM{}
	ctx := NewContext(t)
	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	genesisBytes := BuildGenesisTest(t)
	issuer := make(chan common.Message, 1)
	err := vm.Initialize(
		ctx,
		manager.NewMemDB(version.DefaultVersion1_0_0),
		genesisBytes,
		nil,
		nil,
		issuer,
		[]*common.Fx{
			{
				ID: ids.Empty.Prefix(0),
				Fx: &secp256k1fx.Fx{},
			},
			{
				ID: ids.Empty.Prefix(1),
				Fx: &nftfx.Fx{},
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	vm.batchTimeout = 0

	err = vm.SetState(snow.Bootstrapping)
	if err != nil {
		t.Fatal(err)
	}

	err = vm.SetState(snow.NormalOp)
	if err != nil {
		t.Fatal(err)
	}

	createAssetTx := &txs.Tx{UnsignedTx: &txs.CreateAssetTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
		}},
		Name:         "Team Rocket",
		Symbol:       "TR",
		Denomination: 0,
		States: []*txs.InitialState{{
			FxIndex: 1,
			Outs: []verify.State{
				&nftfx.MintOutput{
					GroupID: 1,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{keys[0].PublicKey().Address()},
					},
				},
				&nftfx.MintOutput{
					GroupID: 2,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{keys[0].PublicKey().Address()},
					},
				},
			},
		}},
	}}
	if err := vm.parser.InitializeTx(createAssetTx); err != nil {
		t.Fatal(err)
	}

	if _, err = vm.IssueTx(createAssetTx.Bytes()); err != nil {
		t.Fatal(err)
	}

	mintNFTTx := &txs.Tx{UnsignedTx: &txs.OperationTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
		}},
		Ops: []*txs.Operation{{
			Asset: axc.Asset{ID: createAssetTx.ID()},
			UTXOIDs: []*axc.UTXOID{{
				TxID:        createAssetTx.ID(),
				OutputIndex: 0,
			}},
			Op: &nftfx.MintOperation{
				MintInput: secp256k1fx.Input{
					SigIndices: []uint32{0},
				},
				GroupID: 1,
				Payload: []byte{'h', 'e', 'l', 'l', 'o'},
				Outputs: []*secp256k1fx.OutputOwners{{}},
			},
		}},
	}}
	if err := mintNFTTx.SignNFTFx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{keys[0]}}); err != nil {
		t.Fatal(err)
	}

	if _, err = vm.IssueTx(mintNFTTx.Bytes()); err != nil {
		t.Fatal(err)
	}

	transferNFTTx := &txs.Tx{
		UnsignedTx: &txs.OperationTx{
			BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
				NetworkID:    networkID,
				BlockchainID: chainID,
			}},
			Ops: []*txs.Operation{{
				Asset: axc.Asset{ID: createAssetTx.ID()},
				UTXOIDs: []*axc.UTXOID{{
					TxID:        mintNFTTx.ID(),
					OutputIndex: 0,
				}},
				Op: &nftfx.TransferOperation{
					Input: secp256k1fx.Input{},
					Output: nftfx.TransferOutput{
						GroupID:      1,
						Payload:      []byte{'h', 'e', 'l', 'l', 'o'},
						OutputOwners: secp256k1fx.OutputOwners{},
					},
				},
			}},
		},
		Creds: []*fxs.FxCredential{
			{Verifiable: &nftfx.Credential{}},
		},
	}
	if err := vm.parser.InitializeTx(transferNFTTx); err != nil {
		t.Fatal(err)
	}

	if _, err = vm.IssueTx(transferNFTTx.Bytes()); err != nil {
		t.Fatal(err)
	}
}

// Test issuing a transaction that creates an Property family
func TestIssueProperty(t *testing.T) {
	vm := &VM{}
	ctx := NewContext(t)
	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	genesisBytes := BuildGenesisTest(t)
	issuer := make(chan common.Message, 1)
	err := vm.Initialize(
		ctx,
		manager.NewMemDB(version.DefaultVersion1_0_0),
		genesisBytes,
		nil,
		nil,
		issuer,
		[]*common.Fx{
			{
				ID: ids.Empty.Prefix(0),
				Fx: &secp256k1fx.Fx{},
			},
			{
				ID: ids.Empty.Prefix(1),
				Fx: &nftfx.Fx{},
			},
			{
				ID: ids.Empty.Prefix(2),
				Fx: &propertyfx.Fx{},
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	vm.batchTimeout = 0

	err = vm.SetState(snow.Bootstrapping)
	if err != nil {
		t.Fatal(err)
	}

	err = vm.SetState(snow.NormalOp)
	if err != nil {
		t.Fatal(err)
	}

	createAssetTx := &txs.Tx{UnsignedTx: &txs.CreateAssetTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
		}},
		Name:         "Team Rocket",
		Symbol:       "TR",
		Denomination: 0,
		States: []*txs.InitialState{{
			FxIndex: 2,
			Outs: []verify.State{
				&propertyfx.MintOutput{
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{keys[0].PublicKey().Address()},
					},
				},
			},
		}},
	}}
	if err := vm.parser.InitializeTx(createAssetTx); err != nil {
		t.Fatal(err)
	}

	if _, err = vm.IssueTx(createAssetTx.Bytes()); err != nil {
		t.Fatal(err)
	}

	mintPropertyTx := &txs.Tx{UnsignedTx: &txs.OperationTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
		}},
		Ops: []*txs.Operation{{
			Asset: axc.Asset{ID: createAssetTx.ID()},
			UTXOIDs: []*axc.UTXOID{{
				TxID:        createAssetTx.ID(),
				OutputIndex: 0,
			}},
			Op: &propertyfx.MintOperation{
				MintInput: secp256k1fx.Input{
					SigIndices: []uint32{0},
				},
				MintOutput: propertyfx.MintOutput{
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{keys[0].PublicKey().Address()},
					},
				},
				OwnedOutput: propertyfx.OwnedOutput{},
			},
		}},
	}}

	codec := vm.parser.Codec()
	err = mintPropertyTx.SignPropertyFx(codec, [][]*crypto.PrivateKeySECP256K1R{
		{keys[0]},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = vm.IssueTx(mintPropertyTx.Bytes()); err != nil {
		t.Fatal(err)
	}

	burnPropertyTx := &txs.Tx{UnsignedTx: &txs.OperationTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
		}},
		Ops: []*txs.Operation{{
			Asset: axc.Asset{ID: createAssetTx.ID()},
			UTXOIDs: []*axc.UTXOID{{
				TxID:        mintPropertyTx.ID(),
				OutputIndex: 1,
			}},
			Op: &propertyfx.BurnOperation{Input: secp256k1fx.Input{}},
		}},
	}}

	err = burnPropertyTx.SignPropertyFx(codec, [][]*crypto.PrivateKeySECP256K1R{
		{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = vm.IssueTx(burnPropertyTx.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func setupTxFeeAssets(t *testing.T) ([]byte, chan common.Message, *VM, *atomic.Memory) {
	addr0Str, _ := address.FormatBech32(testHRP, addrs[0].Bytes())
	addr1Str, _ := address.FormatBech32(testHRP, addrs[1].Bytes())
	addr2Str, _ := address.FormatBech32(testHRP, addrs[2].Bytes())
	assetAlias := "asset1"
	customArgs := &BuildGenesisArgs{
		Encoding: formatting.Hex,
		GenesisData: map[string]AssetDefinition{
			assetAlias: {
				Name:   feeAssetName,
				Symbol: "TST",
				InitialState: map[string][]interface{}{
					"fixedCap": {
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr0Str,
						},
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr1Str,
						},
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr2Str,
						},
					},
				},
			},
			"asset2": {
				Name:   otherAssetName,
				Symbol: "OTH",
				InitialState: map[string][]interface{}{
					"fixedCap": {
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr0Str,
						},
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr1Str,
						},
						Holder{
							Amount:  json.Uint64(startBalance),
							Address: addr2Str,
						},
					},
				},
			},
		},
	}
	genesisBytes, issuer, vm, m := GenesisVMWithArgs(t, nil, customArgs)
	expectedID, err := vm.Aliaser.Lookup(assetAlias)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, vm.feeAssetID)
	return genesisBytes, issuer, vm, m
}

func TestIssueTxWithFeeAsset(t *testing.T) {
	genesisBytes, issuer, vm, _ := setupTxFeeAssets(t)
	ctx := vm.ctx
	defer func() {
		err := vm.Shutdown()
		assert.NoError(t, err)
		ctx.Lock.Unlock()
	}()
	// send first asset
	newTx := NewTxWithAsset(t, genesisBytes, vm, feeAssetName)

	txID, err := vm.IssueTx(newTx.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, txID, newTx.ID())

	ctx.Lock.Unlock()

	msg := <-issuer
	assert.Equal(t, msg, common.PendingTxs)

	ctx.Lock.Lock()
	assert.Len(t, vm.PendingTxs(), 1)
	t.Log(vm.PendingTxs())
}

func TestIssueTxWithAnotherAsset(t *testing.T) {
	genesisBytes, issuer, vm, _ := setupTxFeeAssets(t)
	ctx := vm.ctx
	defer func() {
		err := vm.Shutdown()
		assert.NoError(t, err)
		ctx.Lock.Unlock()
	}()

	// send second asset
	feeAssetCreateTx := GetCreateTxFromGenesisTest(t, genesisBytes, feeAssetName)
	createTx := GetCreateTxFromGenesisTest(t, genesisBytes, otherAssetName)

	newTx := &txs.Tx{UnsignedTx: &txs.BaseTx{
		BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*axc.TransferableInput{
				// fee asset
				{
					UTXOID: axc.UTXOID{
						TxID:        feeAssetCreateTx.ID(),
						OutputIndex: 2,
					},
					Asset: axc.Asset{ID: feeAssetCreateTx.ID()},
					In: &secp256k1fx.TransferInput{
						Amt: startBalance,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
				// issued asset
				{
					UTXOID: axc.UTXOID{
						TxID:        createTx.ID(),
						OutputIndex: 2,
					},
					Asset: axc.Asset{ID: createTx.ID()},
					In: &secp256k1fx.TransferInput{
						Amt: startBalance,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
			},
		},
	}}
	if err := newTx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{keys[0]}, {keys[0]}}); err != nil {
		t.Fatal(err)
	}

	txID, err := vm.IssueTx(newTx.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, txID, newTx.ID())

	ctx.Lock.Unlock()

	msg := <-issuer
	assert.Equal(t, msg, common.PendingTxs)

	ctx.Lock.Lock()
	assert.Len(t, vm.PendingTxs(), 1)
}

func TestVMFormat(t *testing.T) {
	_, _, vm, _ := GenesisVM(t)
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		vm.ctx.Lock.Unlock()
	}()

	tests := []struct {
		in       ids.ShortID
		expected string
	}{
		{ids.ShortEmpty, "Swap-testing1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqtu2yas"},
	}
	for _, test := range tests {
		t.Run(test.in.String(), func(t *testing.T) {
			addrStr, err := vm.FormatLocalAddress(test.in)
			if err != nil {
				t.Error(err)
			}
			if test.expected != addrStr {
				t.Errorf("Expected %q, got %q", test.expected, addrStr)
			}
		})
	}
}

func TestTxCached(t *testing.T) {
	genesisBytes, _, vm, _ := GenesisVM(t)
	ctx := vm.ctx
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	newTx := NewTx(t, genesisBytes, vm)
	txBytes := newTx.Bytes()

	_, err := vm.ParseTx(txBytes)
	assert.NoError(t, err)

	db := mockdb.New()
	called := new(bool)
	db.OnGet = func([]byte) ([]byte, error) {
		*called = true
		return nil, errors.New("")
	}

	registerer := prometheus.NewRegistry()

	err = vm.metrics.Initialize("", registerer)
	assert.NoError(t, err)

	vm.state, err = states.New(prefixdb.New([]byte("tx"), db), vm.parser, registerer)
	assert.NoError(t, err)

	_, err = vm.ParseTx(txBytes)
	assert.NoError(t, err)
	assert.False(t, *called, "shouldn't have called the DB")
}

func TestTxNotCached(t *testing.T) {
	genesisBytes, _, vm, _ := GenesisVM(t)
	ctx := vm.ctx
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	newTx := NewTx(t, genesisBytes, vm)
	txBytes := newTx.Bytes()

	_, err := vm.ParseTx(txBytes)
	assert.NoError(t, err)

	db := mockdb.New()
	called := new(bool)
	db.OnGet = func([]byte) ([]byte, error) {
		*called = true
		return nil, errors.New("")
	}
	db.OnPut = func([]byte, []byte) error { return nil }

	registerer := prometheus.NewRegistry()
	assert.NoError(t, err)

	err = vm.metrics.Initialize("", registerer)
	assert.NoError(t, err)

	vm.state, err = states.New(db, vm.parser, registerer)
	assert.NoError(t, err)

	vm.uniqueTxs.Flush()

	_, err = vm.ParseTx(txBytes)
	assert.NoError(t, err)
	assert.True(t, *called, "should have called the DB")
}

func TestTxVerifyAfterIssueTx(t *testing.T) {
	issuer, vm, ctx, issueTxs := setupIssueTx(t)
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()
	firstTx := issueTxs[1]
	secondTx := issueTxs[2]
	parsedSecondTx, err := vm.ParseTx(secondTx.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := parsedSecondTx.Verify(); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.IssueTx(firstTx.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := parsedSecondTx.Accept(); err != nil {
		t.Fatal(err)
	}
	ctx.Lock.Unlock()

	msg := <-issuer
	if msg != common.PendingTxs {
		t.Fatalf("Wrong message")
	}
	ctx.Lock.Lock()

	txs := vm.PendingTxs()
	if len(txs) != 1 {
		t.Fatalf("Should have returned %d tx(s)", 1)
	}
	parsedFirstTx := txs[0]

	if err := parsedFirstTx.Verify(); err == nil {
		t.Fatalf("Should have erred due to a missing UTXO")
	}
}

func TestTxVerifyAfterGet(t *testing.T) {
	_, vm, ctx, issueTxs := setupIssueTx(t)
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()
	firstTx := issueTxs[1]
	secondTx := issueTxs[2]

	parsedSecondTx, err := vm.ParseTx(secondTx.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := parsedSecondTx.Verify(); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.IssueTx(firstTx.Bytes()); err != nil {
		t.Fatal(err)
	}
	parsedFirstTx, err := vm.GetTx(firstTx.ID())
	if err != nil {
		t.Fatal(err)
	}
	if err := parsedSecondTx.Accept(); err != nil {
		t.Fatal(err)
	}
	if err := parsedFirstTx.Verify(); err == nil {
		t.Fatalf("Should have erred due to a missing UTXO")
	}
}

func TestTxVerifyAfterVerifyAncestorTx(t *testing.T) {
	_, vm, ctx, issueTxs := setupIssueTx(t)
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()
	axcTx := issueTxs[0]
	firstTx := issueTxs[1]
	secondTx := issueTxs[2]
	key := keys[0]
	firstTxDescendant := &txs.Tx{UnsignedTx: &txs.BaseTx{BaseTx: axc.BaseTx{
		NetworkID:    networkID,
		BlockchainID: chainID,
		Ins: []*axc.TransferableInput{{
			UTXOID: axc.UTXOID{
				TxID:        firstTx.ID(),
				OutputIndex: 0,
			},
			Asset: axc.Asset{ID: axcTx.ID()},
			In: &secp256k1fx.TransferInput{
				Amt: startBalance - vm.TxFee,
				Input: secp256k1fx.Input{
					SigIndices: []uint32{
						0,
					},
				},
			},
		}},
		Outs: []*axc.TransferableOutput{{
			Asset: axc.Asset{ID: axcTx.ID()},
			Out: &secp256k1fx.TransferOutput{
				Amt: startBalance - 2*vm.TxFee,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{key.PublicKey().Address()},
				},
			},
		}},
	}}}
	if err := firstTxDescendant.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{key}}); err != nil {
		t.Fatal(err)
	}

	parsedSecondTx, err := vm.ParseTx(secondTx.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := parsedSecondTx.Verify(); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.IssueTx(firstTx.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.IssueTx(firstTxDescendant.Bytes()); err != nil {
		t.Fatal(err)
	}
	parsedFirstTx, err := vm.GetTx(firstTx.ID())
	if err != nil {
		t.Fatal(err)
	}
	if err := parsedSecondTx.Accept(); err != nil {
		t.Fatal(err)
	}
	if err := parsedFirstTx.Verify(); err == nil {
		t.Fatalf("Should have erred due to a missing UTXO")
	}
}

func TestImportTxSerialization(t *testing.T) {
	_, vm, _, _ := setupIssueTx(t)
	expected := []byte{
		// Codec version
		0x00, 0x00,
		// txID:
		0x00, 0x00, 0x00, 0x03,
		// networkID:
		0x00, 0x00, 0x00, 0x02,
		// blockchainID:
		0xff, 0xff, 0xff, 0xff, 0xee, 0xee, 0xee, 0xee,
		0xdd, 0xdd, 0xdd, 0xdd, 0xcc, 0xcc, 0xcc, 0xcc,
		0xbb, 0xbb, 0xbb, 0xbb, 0xaa, 0xaa, 0xaa, 0xaa,
		0x99, 0x99, 0x99, 0x99, 0x88, 0x88, 0x88, 0x88,
		// number of base outs:
		0x00, 0x00, 0x00, 0x00,
		// number of base inputs:
		0x00, 0x00, 0x00, 0x00,
		// Memo length:
		0x00, 0x00, 0x00, 0x04,
		// Memo:
		0x00, 0x01, 0x02, 0x03,
		// Source Chain ID:
		0x1f, 0x8f, 0x9f, 0x0f, 0x1e, 0x8e, 0x9e, 0x0e,
		0x2d, 0x7d, 0xad, 0xfd, 0x2c, 0x7c, 0xac, 0xfc,
		0x3b, 0x6b, 0xbb, 0xeb, 0x3a, 0x6a, 0xba, 0xea,
		0x49, 0x59, 0xc9, 0xd9, 0x48, 0x58, 0xc8, 0xd8,
		// number of inputs:
		0x00, 0x00, 0x00, 0x01,
		// utxoID:
		0x0f, 0x2f, 0x4f, 0x6f, 0x8e, 0xae, 0xce, 0xee,
		0x0d, 0x2d, 0x4d, 0x6d, 0x8c, 0xac, 0xcc, 0xec,
		0x0b, 0x2b, 0x4b, 0x6b, 0x8a, 0xaa, 0xca, 0xea,
		0x09, 0x29, 0x49, 0x69, 0x88, 0xa8, 0xc8, 0xe8,
		// output index
		0x00, 0x00, 0x00, 0x00,
		// assetID:
		0x1f, 0x3f, 0x5f, 0x7f, 0x9e, 0xbe, 0xde, 0xfe,
		0x1d, 0x3d, 0x5d, 0x7d, 0x9c, 0xbc, 0xdc, 0xfc,
		0x1b, 0x3b, 0x5b, 0x7b, 0x9a, 0xba, 0xda, 0xfa,
		0x19, 0x39, 0x59, 0x79, 0x98, 0xb8, 0xd8, 0xf8,
		// input:
		// input ID:
		0x00, 0x00, 0x00, 0x05,
		// amount:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xe8,
		// num sig indices:
		0x00, 0x00, 0x00, 0x01,
		// sig index[0]:
		0x00, 0x00, 0x00, 0x00,
		// number of credentials:
		0x00, 0x00, 0x00, 0x00,
	}

	tx := &txs.Tx{UnsignedTx: &txs.ImportTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID: 2,
			BlockchainID: ids.ID{
				0xff, 0xff, 0xff, 0xff, 0xee, 0xee, 0xee, 0xee,
				0xdd, 0xdd, 0xdd, 0xdd, 0xcc, 0xcc, 0xcc, 0xcc,
				0xbb, 0xbb, 0xbb, 0xbb, 0xaa, 0xaa, 0xaa, 0xaa,
				0x99, 0x99, 0x99, 0x99, 0x88, 0x88, 0x88, 0x88,
			},
			Memo: []byte{0x00, 0x01, 0x02, 0x03},
		}},
		SourceChain: ids.ID{
			0x1f, 0x8f, 0x9f, 0x0f, 0x1e, 0x8e, 0x9e, 0x0e,
			0x2d, 0x7d, 0xad, 0xfd, 0x2c, 0x7c, 0xac, 0xfc,
			0x3b, 0x6b, 0xbb, 0xeb, 0x3a, 0x6a, 0xba, 0xea,
			0x49, 0x59, 0xc9, 0xd9, 0x48, 0x58, 0xc8, 0xd8,
		},
		ImportedIns: []*axc.TransferableInput{{
			UTXOID: axc.UTXOID{TxID: ids.ID{
				0x0f, 0x2f, 0x4f, 0x6f, 0x8e, 0xae, 0xce, 0xee,
				0x0d, 0x2d, 0x4d, 0x6d, 0x8c, 0xac, 0xcc, 0xec,
				0x0b, 0x2b, 0x4b, 0x6b, 0x8a, 0xaa, 0xca, 0xea,
				0x09, 0x29, 0x49, 0x69, 0x88, 0xa8, 0xc8, 0xe8,
			}},
			Asset: axc.Asset{ID: ids.ID{
				0x1f, 0x3f, 0x5f, 0x7f, 0x9e, 0xbe, 0xde, 0xfe,
				0x1d, 0x3d, 0x5d, 0x7d, 0x9c, 0xbc, 0xdc, 0xfc,
				0x1b, 0x3b, 0x5b, 0x7b, 0x9a, 0xba, 0xda, 0xfa,
				0x19, 0x39, 0x59, 0x79, 0x98, 0xb8, 0xd8, 0xf8,
			}},
			In: &secp256k1fx.TransferInput{
				Amt:   1000,
				Input: secp256k1fx.Input{SigIndices: []uint32{0}},
			},
		}},
	}}

	if err := vm.parser.InitializeTx(tx); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, tx.ID().String(), "9wdPb5rsThXYLX4WxkNeyYrNMfDE5cuWLgifSjxKiA2dCmgCZ")
	result := tx.Bytes()
	if !bytes.Equal(expected, result) {
		t.Fatalf("\nExpected: 0x%x\nResult:   0x%x", expected, result)
	}

	credBytes := []byte{
		// type id
		0x00, 0x00, 0x00, 0x09,

		// there are two signers (thus two signatures)
		0x00, 0x00, 0x00, 0x02,

		// 65 bytes
		0x8c, 0xc7, 0xdc, 0x8c, 0x11, 0xd3, 0x75, 0x9e, 0x16, 0xa5,
		0x9f, 0xd2, 0x9c, 0x64, 0xd7, 0x1f, 0x9b, 0xad, 0x1a, 0x62,
		0x33, 0x98, 0xc7, 0xaf, 0x67, 0x02, 0xc5, 0xe0, 0x75, 0x8e,
		0x62, 0xcf, 0x15, 0x6d, 0x99, 0xf5, 0x4e, 0x71, 0xb8, 0xf4,
		0x8b, 0x5b, 0xbf, 0x0c, 0x59, 0x62, 0x79, 0x34, 0x97, 0x1a,
		0x1f, 0x49, 0x9b, 0x0a, 0x4f, 0xbf, 0x95, 0xfc, 0x31, 0x39,
		0x46, 0x4e, 0xa1, 0xaf, 0x00,

		// 65 bytes
		0x8c, 0xc7, 0xdc, 0x8c, 0x11, 0xd3, 0x75, 0x9e, 0x16, 0xa5,
		0x9f, 0xd2, 0x9c, 0x64, 0xd7, 0x1f, 0x9b, 0xad, 0x1a, 0x62,
		0x33, 0x98, 0xc7, 0xaf, 0x67, 0x02, 0xc5, 0xe0, 0x75, 0x8e,
		0x62, 0xcf, 0x15, 0x6d, 0x99, 0xf5, 0x4e, 0x71, 0xb8, 0xf4,
		0x8b, 0x5b, 0xbf, 0x0c, 0x59, 0x62, 0x79, 0x34, 0x97, 0x1a,
		0x1f, 0x49, 0x9b, 0x0a, 0x4f, 0xbf, 0x95, 0xfc, 0x31, 0x39,
		0x46, 0x4e, 0xa1, 0xaf, 0x00,

		// type id
		0x00, 0x00, 0x00, 0x09,

		// there are two signers (thus two signatures)
		0x00, 0x00, 0x00, 0x02,

		// 65 bytes
		0x8c, 0xc7, 0xdc, 0x8c, 0x11, 0xd3, 0x75, 0x9e, 0x16, 0xa5,
		0x9f, 0xd2, 0x9c, 0x64, 0xd7, 0x1f, 0x9b, 0xad, 0x1a, 0x62,
		0x33, 0x98, 0xc7, 0xaf, 0x67, 0x02, 0xc5, 0xe0, 0x75, 0x8e,
		0x62, 0xcf, 0x15, 0x6d, 0x99, 0xf5, 0x4e, 0x71, 0xb8, 0xf4,
		0x8b, 0x5b, 0xbf, 0x0c, 0x59, 0x62, 0x79, 0x34, 0x97, 0x1a,
		0x1f, 0x49, 0x9b, 0x0a, 0x4f, 0xbf, 0x95, 0xfc, 0x31, 0x39,
		0x46, 0x4e, 0xa1, 0xaf, 0x00,

		// 65 bytes
		0x8c, 0xc7, 0xdc, 0x8c, 0x11, 0xd3, 0x75, 0x9e, 0x16, 0xa5,
		0x9f, 0xd2, 0x9c, 0x64, 0xd7, 0x1f, 0x9b, 0xad, 0x1a, 0x62,
		0x33, 0x98, 0xc7, 0xaf, 0x67, 0x02, 0xc5, 0xe0, 0x75, 0x8e,
		0x62, 0xcf, 0x15, 0x6d, 0x99, 0xf5, 0x4e, 0x71, 0xb8, 0xf4,
		0x8b, 0x5b, 0xbf, 0x0c, 0x59, 0x62, 0x79, 0x34, 0x97, 0x1a,
		0x1f, 0x49, 0x9b, 0x0a, 0x4f, 0xbf, 0x95, 0xfc, 0x31, 0x39,
		0x46, 0x4e, 0xa1, 0xaf, 0x00,
	}
	if err := tx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{keys[0], keys[0]}, {keys[0], keys[0]}}); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, tx.ID().String(), "pCW7sVBytzdZ1WrqzGY1DvA2S9UaMr72xpUMxVyx1QHBARNYx")
	result = tx.Bytes()

	// there are two credentials
	expected[len(expected)-1] = 0x02
	expected = append(expected, credBytes...)
	if !bytes.Equal(expected, result) {
		t.Fatalf("\nExpected: 0x%x\nResult:   0x%x", expected, result)
	}
}

// Test issuing an import transaction.
func TestIssueImportTx(t *testing.T) {
	genesisBytes := BuildGenesisTest(t)

	issuer := make(chan common.Message, 1)
	baseDBManager := manager.NewMemDB(version.DefaultVersion1_0_0)

	m := &atomic.Memory{}
	err := m.Initialize(logging.NoLog{}, prefixdb.New([]byte{0}, baseDBManager.Current().Database))
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext(t)
	ctx.SharedMemory = m.NewSharedMemory(chainID)
	peerSharedMemory := m.NewSharedMemory(platformChainID)

	genesisTx := GetAXCTxFromGenesisTest(genesisBytes, t)

	axcID := genesisTx.ID()
	platformID := ids.Empty.Prefix(0)

	ctx.Lock.Lock()

	avmConfig := Config{
		IndexTransactions: true,
	}

	avmConfigBytes, err := stdjson.Marshal(avmConfig)
	assert.NoError(t, err)
	vm := &VM{}
	err = vm.Initialize(
		ctx,
		baseDBManager.NewPrefixDBManager([]byte{1}),
		genesisBytes,
		nil,
		avmConfigBytes,
		issuer,
		[]*common.Fx{{
			ID: ids.Empty,
			Fx: &secp256k1fx.Fx{},
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	vm.batchTimeout = 0

	if err = vm.SetState(snow.Bootstrapping); err != nil {
		t.Fatal(err)
	}

	err = vm.SetState(snow.NormalOp)
	if err != nil {
		t.Fatal(err)
	}

	key := keys[0]

	utxoID := axc.UTXOID{
		TxID: ids.ID{
			0x0f, 0x2f, 0x4f, 0x6f, 0x8e, 0xae, 0xce, 0xee,
			0x0d, 0x2d, 0x4d, 0x6d, 0x8c, 0xac, 0xcc, 0xec,
			0x0b, 0x2b, 0x4b, 0x6b, 0x8a, 0xaa, 0xca, 0xea,
			0x09, 0x29, 0x49, 0x69, 0x88, 0xa8, 0xc8, 0xe8,
		},
	}

	txAssetID := axc.Asset{ID: axcID}
	tx := &txs.Tx{UnsignedTx: &txs.ImportTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Outs: []*axc.TransferableOutput{{
				Asset: txAssetID,
				Out: &secp256k1fx.TransferOutput{
					Amt: 1000,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{keys[0].PublicKey().Address()},
					},
				},
			}},
		}},
		SourceChain: platformChainID,
		ImportedIns: []*axc.TransferableInput{{
			UTXOID: utxoID,
			Asset:  txAssetID,
			In: &secp256k1fx.TransferInput{
				Amt: 1010,
				Input: secp256k1fx.Input{
					SigIndices: []uint32{0},
				},
			},
		}},
	}}
	if err := tx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{key}}); err != nil {
		t.Fatal(err)
	}

	if _, err := vm.IssueTx(tx.Bytes()); err == nil {
		t.Fatal(err)
	}

	// Provide the platform UTXO:

	utxo := &axc.UTXO{
		UTXOID: utxoID,
		Asset:  txAssetID,
		Out: &secp256k1fx.TransferOutput{
			Amt: 1010,
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{key.PublicKey().Address()},
			},
		},
	}

	utxoBytes, err := vm.parser.Codec().Marshal(txs.CodecVersion, utxo)
	if err != nil {
		t.Fatal(err)
	}

	inputID := utxo.InputID()

	if err := peerSharedMemory.Apply(map[ids.ID]*atomic.Requests{vm.ctx.ChainID: {PutRequests: []*atomic.Element{{
		Key:   inputID[:],
		Value: utxoBytes,
		Traits: [][]byte{
			key.PublicKey().Address().Bytes(),
		},
	}}}}); err != nil {
		t.Fatal(err)
	}

	if _, err := vm.IssueTx(tx.Bytes()); err != nil {
		t.Fatalf("should have issued the transaction correctly but erred: %s", err)
	}
	ctx.Lock.Unlock()

	msg := <-issuer
	if msg != common.PendingTxs {
		t.Fatalf("Wrong message")
	}

	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	txs := vm.PendingTxs()
	if len(txs) != 1 {
		t.Fatalf("Should have returned %d tx(s)", 1)
	}

	parsedTx := txs[0]
	if err := parsedTx.Verify(); err != nil {
		t.Fatal("Failed verify", err)
	}

	if err := parsedTx.Accept(); err != nil {
		t.Fatal(err)
	}

	assertIndexedTX(t, vm.db, 0, key.PublicKey().Address(), txAssetID.AssetID(), parsedTx.ID())
	assertLatestIdx(t, vm.db, key.PublicKey().Address(), axcID, 1)

	id := utxoID.InputID()
	if _, err := vm.ctx.SharedMemory.Get(platformID, [][]byte{id[:]}); err == nil {
		t.Fatalf("shouldn't have been able to read the utxo")
	}
}

// Test force accepting an import transaction.
func TestForceAcceptImportTx(t *testing.T) {
	genesisBytes := BuildGenesisTest(t)

	issuer := make(chan common.Message, 1)
	baseDBManager := manager.NewMemDB(version.DefaultVersion1_0_0)

	m := &atomic.Memory{}
	err := m.Initialize(logging.NoLog{}, prefixdb.New([]byte{0}, baseDBManager.Current().Database))
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext(t)
	ctx.SharedMemory = m.NewSharedMemory(chainID)

	platformID := ids.Empty.Prefix(0)

	vm := &VM{}
	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()
	err = vm.Initialize(
		ctx,
		baseDBManager.NewPrefixDBManager([]byte{1}),
		genesisBytes,
		nil,
		nil,
		issuer,
		[]*common.Fx{{
			ID: ids.Empty,
			Fx: &secp256k1fx.Fx{},
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	vm.batchTimeout = 0

	if err = vm.SetState(snow.Bootstrapping); err != nil {
		t.Fatal(err)
	}

	err = vm.SetState(snow.NormalOp)
	if err != nil {
		t.Fatal(err)
	}

	key := keys[0]

	genesisTx := GetAXCTxFromGenesisTest(genesisBytes, t)

	utxoID := axc.UTXOID{
		TxID: ids.ID{
			0x0f, 0x2f, 0x4f, 0x6f, 0x8e, 0xae, 0xce, 0xee,
			0x0d, 0x2d, 0x4d, 0x6d, 0x8c, 0xac, 0xcc, 0xec,
			0x0b, 0x2b, 0x4b, 0x6b, 0x8a, 0xaa, 0xca, 0xea,
			0x09, 0x29, 0x49, 0x69, 0x88, 0xa8, 0xc8, 0xe8,
		},
	}

	tx := &txs.Tx{UnsignedTx: &txs.ImportTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
		}},
		SourceChain: platformChainID,
		ImportedIns: []*axc.TransferableInput{{
			UTXOID: utxoID,
			Asset:  axc.Asset{ID: genesisTx.ID()},
			In: &secp256k1fx.TransferInput{
				Amt:   1000,
				Input: secp256k1fx.Input{SigIndices: []uint32{0}},
			},
		}},
	}}

	if err := tx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{key}}); err != nil {
		t.Fatal(err)
	}

	parsedTx, err := vm.ParseTx(tx.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if err := parsedTx.Verify(); err == nil {
		t.Fatalf("Should have failed verification")
	}

	if err := parsedTx.Accept(); err != nil {
		t.Fatal(err)
	}

	id := utxoID.InputID()
	if _, err := vm.ctx.SharedMemory.Get(platformID, [][]byte{id[:]}); err == nil {
		t.Fatalf("shouldn't have been able to read the utxo")
	}
}

func TestImportTxNotState(t *testing.T) {
	intf := interface{}(&txs.ImportTx{})
	if _, ok := intf.(verify.State); ok {
		t.Fatalf("shouldn't be marked as state")
	}
}

// Test issuing an import transaction.
func TestIssueExportTx(t *testing.T) {
	genesisBytes := BuildGenesisTest(t)

	issuer := make(chan common.Message, 1)
	baseDBManager := manager.NewMemDB(version.DefaultVersion1_0_0)

	m := &atomic.Memory{}
	err := m.Initialize(logging.NoLog{}, prefixdb.New([]byte{0}, baseDBManager.Current().Database))
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext(t)
	ctx.SharedMemory = m.NewSharedMemory(chainID)

	genesisTx := GetAXCTxFromGenesisTest(genesisBytes, t)

	axcID := genesisTx.ID()

	ctx.Lock.Lock()
	vm := &VM{}
	if err := vm.Initialize(
		ctx,
		baseDBManager.NewPrefixDBManager([]byte{1}),
		genesisBytes,
		nil,
		nil,
		issuer, []*common.Fx{{
			ID: ids.Empty,
			Fx: &secp256k1fx.Fx{},
		}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	vm.batchTimeout = 0

	if err := vm.SetState(snow.Bootstrapping); err != nil {
		t.Fatal(err)
	}

	if err := vm.SetState(snow.NormalOp); err != nil {
		t.Fatal(err)
	}

	key := keys[0]

	tx := &txs.Tx{UnsignedTx: &txs.ExportTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*axc.TransferableInput{{
				UTXOID: axc.UTXOID{
					TxID:        axcID,
					OutputIndex: 2,
				},
				Asset: axc.Asset{ID: axcID},
				In: &secp256k1fx.TransferInput{
					Amt:   startBalance,
					Input: secp256k1fx.Input{SigIndices: []uint32{0}},
				},
			}},
		}},
		DestinationChain: platformChainID,
		ExportedOuts: []*axc.TransferableOutput{{
			Asset: axc.Asset{ID: axcID},
			Out: &secp256k1fx.TransferOutput{
				Amt: startBalance - vm.TxFee,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{key.PublicKey().Address()},
				},
			},
		}},
	}}
	if err := tx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{key}}); err != nil {
		t.Fatal(err)
	}

	if _, err := vm.IssueTx(tx.Bytes()); err != nil {
		t.Fatal(err)
	}

	ctx.Lock.Unlock()

	msg := <-issuer
	if msg != common.PendingTxs {
		t.Fatalf("Wrong message")
	}

	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	txs := vm.PendingTxs()
	if len(txs) != 1 {
		t.Fatalf("Should have returned %d tx(s)", 1)
	}

	parsedTx := txs[0]
	if err := parsedTx.Verify(); err != nil {
		t.Fatal(err)
	} else if err := parsedTx.Accept(); err != nil {
		t.Fatal(err)
	}

	peerSharedMemory := m.NewSharedMemory(platformChainID)
	utxoBytes, _, _, err := peerSharedMemory.Indexed(
		vm.ctx.ChainID,
		[][]byte{
			key.PublicKey().Address().Bytes(),
		},
		nil,
		nil,
		math.MaxInt32,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(utxoBytes) != 1 {
		t.Fatalf("wrong number of utxos %d", len(utxoBytes))
	}
}

func TestClearForceAcceptedExportTx(t *testing.T) {
	genesisBytes := BuildGenesisTest(t)

	issuer := make(chan common.Message, 1)
	baseDBManager := manager.NewMemDB(version.DefaultVersion1_0_0)

	m := &atomic.Memory{}
	err := m.Initialize(logging.NoLog{}, prefixdb.New([]byte{0}, baseDBManager.Current().Database))
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext(t)
	ctx.SharedMemory = m.NewSharedMemory(chainID)

	genesisTx := GetAXCTxFromGenesisTest(genesisBytes, t)

	axcID := genesisTx.ID()
	platformID := ids.Empty.Prefix(0)

	ctx.Lock.Lock()

	avmConfig := Config{
		IndexTransactions: true,
	}
	avmConfigBytes, err := stdjson.Marshal(avmConfig)
	assert.NoError(t, err)
	vm := &VM{}
	err = vm.Initialize(
		ctx,
		baseDBManager.NewPrefixDBManager([]byte{1}),
		genesisBytes,
		nil,
		avmConfigBytes,
		issuer,
		[]*common.Fx{{
			ID: ids.Empty,
			Fx: &secp256k1fx.Fx{},
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	vm.batchTimeout = 0

	if err = vm.SetState(snow.Bootstrapping); err != nil {
		t.Fatal(err)
	}

	err = vm.SetState(snow.NormalOp)
	if err != nil {
		t.Fatal(err)
	}

	key := keys[0]

	assetID := axc.Asset{ID: axcID}
	tx := &txs.Tx{UnsignedTx: &txs.ExportTx{
		BaseTx: txs.BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*axc.TransferableInput{{
				UTXOID: axc.UTXOID{
					TxID:        axcID,
					OutputIndex: 2,
				},
				Asset: assetID,
				In: &secp256k1fx.TransferInput{
					Amt:   startBalance,
					Input: secp256k1fx.Input{SigIndices: []uint32{0}},
				},
			}},
		}},
		DestinationChain: platformChainID,
		ExportedOuts: []*axc.TransferableOutput{{
			Asset: assetID,
			Out: &secp256k1fx.TransferOutput{
				Amt: startBalance - vm.TxFee,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{key.PublicKey().Address()},
				},
			},
		}},
	}}
	if err := tx.SignSECP256K1Fx(vm.parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{key}}); err != nil {
		t.Fatal(err)
	}

	if _, err := vm.IssueTx(tx.Bytes()); err != nil {
		t.Fatal(err)
	}

	ctx.Lock.Unlock()

	msg := <-issuer
	if msg != common.PendingTxs {
		t.Fatalf("Wrong message")
	}

	ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		ctx.Lock.Unlock()
	}()

	txs := vm.PendingTxs()
	if len(txs) != 1 {
		t.Fatalf("Should have returned %d tx(s)", 1)
	}

	parsedTx := txs[0]
	if err := parsedTx.Verify(); err != nil {
		t.Fatal(err)
	}

	utxo := axc.UTXOID{
		TxID:        tx.ID(),
		OutputIndex: 0,
	}
	utxoID := utxo.InputID()

	peerSharedMemory := m.NewSharedMemory(platformID)
	if err := peerSharedMemory.Apply(map[ids.ID]*atomic.Requests{vm.ctx.ChainID: {RemoveRequests: [][]byte{utxoID[:]}}}); err != nil {
		t.Fatal(err)
	}

	if err := parsedTx.Accept(); err != nil {
		t.Fatal(err)
	}

	assertIndexedTX(t, vm.db, 0, key.PublicKey().Address(), assetID.AssetID(), parsedTx.ID())
	assertLatestIdx(t, vm.db, key.PublicKey().Address(), assetID.AssetID(), 1)

	if _, err := peerSharedMemory.Get(vm.ctx.ChainID, [][]byte{utxoID[:]}); err == nil {
		t.Fatalf("should have failed to read the utxo")
	}
}
