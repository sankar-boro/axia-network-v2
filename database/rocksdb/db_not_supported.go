//go:build !linux || !amd64 || !rocksdballowed
// +build !linux !amd64 !rocksdballowed

// ^ Only build this file if this computer is not Linux OR it's not AMD64 OR rocksdb is not allowed
// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package rocksdb

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sankar-boro/axia-network-v2/database"
	"github.com/sankar-boro/axia-network-v2/utils/logging"
)

var errUnsupportedDatabase = errors.New("database isn't suppported")

// New returns an error.
func New(string, []byte, logging.Logger, string, prometheus.Registerer) (database.Database, error) {
	return nil, errUnsupportedDatabase
}
