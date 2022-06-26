// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package propertyfx

import (
	"testing"
)

func TestFactory(t *testing.T) {
	factory := Factory{}
	if fx, err := factory.New(nil); err != nil {
		t.Fatal(err)
	} else if fx == nil {
		t.Fatalf("Factory.New returned nil")
	}
}
