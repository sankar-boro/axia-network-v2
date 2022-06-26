// Copyright (C) 2019-2022, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Implements transfer tests.
package transfer

import (
	"context"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/sankar-boro/axia-network-v2/genesis"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow/choices"
	"github.com/sankar-boro/axia-network-v2/tests"
	"github.com/sankar-boro/axia-network-v2/tests/e2e"
	"github.com/sankar-boro/axia-network-v2/utils/crypto"
	"github.com/sankar-boro/axia-network-v2/vms/avm"
	"github.com/sankar-boro/axia-network-v2/vms/components/axc"
	"github.com/sankar-boro/axia-network-v2/vms/secp256k1fx"
	"github.com/sankar-boro/axia-network-v2/wallet/subnet/primary"
	"github.com/sankar-boro/axia-network-v2/wallet/subnet/primary/common"
)

var keyFactory crypto.FactorySECP256K1R

var _ = e2e.DescribeSwapChain("[Virtuous Transfer Tx AXC]", func() {
	ginkgo.It("can issue a virtuous transfer tx for AXC asset", func() {
		if e2e.GetEnableWhitelistTxTests() {
			ginkgo.Skip("whitelist vtx tests are enabled; skipping")
		}

		uris := e2e.GetURIs()
		gomega.Expect(uris).ShouldNot(gomega.BeEmpty())

		// TODO: take pre-funded keys as arguments
		ewoqAddr := genesis.EWOQKey.PublicKey().Address()

		randomKeyIntf, err := keyFactory.NewPrivateKey()
		gomega.Expect(err).Should(gomega.BeNil())

		randomKey := randomKeyIntf.(*crypto.PrivateKeySECP256K1R)
		randomAddr := randomKey.PublicKey().Address()
		keys := secp256k1fx.NewKeychain(
			genesis.EWOQKey,
			randomKey,
		)
		var baseWallet primary.Wallet
		ginkgo.By("setting up a base wallet", func() {
			walletURI := uris[0]

			// 5-second is enough to fetch initial UTXOs for test cluster in "primary.NewWallet"
			ctx, cancel := context.WithTimeout(context.Background(), e2e.DefaultWalletCreationTimeout)
			var err error
			baseWallet, err = primary.NewWalletFromURI(ctx, walletURI, keys)
			cancel()
			gomega.Expect(err).Should(gomega.BeNil())
		})

		allMetrics := []string{
			"axia_X_vtx_processing",
			"axia_X_vtx_accepted_count",
			"axia_X_vtx_rejected_count",
		}

		// URI -> "metric name" -> "metric value"
		curMetrics := make(map[string]map[string]float64)
		ginkgo.By("collect swap-chain metrics", func() {
			for _, u := range uris {
				ep := u + "/ext/metrics"

				mm, err := tests.GetMetricsValue(ep, allMetrics...)
				gomega.Expect(err).Should(gomega.BeNil())
				tests.Outf("{{green}}metrics at %q:{{/}} %v\n", ep, mm)

				if mm["axia_X_vtx_processing"] > 0 {
					tests.Outf("{{red}}{{bold}}%q already has processing vtx!!!{{/}}\n", u)
					ginkgo.Skip("the cluster has already ongoing vtx txs thus skipping to prevent conflicts...")
				}

				curMetrics[u] = mm
			}
		})

		ewoqWallet := primary.NewWalletWithOptions(
			baseWallet,
			common.WithCustomAddresses(ids.ShortSet{
				ewoqAddr: struct{}{},
			}),
		)
		randWallet := primary.NewWalletWithOptions(
			baseWallet,
			common.WithCustomAddresses(ids.ShortSet{
				randomAddr: struct{}{},
			}),
		)
		var txID ids.ID
		ginkgo.By("issue regular, virtuous Swap-Chain tx should succeed", func() {
			balances, err := ewoqWallet.X().Builder().GetFTBalance()
			gomega.Expect(err).Should(gomega.BeNil())

			axcAssetID := baseWallet.X().AXCAssetID()
			ewoqPrevBalX := balances[axcAssetID]
			tests.Outf("{{green}}ewoq wallet balance:{{/}} %d\n", ewoqPrevBalX)

			balances, err = randWallet.X().Builder().GetFTBalance()
			gomega.Expect(err).Should(gomega.BeNil())

			randPrevBalX := balances[axcAssetID]
			tests.Outf("{{green}}rand wallet balance:{{/}} %d\n", randPrevBalX)

			amount := ewoqPrevBalX / 10
			if amount == 0 {
				ginkgo.Skip("not enough balance in the test wallet")
			}
			tests.Outf("{{green}}amount to transfer:{{/}} %d\n", amount)

			// transfer "amount" from "ewoq" to "random"
			tests.Outf("{{blue}}transferring %d from 'ewoq' to 'random' at %q{{/}}\n", amount, uris[0])
			ctx, cancel := context.WithTimeout(context.Background(), e2e.DefaultConfirmTxTimeout)
			txID, err = ewoqWallet.X().IssueBaseTx(
				[]*axc.TransferableOutput{{
					Asset: axc.Asset{
						ID: axcAssetID,
					},
					Out: &secp256k1fx.TransferOutput{
						Amt: amount,
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{randomAddr},
						},
					},
				}},
				common.WithContext(ctx),
			)
			cancel()
			gomega.Expect(err).Should(gomega.BeNil())

			balances, err = ewoqWallet.X().Builder().GetFTBalance()
			gomega.Expect(err).Should(gomega.BeNil())
			ewoqCurBalX := balances[axcAssetID]
			tests.Outf("{{green}}ewoq wallet balance:{{/}} %d\n", ewoqCurBalX)

			balances, err = randWallet.X().Builder().GetFTBalance()
			gomega.Expect(err).Should(gomega.BeNil())
			randCurBalX := balances[axcAssetID]
			tests.Outf("{{green}}ewoq wallet balance:{{/}} %d\n", randCurBalX)

			gomega.Expect(ewoqCurBalX).Should(gomega.Equal(ewoqPrevBalX - amount - baseWallet.X().BaseTxFee()))
			gomega.Expect(randCurBalX).Should(gomega.Equal(randPrevBalX + amount))
		})

		ginkgo.By("accept swap-chain tx in all nodes", func() {
			tests.Outf("{{blue}}waiting before querying metrics{{/}}\n")

			for _, u := range uris {
				xc := avm.NewClient(u, "Swap")
				ctx, cancel := context.WithTimeout(context.Background(), e2e.DefaultConfirmTxTimeout)
				status, err := xc.ConfirmTx(ctx, txID, 2*time.Second)
				cancel()
				gomega.Expect(err).Should(gomega.BeNil())
				gomega.Expect(status).Should(gomega.Equal(choices.Accepted))

				ep := u + "/ext/metrics"
				mm, err := tests.GetMetricsValue(ep, allMetrics...)
				gomega.Expect(err).Should(gomega.BeNil())

				prev := curMetrics[u]

				// +0 since swap-chain tx must have been processed and accepted by now
				gomega.Expect(mm["axia_X_vtx_processing"]).Should(gomega.Equal(prev["axia_X_vtx_processing"]))

				// +1 since swap-chain tx must have been accepted by now
				gomega.Expect(mm["axia_X_vtx_accepted_count"]).Should(gomega.Equal(prev["axia_X_vtx_accepted_count"] + 1))

				// +0 since virtuous swap-chain tx must not be rejected
				gomega.Expect(mm["axia_X_vtx_rejected_count"]).Should(gomega.Equal(prev["axia_X_vtx_rejected_count"]))

				curMetrics[u] = mm
			}
		})
	})
})
