// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pubsub

type Filterer interface {
	Filter(connections []Filter) ([]bool, interface{})
}
