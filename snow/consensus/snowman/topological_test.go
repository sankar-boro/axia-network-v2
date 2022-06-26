// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snowman

import (
	"testing"
)

func TestTopological(t *testing.T) { runConsensusTests(t, TopologicalFactory{}) }
