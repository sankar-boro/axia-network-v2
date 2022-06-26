// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"path"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/utils/constants"
	"github.com/sankar-boro/axia/vms/nftfx"
	"github.com/sankar-boro/axia/vms/platformvm"
	"github.com/sankar-boro/axia/vms/propertyfx"
	"github.com/sankar-boro/axia/vms/secp256k1fx"
)

// Aliases returns the default aliases based on the network ID
func Aliases(genesisBytes []byte) (map[string][]string, map[ids.ID][]string, error) {
	apiAliases := map[string][]string{
		path.Join(constants.ChainAliasPrefix, constants.PlatformChainID.String()): {
			"Core",
			"core",
			path.Join(constants.ChainAliasPrefix, "Core"),
			path.Join(constants.ChainAliasPrefix, "core"),
		},
	}
	chainAliases := map[ids.ID][]string{
		constants.PlatformChainID: {"Core", "core"},
	}
	genesis := &platformvm.Genesis{} // TODO let's not re-create genesis to do aliasing
	if _, err := platformvm.GenesisCodec.Unmarshal(genesisBytes, genesis); err != nil {
		return nil, nil, err
	}
	if err := genesis.Initialize(); err != nil {
		return nil, nil, err
	}

	for _, chain := range genesis.Chains {
		uChain := chain.UnsignedTx.(*platformvm.UnsignedCreateChainTx)
		chainID := chain.ID()
		endpoint := path.Join(constants.ChainAliasPrefix, chainID.String())
		switch uChain.VMID {
		case constants.AVMID:
			apiAliases[endpoint] = []string{
				"Swap",
				"avm",
				path.Join(constants.ChainAliasPrefix, "Swap"),
				path.Join(constants.ChainAliasPrefix, "avm"),
			}
			chainAliases[chainID] = GetSwapChainAliases()
		case constants.EVMID:
			apiAliases[endpoint] = []string{
				"AXC",
				"evm",
				path.Join(constants.ChainAliasPrefix, "AXC"),
				path.Join(constants.ChainAliasPrefix, "evm"),
			}
			chainAliases[chainID] = GetAXCChainAliases()
		}
	}
	return apiAliases, chainAliases, nil
}

func GetAXCChainAliases() []string {
	return []string{"AXC", "evm"}
}

func GetSwapChainAliases() []string {
	return []string{"Swap", "avm"}
}

func GetVMAliases() map[ids.ID][]string {
	return map[ids.ID][]string{
		constants.PlatformVMID: {"core"},
		constants.AVMID:        {"avm"},
		constants.EVMID:        {"evm"},
		secp256k1fx.ID:         {"secp256k1fx"},
		nftfx.ID:               {"nftfx"},
		propertyfx.ID:          {"propertyfx"},
	}
}
