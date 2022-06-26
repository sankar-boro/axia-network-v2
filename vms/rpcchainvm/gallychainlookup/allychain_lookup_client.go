// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gallychainlookup

import (
	"context"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/snow"

	allychainlookuppb "github.com/sankar-boro/axia-network-v2/proto/pb/allychainlookup"
)

var _ snow.AllychainLookup = &Client{}

// Client is a allychain lookup that talks over RPC.
type Client struct {
	client allychainlookuppb.AllychainLookupClient
}

// NewClient returns an alias lookup connected to a remote alias lookup
func NewClient(client allychainlookuppb.AllychainLookupClient) *Client {
	return &Client{client: client}
}

func (c *Client) AllychainID(chainID ids.ID) (ids.ID, error) {
	resp, err := c.client.AllychainID(context.Background(), &allychainlookuppb.AllychainIDRequest{
		ChainId: chainID[:],
	})
	if err != nil {
		return ids.ID{}, err
	}
	return ids.ToID(resp.Id)
}
