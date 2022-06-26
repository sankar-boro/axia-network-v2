// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"github.com/sankar-boro/axia/ids"
)

// Fx wraps an instance of a feature extension
type Fx struct {
	ID ids.ID
	Fx interface{}
}
