// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"encoding/json"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/utils/formatting/address"
)

// ClientStaker is the representation of a staker sent via client.
type ClientStaker struct {
	// the txID of the transaction that added this staker.
	TxID ids.ID
	// the Unix time when they start staking
	StartTime uint64
	// the Unix time when they are done staking
	EndTime uint64
	// the validator weight when sampling validators
	Weight *uint64
	// the amount of tokens being staked.
	StakeAmount *uint64
	// the node ID of the staker
	NodeID ids.NodeID
}

// ClientOwner is the repr. of a reward owner sent over client
type ClientOwner struct {
	Locktime  uint64
	Threshold uint32
	Addresses []ids.ShortID
}

// ClientPrimaryValidator is the repr. of a primary network validator sent over client
type ClientPrimaryValidator struct {
	ClientStaker
	// The owner the staking reward, if applicable, will go to
	RewardOwner     *ClientOwner
	PotentialReward *uint64
	DelegationFee   float32
	Uptime          *float32
	Connected       *bool
	// The nominators delegating to this validator
	Nominators []ClientPrimaryNominator
}

// ClientPrimaryNominator is the repr. of a primary network nominator sent over client
type ClientPrimaryNominator struct {
	ClientStaker
	RewardOwner     *ClientOwner
	PotentialReward *uint64
}

func apiStakerToClientStaker(validator APIStaker) ClientStaker {
	return ClientStaker{
		TxID:        validator.TxID,
		StartTime:   uint64(validator.StartTime),
		EndTime:     uint64(validator.EndTime),
		Weight:      (*uint64)(validator.Weight),
		StakeAmount: (*uint64)(validator.StakeAmount),
		NodeID:      validator.NodeID,
	}
}

func apiOwnerToClientOwner(rewardOwner *APIOwner) (*ClientOwner, error) {
	if rewardOwner == nil {
		return nil, nil
	}

	addrs, err := address.ParseToIDs(rewardOwner.Addresses)
	return &ClientOwner{
		Locktime:  uint64(rewardOwner.Locktime),
		Threshold: uint32(rewardOwner.Threshold),
		Addresses: addrs,
	}, err
}

func getClientPrimaryValidators(validatorsSliceIntf []interface{}) ([]ClientPrimaryValidator, error) {
	clientValidators := make([]ClientPrimaryValidator, len(validatorsSliceIntf))
	for i, validatorMapIntf := range validatorsSliceIntf {
		validatorMapJSON, err := json.Marshal(validatorMapIntf)
		if err != nil {
			return nil, err
		}

		var apiValidator APIPrimaryValidator
		err = json.Unmarshal(validatorMapJSON, &apiValidator)
		if err != nil {
			return nil, err
		}

		rewardOwner, err := apiOwnerToClientOwner(apiValidator.RewardOwner)
		if err != nil {
			return nil, err
		}

		clientNominators := make([]ClientPrimaryNominator, len(apiValidator.Nominators))
		for j, apiNominator := range apiValidator.Nominators {
			rewardOwner, err := apiOwnerToClientOwner(apiNominator.RewardOwner)
			if err != nil {
				return nil, err
			}

			clientNominators[j] = ClientPrimaryNominator{
				ClientStaker:    apiStakerToClientStaker(apiNominator.APIStaker),
				RewardOwner:     rewardOwner,
				PotentialReward: (*uint64)(apiNominator.PotentialReward),
			}
		}

		clientValidators[i] = ClientPrimaryValidator{
			ClientStaker:    apiStakerToClientStaker(apiValidator.APIStaker),
			RewardOwner:     rewardOwner,
			PotentialReward: (*uint64)(apiValidator.PotentialReward),
			DelegationFee:   float32(apiValidator.DelegationFee),
			Uptime:          (*float32)(apiValidator.Uptime),
			Connected:       &apiValidator.Connected,
			Nominators:      clientNominators,
		}
	}
	return clientValidators, nil
}
