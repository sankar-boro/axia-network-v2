// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package axia

import (
	"testing"
)

func TestTopological(t *testing.T) { runConsensusTests(t, TopologicalFactory{}) }
