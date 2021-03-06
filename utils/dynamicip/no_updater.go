// Copyright (C) 2019-2022, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package dynamicip

import "github.com/sankar-boro/axia-network-v2/utils/logging"

var _ Updater = noUpdater{}

func NewNoUpdater() Updater {
	return noUpdater{}
}

type noUpdater struct{}

func (noUpdater) Dispatch(logging.Logger) {}

func (noUpdater) Stop() {}
