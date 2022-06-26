// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nftfx

import (
	"testing"

	"github.com/sankar-boro/axia/vms/components/verify"
)

func TestCredentialState(t *testing.T) {
	intf := interface{}(&Credential{})
	if _, ok := intf.(verify.State); ok {
		t.Fatalf("shouldn't be marked as state")
	}
}
