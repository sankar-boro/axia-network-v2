// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package getter

import (
	"time"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow/choices"
	"github.com/sankar-boro/axia/snow/consensus/axia"
	"github.com/sankar-boro/axia/snow/engine/axia/vertex"
	"github.com/sankar-boro/axia/snow/engine/common"
	"github.com/sankar-boro/axia/utils/constants"
	"github.com/sankar-boro/axia/utils/logging"
	"github.com/sankar-boro/axia/utils/metric"
	"github.com/sankar-boro/axia/utils/wrappers"
)

// Get requests are always served, regardless node state (bootstrapping or normal operations).
var _ common.AllGetsServer = &getter{}

func New(storage vertex.Storage, commonCfg common.Config) (common.AllGetsServer, error) {
	gh := &getter{
		storage: storage,
		sender:  commonCfg.Sender,
		cfg:     commonCfg,
		log:     commonCfg.Ctx.Log,
	}

	var err error
	gh.getAncestorsVtxs, err = metric.NewAverager(
		"bs",
		"get_ancestors_vtxs",
		"vertices fetched in a call to GetAncestors",
		commonCfg.Ctx.Registerer,
	)
	return gh, err
}

type getter struct {
	storage vertex.Storage
	sender  common.Sender
	cfg     common.Config

	log              logging.Logger
	getAncestorsVtxs metric.Averager
}

func (gh *getter) GetStateSummaryFrontier(validatorID ids.NodeID, requestID uint32) error {
	gh.log.Debug("GetStateSummaryFrontier(%s, %d) unhandled by this gear. Dropped.", validatorID, requestID)
	return nil
}

func (gh *getter) GetAcceptedStateSummary(validatorID ids.NodeID, requestID uint32, heights []uint64) error {
	gh.log.Debug("GetAcceptedStateSummary(%s, %d) unhandled by this gear. Dropped.", validatorID, requestID)
	return nil
}

func (gh *getter) GetAcceptedFrontier(validatorID ids.NodeID, requestID uint32) error {
	acceptedFrontier := gh.storage.Edge()
	gh.sender.SendAcceptedFrontier(validatorID, requestID, acceptedFrontier)
	return nil
}

func (gh *getter) GetAccepted(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) error {
	acceptedVtxIDs := make([]ids.ID, 0, len(containerIDs))
	for _, vtxID := range containerIDs {
		if vtx, err := gh.storage.GetVtx(vtxID); err == nil && vtx.Status() == choices.Accepted {
			acceptedVtxIDs = append(acceptedVtxIDs, vtxID)
		}
	}
	gh.sender.SendAccepted(nodeID, requestID, acceptedVtxIDs)
	return nil
}

func (gh *getter) GetAncestors(nodeID ids.NodeID, requestID uint32, vtxID ids.ID) error {
	startTime := time.Now()
	gh.log.Verbo("GetAncestors(%s, %d, %s) called", nodeID, requestID, vtxID)
	vertex, err := gh.storage.GetVtx(vtxID)
	if err != nil || vertex.Status() == choices.Unknown {
		gh.log.Verbo("dropping getAncestors")
		return nil // Don't have the requested vertex. Drop message.
	}

	queue := make([]axia.Vertex, 1, gh.cfg.AncestorsMaxContainersSent) // for BFS
	queue[0] = vertex
	ancestorsBytesLen := 0                                                 // length, in bytes, of vertex and its ancestors
	ancestorsBytes := make([][]byte, 0, gh.cfg.AncestorsMaxContainersSent) // vertex and its ancestors in BFS order
	visited := ids.Set{}                                                   // IDs of vertices that have been in queue before
	visited.Add(vertex.ID())

	for len(ancestorsBytes) < gh.cfg.AncestorsMaxContainersSent && len(queue) > 0 && time.Since(startTime) < gh.cfg.MaxTimeGetAncestors {
		var vtx axia.Vertex
		vtx, queue = queue[0], queue[1:] // pop
		vtxBytes := vtx.Bytes()
		// Ensure response size isn't too large. Include wrappers.IntLen because the size of the message
		// is included with each container, and the size is repr. by an int.
		if newLen := wrappers.IntLen + ancestorsBytesLen + len(vtxBytes); newLen < constants.MaxContainersLen {
			ancestorsBytes = append(ancestorsBytes, vtxBytes)
			ancestorsBytesLen = newLen
		} else { // reached maximum response size
			break
		}
		parents, err := vtx.Parents()
		if err != nil {
			return err
		}
		for _, parent := range parents {
			if parent.Status() == choices.Unknown { // Don't have this vertex;ignore
				continue
			}
			if parentID := parent.ID(); !visited.Contains(parentID) { // If already visited, ignore
				queue = append(queue, parent)
				visited.Add(parentID)
			}
		}
	}

	gh.getAncestorsVtxs.Observe(float64(len(ancestorsBytes)))
	gh.sender.SendAncestors(nodeID, requestID, ancestorsBytes)
	return nil
}

func (gh *getter) Get(nodeID ids.NodeID, requestID uint32, vtxID ids.ID) error {
	// If this engine has access to the requested vertex, provide it
	if vtx, err := gh.storage.GetVtx(vtxID); err == nil {
		gh.sender.SendPut(nodeID, requestID, vtxID, vtx.Bytes())
	}
	return nil
}
