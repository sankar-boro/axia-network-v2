// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/ids"
)

func TestRequests(t *testing.T) {
	req := Requests{}

	length := req.Len()
	assert.Equal(t, 0, length, "should have had no outstanding requests")

	_, removed := req.Remove(ids.EmptyNodeID, 0)
	assert.False(t, removed, "shouldn't have removed the request")

	removed = req.RemoveAny(ids.Empty)
	assert.False(t, removed, "shouldn't have removed the request")

	constains := req.Contains(ids.Empty)
	assert.False(t, constains, "shouldn't contain this request")

	req.Add(ids.EmptyNodeID, 0, ids.Empty)

	length = req.Len()
	assert.Equal(t, 1, length, "should have had one outstanding request")

	_, removed = req.Remove(ids.EmptyNodeID, 1)
	assert.False(t, removed, "shouldn't have removed the request")

	_, removed = req.Remove(ids.NodeID{1}, 0)
	assert.False(t, removed, "shouldn't have removed the request")

	constains = req.Contains(ids.Empty)
	assert.True(t, constains, "should contain this request")

	length = req.Len()
	assert.Equal(t, 1, length, "should have had one outstanding request")

	req.Add(ids.EmptyNodeID, 10, ids.Empty.Prefix(0))

	length = req.Len()
	assert.Equal(t, 2, length, "should have had two outstanding requests")

	_, removed = req.Remove(ids.EmptyNodeID, 1)
	assert.False(t, removed, "shouldn't have removed the request")

	_, removed = req.Remove(ids.NodeID{1}, 0)
	assert.False(t, removed, "shouldn't have removed the request")

	constains = req.Contains(ids.Empty)
	assert.True(t, constains, "should contain this request")

	length = req.Len()
	assert.Equal(t, 2, length, "should have had two outstanding requests")

	removedID, removed := req.Remove(ids.EmptyNodeID, 0)
	assert.Equal(t, ids.Empty, removedID, "should have removed the requested ID")
	assert.True(t, removed, "should have removed the request")

	removedID, removed = req.Remove(ids.EmptyNodeID, 10)
	assert.Equal(t, ids.Empty.Prefix(0), removedID, "should have removed the requested ID")
	assert.True(t, removed, "should have removed the request")

	length = req.Len()
	assert.Equal(t, 0, length, "should have had no outstanding requests")

	req.Add(ids.EmptyNodeID, 0, ids.Empty)

	length = req.Len()
	assert.Equal(t, 1, length, "should have had one outstanding request")

	removed = req.RemoveAny(ids.Empty)
	assert.True(t, removed, "should have removed the request")

	length = req.Len()
	assert.Equal(t, 0, length, "should have had no outstanding requests")

	removed = req.RemoveAny(ids.Empty)
	assert.False(t, removed, "shouldn't have removed the request")

	length = req.Len()
	assert.Equal(t, 0, length, "should have had no outstanding requests")
}
