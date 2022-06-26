// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	stdcontext "context"

	"github.com/sankar-boro/axia-network-v2/api/info"
	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/constants"
	"github.com/sankar-boro/axia-network-v2/vms/avm"
)

var _ Context = &context{}

type Context interface {
	NetworkID() uint32
	HRP() string
	AXCAssetID() ids.ID
	BaseTxFee() uint64
	CreateAllychainTxFee() uint64
	CreateBlockchainTxFee() uint64
}

type context struct {
	networkID             uint32
	hrp                   string
	axcAssetID           ids.ID
	baseTxFee             uint64
	createAllychainTxFee     uint64
	createBlockchainTxFee uint64
}

func NewContextFromURI(ctx stdcontext.Context, uri string) (Context, error) {
	infoClient := info.NewClient(uri)
	swapChainClient := avm.NewClient(uri, "Swap")
	return NewContextFromClients(ctx, infoClient, swapChainClient)
}

func NewContextFromClients(
	ctx stdcontext.Context,
	infoClient info.Client,
	swapChainClient avm.Client,
) (Context, error) {
	networkID, err := infoClient.GetNetworkID(ctx)
	if err != nil {
		return nil, err
	}

	asset, err := swapChainClient.GetAssetDescription(ctx, "AXC")
	if err != nil {
		return nil, err
	}

	txFees, err := infoClient.GetTxFee(ctx)
	if err != nil {
		return nil, err
	}

	return NewContext(
		networkID,
		asset.AssetID,
		uint64(txFees.TxFee),
		uint64(txFees.CreateAllychainTxFee),
		uint64(txFees.CreateBlockchainTxFee),
	), nil
}

func NewContext(
	networkID uint32,
	axcAssetID ids.ID,
	baseTxFee uint64,
	createAllychainTxFee uint64,
	createBlockchainTxFee uint64,
) Context {
	return &context{
		networkID:             networkID,
		hrp:                   constants.GetHRP(networkID),
		axcAssetID:           axcAssetID,
		baseTxFee:             baseTxFee,
		createAllychainTxFee:     createAllychainTxFee,
		createBlockchainTxFee: createBlockchainTxFee,
	}
}

func (c *context) NetworkID() uint32             { return c.networkID }
func (c *context) HRP() string                   { return c.hrp }
func (c *context) AXCAssetID() ids.ID           { return c.axcAssetID }
func (c *context) BaseTxFee() uint64             { return c.baseTxFee }
func (c *context) CreateAllychainTxFee() uint64     { return c.createAllychainTxFee }
func (c *context) CreateBlockchainTxFee() uint64 { return c.createBlockchainTxFee }
