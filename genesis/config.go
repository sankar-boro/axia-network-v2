// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/utils/formatting/address"
	"github.com/sankar-boro/axia-network-v2/utils/wrappers"

	safemath "github.com/sankar-boro/axia-network-v2/utils/math"
)

type LockedAmount struct {
	Amount   uint64 `json:"amount"`
	Locktime uint64 `json:"locktime"`
}

type Allocation struct {
	ETHAddr        ids.ShortID    `json:"ethAddr"`
	AXCAddr       ids.ShortID    `json:"axcAddr"`
	InitialAmount  uint64         `json:"initialAmount"`
	UnlockSchedule []LockedAmount `json:"unlockSchedule"`
}

func (a Allocation) Unparse(networkID uint32) (UnparsedAllocation, error) {
	ua := UnparsedAllocation{
		InitialAmount:  a.InitialAmount,
		UnlockSchedule: a.UnlockSchedule,
		ETHAddr:        "0x" + hex.EncodeToString(a.ETHAddr.Bytes()),
	}
	axcAddr, err := address.Format(
		"Swap",
		constants.GetHRP(networkID),
		a.AXCAddr.Bytes(),
	)
	ua.AXCAddr = axcAddr
	return ua, err
}

type Staker struct {
	NodeID        ids.NodeID  `json:"nodeID"`
	RewardAddress ids.ShortID `json:"rewardAddress"`
	DelegationFee uint32      `json:"delegationFee"`
}

func (s Staker) Unparse(networkID uint32) (UnparsedStaker, error) {
	axcAddr, err := address.Format(
		"Swap",
		constants.GetHRP(networkID),
		s.RewardAddress.Bytes(),
	)
	return UnparsedStaker{
		NodeID:        s.NodeID,
		RewardAddress: axcAddr,
		DelegationFee: s.DelegationFee,
	}, err
}

// Config contains the genesis addresses used to construct a genesis
type Config struct {
	NetworkID uint32 `json:"networkID"`

	Allocations []Allocation `json:"allocations"`

	StartTime                  uint64        `json:"startTime"`
	InitialStakeDuration       uint64        `json:"initialStakeDuration"`
	InitialStakeDurationOffset uint64        `json:"initialStakeDurationOffset"`
	InitialStakedFunds         []ids.ShortID `json:"initialStakedFunds"`
	InitialStakers             []Staker      `json:"initialStakers"`

	AXCChainGenesis string `json:"axcChainGenesis"`

	Message string `json:"message"`
}

func (c Config) Unparse() (UnparsedConfig, error) {
	uc := UnparsedConfig{
		NetworkID:                  c.NetworkID,
		Allocations:                make([]UnparsedAllocation, len(c.Allocations)),
		StartTime:                  c.StartTime,
		InitialStakeDuration:       c.InitialStakeDuration,
		InitialStakeDurationOffset: c.InitialStakeDurationOffset,
		InitialStakedFunds:         make([]string, len(c.InitialStakedFunds)),
		InitialStakers:             make([]UnparsedStaker, len(c.InitialStakers)),
		AXCChainGenesis:              c.AXCChainGenesis,
		Message:                    c.Message,
	}
	for i, a := range c.Allocations {
		ua, err := a.Unparse(uc.NetworkID)
		if err != nil {
			return uc, err
		}
		uc.Allocations[i] = ua
	}
	for i, isa := range c.InitialStakedFunds {
		axcAddr, err := address.Format(
			"Swap",
			constants.GetHRP(uc.NetworkID),
			isa.Bytes(),
		)
		if err != nil {
			return uc, err
		}
		uc.InitialStakedFunds[i] = axcAddr
	}
	for i, is := range c.InitialStakers {
		uis, err := is.Unparse(c.NetworkID)
		if err != nil {
			return uc, err
		}
		uc.InitialStakers[i] = uis
	}

	return uc, nil
}

func (c *Config) InitialSupply() (uint64, error) {
	initialSupply := uint64(0)
	for _, allocation := range c.Allocations {
		newInitialSupply, err := safemath.Add64(initialSupply, allocation.InitialAmount)
		if err != nil {
			return 0, err
		}
		for _, unlock := range allocation.UnlockSchedule {
			newInitialSupply, err = safemath.Add64(newInitialSupply, unlock.Amount)
			if err != nil {
				return 0, err
			}
		}
		initialSupply = newInitialSupply
	}
	return initialSupply, nil
}

