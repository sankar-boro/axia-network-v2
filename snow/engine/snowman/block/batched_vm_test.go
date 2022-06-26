// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package block

import (
	"errors"
	"testing"
	"time"

	"github.com/sankar-boro/avalanchego/database"
	"github.com/sankar-boro/avalanchego/ids"
	"github.com/sankar-boro/avalanchego/snow/consensus/snowman"
	"github.com/stretchr/testify/assert"
)

func TestGetAncestorsDatabaseNotFound(t *testing.T) {
	vm := &TestVM{}
	someID := ids.GenerateTestID()
	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		assert.Equal(t, someID, id)
		return nil, database.ErrNotFound
	}
	containers, err := GetAncestors(vm, someID, 10, 10, 1*time.Second)
	assert.NoError(t, err)
	assert.Len(t, containers, 0)
}

// TestGetAncestorsPropagatesErrors checks errors other than
// database.ErrNotFound propagate to caller.
func TestGetAncestorsPropagatesErrors(t *testing.T) {
	vm := &TestVM{}
	someID := ids.GenerateTestID()
	someError := errors.New("some error that is not ErrNotFound")
	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		assert.Equal(t, someID, id)
		return nil, someError
	}
	containers, err := GetAncestors(vm, someID, 10, 10, 1*time.Second)
	assert.Nil(t, containers)
	assert.ErrorIs(t, err, someError)
}
