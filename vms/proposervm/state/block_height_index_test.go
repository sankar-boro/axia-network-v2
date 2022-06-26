// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia/database/memdb"
	"github.com/sankar-boro/axia/database/versiondb"
	"github.com/sankar-boro/axia/utils/logging"
)

func TestHasIndexReset(t *testing.T) {
	a := assert.New(t)

	db := memdb.New()
	vdb := versiondb.New(db)
	s := New(vdb)
	wasReset, err := s.HasIndexReset()
	a.NoError(err)
	a.False(wasReset)
	err = s.ResetHeightIndex(logging.NoLog{}, vdb)
	a.NoError(err)
	wasReset, err = s.HasIndexReset()
	a.NoError(err)
	a.True(wasReset)
}