var (
	// MainnetConfig is the config that should be used to generate the mainnet
	// genesis.
	MainnetConfig Config

	// TestConfig is the config that should be used to generate the test
	// genesis.
	TestConfig Config

	// LocalConfig is the config that should be used to generate a local
	// genesis.
	LocalConfig Config
)

var (
	testGenesisConfigJSON = `{
		"networkID": 5,
		"allocations": [
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"initialAmount": 0,
				"unlockSchedule": [
					{
						"amount": 40000000000000000
					}
				]
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test1xpmx0ljrpvqexrvrj26fnggvr0ax9wm32gaxmx",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test1wrv92qg5x3dsqrtukdc8qxnpqust3qdakxgm4s",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test1qrmj7u9pquyy3mahzxeq0nnlnj2aceedjfqqrq",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test1cap3ru2ghc3jtdnuyey738ru8u5ekdadcvrtyk",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test18g2m7483k6swe46cpfmq96t09sp63pgv7judr4",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test1zwe0kxhg73x3ehgtkkz24k9czlfgztc45hgrg3",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test1fqcs4m9p8gdp7gckk30n8u68d55jk0hdumx30f",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test18lany6fjlzxc7vuqfd9x4k9wqp0yhk074p283d",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test18lany6fjlzxc7vuqfd9x4k9wqp0yhk074p283d",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			},
			{
				"ethAddr": "0xb3d82b1367d362de99ab59a658165aff520cbd4d",
				"axcAddr": "Swap-test10d2fqjfl3ghl73z2ez65ufanxwwhccxugq8z2t",
				"initialAmount": 32000000000000000,
				"unlockSchedule": []
			}
		],
		"startTime": 1599696000,
		"initialStakeDuration": 31536000,
		"initialStakeDurationOffset": 54000,
		"initialStakedFunds": [
			"Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw"
		],
		"initialStakers": [
			{
				"nodeID": "NodeID-NpagUxt6KQiwPch9Sd4osv8kD1TZnkjdk",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 1000000
			},
			{
				"nodeID": "NodeID-2m38qc95mhHXtrhjyGbe7r2NhniqHHJRB",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 500000
			},
			{
				"nodeID": "NodeID-LQwRLm4cbJ7T2kxcxp4uXCU5XD8DFrE1C",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 250000
			},
			{
				"nodeID": "NodeID-hArafGhY2HFTbwaaVh1CSCUCUCiJ2Vfb",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 125000
			},
			{
				"nodeID": "NodeID-4QBwET5o8kUhvt9xArhir4d3R25CtmZho",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 62500
			},
			{
				"nodeID": "NodeID-HGZ8ae74J3odT8ESreAdCtdnvWG1J4X5n",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 31250
			},
			{
				"nodeID": "NodeID-4KXitMCoE9p2BHA6VzXtaTxLoEjNDo2Pt",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-JyE4P8f4cTryNV8DCz2M81bMtGhFFHexG",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-EzGaipqomyK9UKx9DBHV6Ky3y68hoknrF",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-CYKruAjwH1BmV3m37sXNuprbr7dGQuJwG",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-LegbVf6qaMKcsXPnLStkdc1JVktmmiDxy",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-FesGqwKq7z5nPFHa5iwZctHE5EZV9Lpdq",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-BFa1padLXBj7VHa2JYvYGzcTBPQGjPhUy",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-4B4rc5vdD1758JSBYL1xyvE5NHGzz6xzH",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-EDESh4DfZFC15i613pMtWniQ9arbBZRnL",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-CZmZ9xpCzkWqjAyS7L4htzh5Lg6kf1k18",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-CTtkcXvVdhpNp6f97LEUXPwsRD3A2ZHqP",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-84KbQHSDnojroCVY7vQ7u9Tx7pUonPaS",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-JjvzhxnLHLUQ5HjVRkvG827ivbLXPwA9u",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			},
			{
				"nodeID": "NodeID-4CWTbdvgXHY1CLXqQNAp22nJDo5nAmts6",
				"rewardAddress": "Swap-test1wycv8n7d2fg9aq6unp23pnj4q0arv03ysya8jw",
				"delegationFee": 20000
			}
		],
		"axcChainGenesis": "{\"config\":{\"chainId\":43113,\"homesteadBlock\":0,\"daoForkBlock\":0,\"daoForkSupport\":true,\"eip150Block\":0,\"eip150Hash\":\"0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0\",\"eip155Block\":0,\"eip158Block\":0,\"byzantiumBlock\":0,\"constantinopleBlock\":0,\"petersburgBlock\":0,\"istanbulBlock\":0,\"muirGlacierBlock\":0},\"nonce\":\"0x0\",\"timestamp\":\"0x0\",\"extraData\":\"0x00\",\"gasLimit\":\"0x5f5e100\",\"difficulty\":\"0x0\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"coinbase\":\"0x0000000000000000000000000000000000000000\",\"alloc\":{\"0100000000000000000000000000000000000000\":{\"code\":\"0x7300000000000000000000000000000000000000003014608060405260043610603d5760003560e01c80631e010439146042578063b6510bb314606e575b600080fd5b605c60048036036020811015605657600080fd5b503560b1565b60408051918252519081900360200190f35b818015607957600080fd5b5060af60048036036080811015608e57600080fd5b506001600160a01b03813516906020810135906040810135906060013560b6565b005b30cd90565b836001600160a01b031681836108fc8690811502906040516000604051808303818888878c8acf9550505050505015801560f4573d6000803e3d6000fd5b505050505056fea26469706673582212201eebce970fe3f5cb96bf8ac6ba5f5c133fc2908ae3dcd51082cfee8f583429d064736f6c634300060a0033\",\"balance\":\"0x0\"}},\"number\":\"0x0\",\"gasUsed\":\"0x0\",\"parentHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\"}",
		"message": "hi mom"
	}`
)

