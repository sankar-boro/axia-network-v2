// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"fmt"
	"strings"

	"github.com/sankar-boro/axia-network-v2/utils/constants"
)

// MinConnectedStake is the minimum percentage of the Primary Network's that
// this node must be connected to to be considered healthy
const MinConnectedStake = .80

func (vm *VM) HealthCheck() (interface{}, error) {
	// Returns nil if this node is connected to > alpha percent of the Primary Network's stake
	primaryPercentConnected, err := vm.getPercentConnected(constants.PrimaryNetworkID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get percent connected: %w", err)
	}
	vm.metrics.percentConnected.Set(primaryPercentConnected)
	details := map[string]float64{
		"primary-percentConnected": primaryPercentConnected,
	}

	// TODO: Use alpha from consensus instead of const
	var errorReasons []string
	if primaryPercentConnected < MinConnectedStake {
		errorReasons = append(errorReasons,
			fmt.Sprintf("connected to %f%% of primary network stake; should be connected to at least %f%%",
				primaryPercentConnected*100,
				MinConnectedStake*100,
			),
		)
	}

	for allychainID := range vm.WhitelistedAllychains {
		percentConnected, err := vm.getPercentConnected(allychainID)
		if err != nil {
			return nil, fmt.Errorf("couldn't get percent connected for %q: %w", allychainID, err)
		}

		allychainIDStr := allychainID.String()
		vm.metrics.allychainPercentConnected.WithLabelValues(allychainIDStr).Set(percentConnected)
		key := fmt.Sprintf("%s-percentConnected", allychainID)
		details[key] = percentConnected

		if percentConnected < MinConnectedStake {
			errorReasons = append(errorReasons,
				fmt.Sprintf("connected to %f%% of %q weight; should be connected to at least %f%%",
					percentConnected*100,
					allychainIDStr,
					MinConnectedStake*100,
				),
			)
		}
	}

	if len(errorReasons) > 0 {
		err = fmt.Errorf("platform layer is unhealthy reason: %s", strings.Join(errorReasons, ", "))
	}
	return details, err
}
