// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package beacon

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/ips"
)

func TestSet(t *testing.T) {
	assert := assert.New(t)

	id0 := ids.NodeID{0}
	id1 := ids.NodeID{1}
	id2 := ids.NodeID{2}

	ip0 := ips.IPPort{
		IP:   net.IPv4zero,
		Port: 0,
	}
	ip1 := ips.IPPort{
		IP:   net.IPv4zero,
		Port: 1,
	}
	ip2 := ips.IPPort{
		IP:   net.IPv4zero,
		Port: 2,
	}

	b0 := New(id0, ip0)
	b1 := New(id1, ip1)
	b2 := New(id2, ip2)

	s := NewSet()

	idsArg := s.IDsArg()
	assert.Equal("", idsArg)
	ipsArg := s.IPsArg()
	assert.Equal("", ipsArg)
	len := s.Len()
	assert.Equal(0, len)

	err := s.Add(b0)
	assert.NoError(err)

	idsArg = s.IDsArg()
	assert.Equal("NodeID-111111111111111111116DBWJs", idsArg)
	ipsArg = s.IPsArg()
	assert.Equal("0.0.0.0:0", ipsArg)
	len = s.Len()
	assert.Equal(1, len)

	err = s.Add(b0)
	assert.ErrorIs(err, errDuplicateID)

	idsArg = s.IDsArg()
	assert.Equal("NodeID-111111111111111111116DBWJs", idsArg)
	ipsArg = s.IPsArg()
	assert.Equal("0.0.0.0:0", ipsArg)
	len = s.Len()
	assert.Equal(1, len)

	err = s.Add(b1)
	assert.NoError(err)

	idsArg = s.IDsArg()
	assert.Equal("NodeID-111111111111111111116DBWJs,NodeID-6HgC8KRBEhXYbF4riJyJFLSHt37UNuRt", idsArg)
	ipsArg = s.IPsArg()
	assert.Equal("0.0.0.0:0,0.0.0.0:1", ipsArg)
	len = s.Len()
	assert.Equal(2, len)

	err = s.Add(b2)
	assert.NoError(err)

	idsArg = s.IDsArg()
	assert.Equal("NodeID-111111111111111111116DBWJs,NodeID-6HgC8KRBEhXYbF4riJyJFLSHt37UNuRt,NodeID-BaMPFdqMUQ46BV8iRcwbVfsam55kMqcp", idsArg)
	ipsArg = s.IPsArg()
	assert.Equal("0.0.0.0:0,0.0.0.0:1,0.0.0.0:2", ipsArg)
	len = s.Len()
	assert.Equal(3, len)

	err = s.RemoveByID(b0.ID())
	assert.NoError(err)

	idsArg = s.IDsArg()
	assert.Equal("NodeID-BaMPFdqMUQ46BV8iRcwbVfsam55kMqcp,NodeID-6HgC8KRBEhXYbF4riJyJFLSHt37UNuRt", idsArg)
	ipsArg = s.IPsArg()
	assert.Equal("0.0.0.0:2,0.0.0.0:1", ipsArg)
	len = s.Len()
	assert.Equal(2, len)

	err = s.RemoveByIP(b1.IP())
	assert.NoError(err)

	idsArg = s.IDsArg()
	assert.Equal("NodeID-BaMPFdqMUQ46BV8iRcwbVfsam55kMqcp", idsArg)
	ipsArg = s.IPsArg()
	assert.Equal("0.0.0.0:2", ipsArg)
	len = s.Len()
	assert.Equal(1, len)
}
