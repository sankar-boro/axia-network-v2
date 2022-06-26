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

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/utils/constants"
	"github.com/sankar-boro/axia/utils/formatting/address"
	"github.com/sankar-boro/axia/utils/wrappers"

	safemath "github.com/sankar-boro/axia/utils/math"
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
