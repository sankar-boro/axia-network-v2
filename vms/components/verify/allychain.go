// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package verify

import (
	"errors"
	"fmt"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow"
)

var (
	errSameChainID         = errors.New("same chainID")
	errMismatchedAllychainIDs = errors.New("mismatched allychainIDs")
)

// SameAllychain verifies that the provided [ctx] was provided to a chain in the
// same allychain as [peerChainID], but not the same chain. If this verification
// fails, a non-nil error will be returned.
func SameAllychain(ctx *snow.Context, peerChainID ids.ID) error {
	if peerChainID == ctx.ChainID {
		return errSameChainID
	}

	allychainID, err := ctx.SNLookup.AllychainID(peerChainID)
	if err != nil {
		return fmt.Errorf("failed to get allychain of %q: %w", peerChainID, err)
	}
	if ctx.AllychainID != allychainID {
		return fmt.Errorf("%w; expected %q got %q", errMismatchedAllychainIDs, ctx.AllychainID, allychainID)
	}
	return nil
}
