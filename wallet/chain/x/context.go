// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package x

import (
	stdcontext "context"

	"github.com/sankar-boro/axia/api/info"
	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/utils/constants"
	"github.com/sankar-boro/axia/vms/avm"
)

var _ Context = &context{}

type Context interface {
	NetworkID() uint32
	HRP() string
	BlockchainID() ids.ID
	AXCAssetID() ids.ID
	BaseTxFee() uint64
	CreateAssetTxFee() uint64
}

type context struct {
	networkID        uint32
	hrp              string
	blockchainID     ids.ID
	axcAssetID      ids.ID
	baseTxFee        uint64
	createAssetTxFee uint64
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

	chainID, err := infoClient.GetBlockchainID(ctx, "Swap")
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
		chainID,
		asset.AssetID,
		uint64(txFees.TxFee),
		uint64(txFees.CreateAssetTxFee),
	), nil
}

func NewContext(
	networkID uint32,
	blockchainID ids.ID,
	axcAssetID ids.ID,
	baseTxFee uint64,
	createAssetTxFee uint64,
) Context {
	return &context{
		networkID:        networkID,
		hrp:              constants.GetHRP(networkID),
		blockchainID:     blockchainID,
		axcAssetID:      axcAssetID,
		baseTxFee:        baseTxFee,
		createAssetTxFee: createAssetTxFee,
	}
}

func (c *context) NetworkID() uint32        { return c.networkID }
func (c *context) HRP() string              { return c.hrp }
func (c *context) BlockchainID() ids.ID     { return c.blockchainID }
func (c *context) AXCAssetID() ids.ID      { return c.axcAssetID }
func (c *context) BaseTxFee() uint64        { return c.baseTxFee }
func (c *context) CreateAssetTxFee() uint64 { return c.createAssetTxFee }
