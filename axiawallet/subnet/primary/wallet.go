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
	"github.com/sankar-boro/axia-network-v2/axiawallet/chain/p"
	"github.com/sankar-boro/axia-network-v2/axiawallet/chain/x"
	"github.com/sankar-boro/axia-network-v2/axiawallet/subnet/primary/common"
)

var _ AxiaWallet = &axiawallet{}

// AxiaWallet provides chain axiawallets for the primary network.
type AxiaWallet interface {
	P() p.AxiaWallet
	X() x.AxiaWallet
}

type axiawallet struct {
	p p.AxiaWallet
	x x.AxiaWallet
}

func (w *axiawallet) P() p.AxiaWallet { return w.p }
func (w *axiawallet) X() x.AxiaWallet { return w.x }

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
	pCTX p.Context,
	xCTX x.Context,
	utxos UTXOs,
	kc *secp256k1fx.Keychain,
) AxiaWallet {
	pUTXOs := NewChainUTXOs(constants.PlatformChainID, utxos)
	pTXs := make(map[ids.ID]*platformvm.Tx)
	pBackend := p.NewBackend(pCTX, pUTXOs, pTXs)
	pBuilder := p.NewBuilder(kc.Addrs, pBackend)
	pSigner := p.NewSigner(kc, pBackend)
	pClient := platformvm.NewClient(uri)

	swapChainID := xCTX.BlockchainID()
	xUTXOs := NewChainUTXOs(swapChainID, utxos)
	xBackend := x.NewBackend(xCTX, swapChainID, xUTXOs)
	xBuilder := x.NewBuilder(kc.Addrs, xBackend)
	xSigner := x.NewSigner(kc, xBackend)
	xClient := avm.NewClient(uri, "Swap")

	return NewAxiaWallet(
		p.NewAxiaWallet(pBuilder, pSigner, pClient, pBackend),
		x.NewAxiaWallet(xBuilder, xSigner, xClient, xBackend),
	)
}

func NewAxiaWalletWithOptions(w AxiaWallet, options ...common.Option) AxiaWallet {
	return NewAxiaWallet(
		p.NewAxiaWalletWithOptions(w.P(), options...),
		x.NewAxiaWalletWithOptions(w.X(), options...),
	)
}

func NewAxiaWallet(p p.AxiaWallet, x x.AxiaWallet) AxiaWallet {
	return &axiawallet{
		p: p,
		x: x,
	}
}
