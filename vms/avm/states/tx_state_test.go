// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package states

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/database"
	"github.com/sankar-boro/axia-network-v2/database/memdb"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/crypto"
	"github.com/sankar-boro/axia-network-v2/utils/formatting"
	"github.com/sankar-boro/axia-network-v2/utils/units"
	"github.com/sankar-boro/axia-network-v2/vms/avm/fxs"
	"github.com/sankar-boro/axia-network-v2/vms/avm/txs"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/nftfx"
	"github.com/sankar-boro/axia-network-v2/vms/propertyfx"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
)

var (
	networkID uint32 = 10
	chainID          = ids.ID{5, 4, 3, 2, 1}
	assetID          = ids.ID{1, 2, 3}
	keys      []*crypto.PrivateKeySECP256K1R
	addrs     []ids.ShortID // addrs[i] corresponds to keys[i]
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

func TestTxState(t *testing.T) {
	assert := assert.New(t)

	db := memdb.New()
	parser, err := txs.NewParser([]fxs.Fx{
		&secp256k1fx.Fx{},
		&nftfx.Fx{},
		&propertyfx.Fx{},
	})
	assert.NoError(err)

	stateIntf, err := NewTxState(db, parser, prometheus.NewRegistry())
	assert.NoError(err)

	s := stateIntf.(*txState)

	_, err = s.GetTx(ids.Empty)
	assert.Equal(database.ErrNotFound, err)

	tx := &txs.Tx{
		UnsignedTx: &txs.BaseTx{
			BaseTx: axc.BaseTx{
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
			},
		},
	}

	err = tx.SignSECP256K1Fx(parser.Codec(), [][]*crypto.PrivateKeySECP256K1R{{keys[0]}})
	assert.NoError(err)

	err = s.PutTx(ids.Empty, tx)
	assert.NoError(err)

	loadedTx, err := s.GetTx(ids.Empty)
	assert.NoError(err)
	assert.Equal(tx.ID(), loadedTx.ID())

	s.txCache.Flush()

	loadedTx, err = s.GetTx(ids.Empty)
	assert.NoError(err)
	assert.Equal(tx.ID(), loadedTx.ID())

	err = s.DeleteTx(ids.Empty)
	assert.NoError(err)

	_, err = s.GetTx(ids.Empty)
	assert.Equal(database.ErrNotFound, err)

	s.txCache.Flush()

	_, err = s.GetTx(ids.Empty)
	assert.Equal(database.ErrNotFound, err)
}