func init() {
	unparsedMainnetConfig := UnparsedConfig{}
	unparsedTestConfig := UnparsedConfig{}
	unparsedLocalConfig := UnparsedConfig{}

	errs := wrappers.Errs{}
	errs.Add(
		json.Unmarshal([]byte(mainnetGenesisConfigJSON), &unparsedMainnetConfig),
		json.Unmarshal([]byte(testGenesisConfigJSON), &unparsedTestConfig),
		json.Unmarshal([]byte(localGenesisConfigJSON), &unparsedLocalConfig),
	)
	if errs.Errored() {
		panic(errs.Err)
	}

	mainnetConfig, err := unparsedMainnetConfig.Parse()
	errs.Add(err)
	MainnetConfig = mainnetConfig

	testConfig, err := unparsedTestConfig.Parse()
	errs.Add(err)
	TestConfig = testConfig

	localConfig, err := unparsedLocalConfig.Parse()
	errs.Add(err)
	LocalConfig = localConfig

	if errs.Errored() {
		panic(errs.Err)
	}
}

func GetConfig(networkID uint32) *Config {
	switch networkID {
	case constants.MainnetID:
		return &MainnetConfig
	case constants.TestID:
		return &TestConfig
	case constants.LocalID:
		return &LocalConfig
	default:
		tempConfig := LocalConfig
		tempConfig.NetworkID = networkID
		return &tempConfig
	}
}

// GetConfigFile loads a *Config from a provided filepath.
func GetConfigFile(fp string) (*Config, error) {
	bytes, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, fmt.Errorf("unable to load file %s: %w", fp, err)
	}
	return parseGenesisJSONBytesToConfig(bytes)
}

// GetConfigContent loads a *Config from a provided environment variable
func GetConfigContent(genesisContent string) (*Config, error) {
	bytes, err := base64.StdEncoding.DecodeString(genesisContent)
	if err != nil {
		return nil, fmt.Errorf("unable to decode base64 content: %w", err)
	}
	return parseGenesisJSONBytesToConfig(bytes)
}

func parseGenesisJSONBytesToConfig(bytes []byte) (*Config, error) {
	var unparsedConfig UnparsedConfig
	if err := json.Unmarshal(bytes, &unparsedConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON: %w", err)
	}

	config, err := unparsedConfig.Parse()
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %w", err)
	}
	return &config, nil
}
