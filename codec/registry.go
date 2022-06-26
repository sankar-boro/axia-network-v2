// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package codec

// Registry registers new types that can be marshaled into
type Registry interface {
	RegisterType(interface{}) error
}
