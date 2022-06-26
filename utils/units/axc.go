// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package units

// Denominations of value
const (
	NanoAxc  uint64 = 1
	MicroAxc uint64 = 1000 * NanoAxc
	Schmeckle uint64 = 49*MicroAxc + 463*NanoAxc
	MilliAxc uint64 = 1000 * MicroAxc
	Axc      uint64 = 1000 * MilliAxc
	KiloAxc  uint64 = 1000 * Axc
	MegaAxc  uint64 = 1000 * KiloAxc
)
