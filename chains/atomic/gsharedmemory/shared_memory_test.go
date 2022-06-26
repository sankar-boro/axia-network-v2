// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gsharedmemory

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/sankar-boro/axia-network-v2/chains/atomic"
	"github.com/sankar-boro/axia-network-v2/database"
	"github.com/sankar-boro/axia-network-v2/database/memdb"
	"github.com/sankar-boro/axia-network-v2/database/prefixdb"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/logging"
	"github.com/sankar-boro/axia-network-v2/utils/units"
	"github.com/sankar-boro/axia-network-v2/vms/rpcchainvm/grpcutils"

	sharedmemorypb "github.com/sankar-boro/axia-network-v2/proto/pb/sharedmemory"
)

const (
	bufSize = units.MiB
)

func TestInterface(t *testing.T) {
	assert := assert.New(t)

	chainID0 := ids.GenerateTestID()
	chainID1 := ids.GenerateTestID()

	for _, test := range atomic.SharedMemoryTests {
		m := atomic.Memory{}
		baseDB := memdb.New()
		memoryDB := prefixdb.New([]byte{0}, baseDB)
		testDB := prefixdb.New([]byte{1}, baseDB)

		err := m.Initialize(logging.NoLog{}, memoryDB)
		assert.NoError(err)

		sm0, conn0 := wrapSharedMemory(t, m.NewSharedMemory(chainID0), baseDB)
		sm1, conn1 := wrapSharedMemory(t, m.NewSharedMemory(chainID1), baseDB)

		test(t, chainID0, chainID1, sm0, sm1, testDB)

		err = conn0.Close()
		assert.NoError(err)

		err = conn1.Close()
		assert.NoError(err)
	}
}

func wrapSharedMemory(t *testing.T, sm atomic.SharedMemory, db database.Database) (atomic.SharedMemory, io.Closer) {
	listener := bufconn.Listen(bufSize)
	serverCloser := grpcutils.ServerCloser{}

	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		server := grpcutils.NewDefaultServer(opts)
		sharedmemorypb.RegisterSharedMemoryServer(server, NewServer(sm, db))
		serverCloser.Add(server)
		return server
	}

	go grpcutils.Serve(listener, serverFunc)

	dialer := grpc.WithContextDialer(
		func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		},
	)

	dopts := grpcutils.DefaultDialOptions
	dopts = append(dopts, dialer)
	conn, err := grpcutils.Dial("", dopts...)
	if err != nil {
		t.Fatalf("Failed to dial: %s", err)
	}

	rpcsm := NewClient(sharedmemorypb.NewSharedMemoryClient(conn))
	return rpcsm, conn
}
