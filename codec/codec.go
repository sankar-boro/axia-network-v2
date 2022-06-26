// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package codec

import "github.com/sankar-boro/axia/utils/wrappers"

// Codec marshals and unmarshals
type Codec interface {
	MarshalInto(interface{}, *wrappers.Packer) error
	Unmarshal([]byte, interface{}) error
}
