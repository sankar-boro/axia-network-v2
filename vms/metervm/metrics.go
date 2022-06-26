// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metervm

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sankar-boro/axia-network-v2/utils/metric"
	"github.com/sankar-boro/axia-network-v2/utils/wrappers"
)

func newAverager(namespace, name string, reg prometheus.Registerer, errs *wrappers.Errs) metric.Averager {
	return metric.NewAveragerWithErrs(
		namespace,
		name,
		fmt.Sprintf("time (in ns) of a %s", name),
		reg,
		errs,
	)
}
