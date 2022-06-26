// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package block

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"time"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/hashing"
	"github.com/sankar-boro/axia-network-v2/utils/wrappers"
)

func BuildUnsigned(
	parentID ids.ID,
	timestamp time.Time,
	coreChainHeight uint64,
	blockBytes []byte,
) (SignedBlock, error) {
	var block SignedBlock = &statelessBlock{
		StatelessBlock: statelessUnsignedBlock{
			ParentID:     parentID,
			Timestamp:    timestamp.Unix(),
			CoreChainHeight: coreChainHeight,
			Certificate:  nil,
			Block:        blockBytes,
		},
		timestamp: timestamp,
	}

	bytes, err := c.Marshal(version, &block)
	if err != nil {
		return nil, err
	}
	return block, block.initialize(bytes)
}

func Build(
	parentID ids.ID,
	timestamp time.Time,
	coreChainHeight uint64,
	cert *x509.Certificate,
	blockBytes []byte,
	chainID ids.ID,
	key crypto.Signer,
) (SignedBlock, error) {
	block := &statelessBlock{
		StatelessBlock: statelessUnsignedBlock{
			ParentID:     parentID,
			Timestamp:    timestamp.Unix(),
			CoreChainHeight: coreChainHeight,
			Certificate:  cert.Raw,
			Block:        blockBytes,
		},
		timestamp: timestamp,
		cert:      cert,
		proposer:  ids.NodeIDFromCert(cert),
	}
	var blockIntf SignedBlock = block

	unsignedBytesWithEmptySignature, err := c.Marshal(version, &blockIntf)
	if err != nil {
		return nil, err
	}

	// The serialized form of the block is the unsignedBytes followed by the
	// signature, which is prefixed by a uint32. Because we are marshalling the
	// block with an empty signature, we only need to strip off the length
	// prefix to get the unsigned bytes.
	lenUnsignedBytes := len(unsignedBytesWithEmptySignature) - wrappers.IntLen
	unsignedBytes := unsignedBytesWithEmptySignature[:lenUnsignedBytes]
	block.id = hashing.ComputeHash256Array(unsignedBytes)

	header, err := BuildHeader(chainID, parentID, block.id)
	if err != nil {
		return nil, err
	}

	headerHash := hashing.ComputeHash256(header.Bytes())
	block.Signature, err = key.Sign(rand.Reader, headerHash, crypto.SHA256)
	if err != nil {
		return nil, err
	}

	block.bytes, err = c.Marshal(version, &blockIntf)
	return block, err
}

func BuildHeader(
	chainID ids.ID,
	parentID ids.ID,
	bodyID ids.ID,
) (Header, error) {
	header := statelessHeader{
		Chain:  chainID,
		Parent: parentID,
		Body:   bodyID,
	}

	bytes, err := c.Marshal(version, &header)
	header.bytes = bytes
	return &header, err
}

// BuildOption the option block
// [parentID] is the ID of this option's wrapper parent block
// [innerBytes] is the byte representation of a child option block
func BuildOption(
	parentID ids.ID,
	innerBytes []byte,
) (Block, error) {
	var block Block = &option{
		PrntID:     parentID,
		InnerBytes: innerBytes,
	}

	bytes, err := c.Marshal(version, &block)
	if err != nil {
		return nil, err
	}
	return block, block.initialize(bytes)
}
