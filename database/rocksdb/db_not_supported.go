//go:build !linux || !amd64 || !rocksdballowed
// +build !linux !amd64 !rocksdballowed

// ^ Only build this file if this computer is not Linux OR it's not AMD64 OR rocksdb is not allowed
// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package rocksdb

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sankar-boro/axia/database"
	"github.com/sankar-boro/axia/utils/logging"
)

var errUnsupportedDatabase = errors.New("database isn't suppported")

// New returns an error.
func New(string, []byte, logging.Logger, string, prometheus.Registerer) (database.Database, error) {
	return nil, errUnsupportedDatabase
}
