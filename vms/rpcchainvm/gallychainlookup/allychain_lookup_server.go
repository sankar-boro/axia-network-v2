// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gallychainlookup

import (
	"context"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow"

	allychainlookuppb "github.com/sankar-boro/axia-network-v2/proto/pb/allychainlookup"
)

var _ allychainlookuppb.AllychainLookupServer = &Server{}

// Server is a allychain lookup that is managed over RPC.
type Server struct {
	allychainlookuppb.UnsafeAllychainLookupServer
	aliaser snow.AllychainLookup
}

// NewServer returns a allychain lookup connected to a remote allychain lookup
func NewServer(aliaser snow.AllychainLookup) *Server {
	return &Server{aliaser: aliaser}
}

func (s *Server) AllychainID(
	_ context.Context,
	req *allychainlookuppb.AllychainIDRequest,
) (*allychainlookuppb.AllychainIDResponse, error) {
	chainID, err := ids.ToID(req.ChainId)
	if err != nil {
		return nil, err
	}
	id, err := s.aliaser.AllychainID(chainID)
	if err != nil {
		return nil, err
	}
	return &allychainlookuppb.AllychainIDResponse{
		Id: id[:],
	}, nil
}
