// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package axia

// Factory returns new instances of Consensus
type Factory interface {
	New() Consensus
}
