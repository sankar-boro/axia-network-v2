// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"
	"fmt"
	"time"

	"github.com/sankar-boro/axia-network-v2/genesis"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/utils/units"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
)

func ExampleAxiaWallet() {
	ctx := context.Background()
	kc := secp256k1fx.NewKeychain(genesis.EWOQKey)

	// NewAxiaWalletFromURI fetches the available UTXOs owned by [kc] on the network
	// that [LocalAPIURI] is hosting.
	axiawalletSyncStartTime := time.Now()
	axiawallet, err := NewAxiaWalletFromURI(ctx, LocalAPIURI, kc)
	if err != nil {
		fmt.Printf("failed to initialize axiawallet with: %s\n", err)
		return
	}
	fmt.Printf("synced axiawallet in %s\n", time.Since(axiawalletSyncStartTime))

	// Get the Core-chain and the Swap-chain axiawallets
	pAxiaWallet := axiawallet.P()
	xAxiaWallet := axiawallet.X()

	// Pull out useful constants to use when issuing transactions.
	swapChainID := xAxiaWallet.BlockchainID()
	axcAssetID := xAxiaWallet.AXCAssetID()
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			genesis.EWOQKey.PublicKey().Address(),
		},
	}

	// Send 100 schmeckles to the Core-chain.
	exportStartTime := time.Now()
	exportTxID, err := xAxiaWallet.IssueExportTx(
		constants.PlatformChainID,
		[]*axc.TransferableOutput{
			{
				Asset: axc.Asset{
					ID: axcAssetID,
				},
				Out: &secp256k1fx.TransferOutput{
					Amt:          100 * units.Schmeckle,
					OutputOwners: *owner,
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("failed to issue X->P export transaction with: %s\n", err)
		return
	}
	fmt.Printf("issued X->P export %s in %s\n", exportTxID, time.Since(exportStartTime))

	// Import the 100 schmeckles from the Swap-chain into the Core-chain.
	importStartTime := time.Now()
	importTxID, err := pAxiaWallet.IssueImportTx(swapChainID, owner)
	if err != nil {
		fmt.Printf("failed to issue X->P import transaction with: %s\n", err)
		return
	}
	fmt.Printf("issued X->P import %s in %s\n", importTxID, time.Since(importStartTime))
}
