// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/vms/avm"
	"github.com/sankar-boro/axia-network-v2/vms/platformvm"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
	"github.com/sankar-boro/axia-network-v2/axiawallet/chain/core"
	"github.com/sankar-boro/axia-network-v2/axiawallet/chain/swap"
	"github.com/sankar-boro/axia-network-v2/axiawallet/allychain/primary/common"
)

var _ AxiaWallet = &axiawallet{}

// AxiaWallet provides chain axiawallets for the primary network.
type AxiaWallet interface {
	Core() core.AxiaWallet
	Swap() swap.AxiaWallet
}

type axiawallet struct {
	core core.AxiaWallet
	swap swap.AxiaWallet
}

func (w *axiawallet) Core() core.AxiaWallet { return w.core }
func (w *axiawallet) Swap() swap.AxiaWallet { return w.swap }

// NewAxiaWalletFromURI returns a axiawallet that supports issuing transactions to the
// chains living in the primary network to a provided [uri].
//
// On creation, the axiawallet attaches to the provided [uri] and fetches all UTXOs
// that reference any of the keys contained in [kc]. If the UTXOs are modified
// through an external issuance process, such as another instance of the axiawallet,
// the UTXOs may become out of sync.
//
// The axiawallet manages all UTXOs locally, and performs all tx signing locally.
func NewAxiaWalletFromURI(ctx context.Context, uri string, kc *secp256k1fx.Keychain) (AxiaWallet, error) {
	pCTX, xCTX, utxos, err := FetchState(ctx, uri, kc.Addrs)
	if err != nil {
		return nil, err
	}
	return NewAxiaWalletWithState(uri, pCTX, xCTX, utxos, kc), nil
}

func NewAxiaWalletWithState(
	uri string,
	pCTX core.Context,
	xCTX swap.Context,
	utxos UTXOs,
	kc *secp256k1fx.Keychain,
) AxiaWallet {
	pUTXOs := NewChainUTXOs(constants.PlatformChainID, utxos)
	pTXs := make(map[ids.ID]*platformvm.Tx)
	pBackend := core.NewBackend(pCTX, pUTXOs, pTXs)
	pBuilder := core.NewBuilder(kc.Addrs, pBackend)
	pSigner := core.NewSigner(kc, pBackend)
	pClient := platformvm.NewClient(uri)

	swapChainID := xCTX.BlockchainID()
	xUTXOs := NewChainUTXOs(swapChainID, utxos)
	xBackend := swap.NewBackend(xCTX, swapChainID, xUTXOs)
	xBuilder := swap.NewBuilder(kc.Addrs, xBackend)
	xSigner := swap.NewSigner(kc, xBackend)
	xClient := avm.NewClient(uri, "Swap")

	return NewAxiaWallet(
		core.NewAxiaWallet(pBuilder, pSigner, pClient, pBackend),
		swap.NewAxiaWallet(xBuilder, xSigner, xClient, xBackend),
	)
}

func NewAxiaWalletWithOptions(w AxiaWallet, options ...common.Option) AxiaWallet {
	return NewAxiaWallet(
		p.NewAxiaWalletWithOptions(w.Core(), options...),
		x.NewAxiaWalletWithOptions(w.Swap(), options...),
	)
}

func NewAxiaWallet(core core.AxiaWallet, swap swap.AxiaWallet) AxiaWallet {
	return &axiawallet{
		core: core,
		swap: swap,
	}
}
