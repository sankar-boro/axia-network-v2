// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"errors"
	"testing"

	"github.com/sankar-boro/avalanchego/codec"
	"github.com/sankar-boro/avalanchego/codec/linearcodec"
	"github.com/sankar-boro/avalanchego/ids"
	"github.com/sankar-boro/avalanchego/snow"
	"github.com/sankar-boro/avalanchego/utils/crypto"
	"github.com/sankar-boro/avalanchego/utils/formatting"
	"github.com/sankar-boro/avalanchego/utils/units"
	"github.com/sankar-boro/avalanchego/utils/wrappers"
	"github.com/sankar-boro/avalanchego/vms/avm/fxs"
	"github.com/sankar-boro/avalanchego/vms/components/axc"
	"github.com/sankar-boro/avalanchego/vms/secp256k1fx"
)

var (
	networkID       uint32 = 10
	chainID                = ids.ID{5, 4, 3, 2, 1}
	platformChainID        = ids.Empty.Prefix(0)

	keys  []*crypto.PrivateKeySECP256K1R
	addrs []ids.ShortID // addrs[i] corresponds to keys[i]

	assetID = ids.ID{1, 2, 3}
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

func setupCodec() codec.Manager {
	c := linearcodec.NewDefault()
	m := codec.NewDefaultManager()
	errs := wrappers.Errs{}
	errs.Add(
		c.RegisterType(&BaseTx{}),
		c.RegisterType(&CreateAssetTx{}),
		c.RegisterType(&OperationTx{}),
		c.RegisterType(&ImportTx{}),
		c.RegisterType(&ExportTx{}),
		c.RegisterType(&secp256k1fx.TransferInput{}),
		c.RegisterType(&secp256k1fx.MintOutput{}),
		c.RegisterType(&secp256k1fx.TransferOutput{}),
		c.RegisterType(&secp256k1fx.MintOperation{}),
		c.RegisterType(&secp256k1fx.Credential{}),
		m.RegisterCodec(CodecVersion, c),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
	return m
}

func NewContext(tb testing.TB) *snow.Context {
	ctx := snow.DefaultContextTest()
	ctx.NetworkID = networkID
	ctx.ChainID = chainID
	axcAssetID, err := ids.FromString("2XGxUr7VF7j1iwUp2aiGe4b6Ue2yyNghNS1SuNTNmZ77dPpXFZ")
	if err != nil {
		tb.Fatal(err)
	}
	ctx.AXCAssetID = axcAssetID
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
	return ctx
}

func TestTxNil(t *testing.T) {
	ctx := NewContext(t)
	c := linearcodec.NewDefault()
	m := codec.NewDefaultManager()
	if err := m.RegisterCodec(CodecVersion, c); err != nil {
		t.Fatal(err)
	}

	tx := (*Tx)(nil)
	if err := tx.SyntacticVerify(ctx, m, ids.Empty, 0, 0, 1); err == nil {
		t.Fatalf("Should have erred due to nil tx")
	}
}

func TestTxEmpty(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()
	tx := &Tx{}
	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 0, 1); err == nil {
		t.Fatalf("Should have erred due to nil tx")
	}
}

func TestTxInvalidCredential(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()

	tx := &Tx{
		UnsignedTx: &BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*axc.TransferableInput{{
				UTXOID: axc.UTXOID{
					TxID:        ids.Empty,
					OutputIndex: 0,
				},
				Asset: axc.Asset{ID: assetID},
				In: &secp256k1fx.TransferInput{
					Amt: 20 * units.KiloAxc,
					Input: secp256k1fx.Input{
						SigIndices: []uint32{
							0,
						},
					},
				},
			}},
		}},
		Creds: []*fxs.FxCredential{{Verifiable: &axc.TestVerifiable{Err: errors.New("")}}},
	}
	tx.Initialize(nil, nil)

	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 0, 1); err == nil {
		t.Fatalf("Tx should have failed due to an invalid credential")
	}
}

func TestTxInvalidUnsignedTx(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()

	tx := &Tx{
		UnsignedTx: &BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*axc.TransferableInput{
				{
					UTXOID: axc.UTXOID{
						TxID:        ids.Empty,
						OutputIndex: 0,
					},
					Asset: axc.Asset{ID: assetID},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloAxc,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
				{
					UTXOID: axc.UTXOID{
						TxID:        ids.Empty,
						OutputIndex: 0,
					},
					Asset: axc.Asset{ID: assetID},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloAxc,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
			},
		}},
		Creds: []*fxs.FxCredential{
			{Verifiable: &axc.TestVerifiable{}},
			{Verifiable: &axc.TestVerifiable{}},
		},
	}
	tx.Initialize(nil, nil)

	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 0, 1); err == nil {
		t.Fatalf("Tx should have failed due to an invalid unsigned tx")
	}
}

func TestTxInvalidNumberOfCredentials(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()

	tx := &Tx{
		UnsignedTx: &BaseTx{BaseTx: axc.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*axc.TransferableInput{
				{
					UTXOID: axc.UTXOID{TxID: ids.Empty, OutputIndex: 0},
					Asset:  axc.Asset{ID: assetID},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloAxc,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
				{
					UTXOID: axc.UTXOID{TxID: ids.Empty, OutputIndex: 1},
					Asset:  axc.Asset{ID: assetID},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloAxc,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
			},
		}},
		Creds: []*fxs.FxCredential{{Verifiable: &axc.TestVerifiable{}}},
	}
	tx.Initialize(nil, nil)

	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 0, 1); err == nil {
		t.Fatalf("Tx should have failed due to an invalid number of credentials")
	}
}
