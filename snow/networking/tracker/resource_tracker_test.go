// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tracker

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia-network-v2/ids"
	"github.com/sankar-boro/axia-network-v2/utils/math/meter"
	"github.com/sankar-boro/axia-network-v2/utils/resource"
)

func TestNewCPUTracker(t *testing.T) {
	assert := assert.New(t)

	reg := prometheus.NewRegistry()
	halflife := 5 * time.Second
	factory := &meter.ContinuousFactory{}

	trackerIntf, err := NewResourceTracker(reg, resource.NoUsage, factory, halflife)
	assert.NoError(err)
	tracker, ok := trackerIntf.(*resourceTracker)
	assert.True(ok)
	assert.Equal(factory, tracker.factory)
	assert.NotNil(tracker.processingMeter)
	assert.Equal(halflife, tracker.halflife)
	assert.NotNil(tracker.meters)
	assert.NotNil(tracker.metrics)
}

func TestCPUTracker(t *testing.T) {
	halflife := 5 * time.Second

	ctrl := gomock.NewController(t)
	mockUser := resource.NewMockUser(ctrl)
	mockUser.EXPECT().CPUUsage().Return(1.0).Times(3)

	tracker, err := NewResourceTracker(prometheus.NewRegistry(), mockUser, meter.ContinuousFactory{}, time.Second)
	assert.NoError(t, err)

	node1 := ids.NodeID{1}
	node2 := ids.NodeID{2}

	// Note that all the durations between start and end are [halflife].
	startTime1 := time.Now()
	endTime1 := startTime1.Add(halflife)
	// Note that all CPU usage is attributed to at-large allocation.
	tracker.StartProcessing(node1, startTime1)
	tracker.StopProcessing(node1, endTime1)

	startTime2 := endTime1
	endTime2 := startTime2.Add(halflife)
	// Note that all CPU usage is attributed to at-large allocation.
	tracker.StartProcessing(node2, startTime2)
	tracker.StopProcessing(node2, endTime2)

	cpuTracker := tracker.CPUTracker()

	node1Utilization := cpuTracker.Usage(node1, endTime2)
	node2Utilization := cpuTracker.Usage(node2, endTime2)
	if node1Utilization >= node2Utilization {
		t.Fatalf("Utilization should have been higher for the more recent spender")
	}

	cumulative := cpuTracker.TotalUsage()
	sum := node1Utilization + node2Utilization
	if cumulative != sum {
		t.Fatalf("Cumulative utilization: %f should have been equal to the sum of the spenders: %f", cumulative, sum)
	}

	mockUser.EXPECT().CPUUsage().Return(.5).Times(3)

	startTime3 := endTime2
	endTime3 := startTime3.Add(halflife)
	newNode1Utilization := cpuTracker.Usage(node1, endTime3)
	if newNode1Utilization >= node1Utilization {
		t.Fatalf("node CPU utilization should decrease over time")
	}
	newCumulative := cpuTracker.TotalUsage()
	if newCumulative >= cumulative {
		t.Fatal("at-large CPU utilization should decrease over time ")
	}

	startTime4 := endTime3
	endTime4 := startTime4.Add(halflife)
	// Note that only half of CPU usage is attributed to at-large allocation.
	tracker.StartProcessing(node1, startTime4)
	tracker.StopProcessing(node1, endTime4)

	cumulative = cpuTracker.TotalUsage()
	sum = node1Utilization + node2Utilization
	if cumulative >= sum {
		t.Fatal("Sum of CPU usage should exceed cumulative at-large utilization")
	}
}

func TestCPUTrackerTimeUntilCPUUtilization(t *testing.T) {
	halflife := 5 * time.Second
	tracker, err := NewResourceTracker(prometheus.NewRegistry(), resource.NoUsage, meter.ContinuousFactory{}, halflife)
	assert.NoError(t, err)
	now := time.Now()
	nodeID := ids.GenerateTestNodeID()
	// Start the meter
	tracker.StartProcessing(nodeID, now)
	// One halflife passes; stop the meter
	now = now.Add(halflife)
	tracker.StopProcessing(nodeID, now)
	cpuTracker := tracker.CPUTracker()
	// Read the current value
	currentVal := cpuTracker.Usage(nodeID, now)
	// Suppose we want to wait for the value to be
	// a third of its current value
	desiredVal := currentVal / 3
	// See when that should happen
	timeUntilDesiredVal := cpuTracker.TimeUntilUsage(nodeID, now, desiredVal)
	// Get the actual value at that time
	now = now.Add(timeUntilDesiredVal)
	actualVal := cpuTracker.Usage(nodeID, now)
	// Make sure the actual/expected are close
	assert.InDelta(t, desiredVal, actualVal, .00001)
	// Make sure TimeUntilUsage returns the zero duration if
	// the value provided >= the current value
	assert.Zero(t, cpuTracker.TimeUntilUsage(nodeID, now, actualVal))
	assert.Zero(t, cpuTracker.TimeUntilUsage(nodeID, now, actualVal+.1))
	// Make sure it returns the zero duration if the node isn't known
	assert.Zero(t, cpuTracker.TimeUntilUsage(ids.GenerateTestNodeID(), now, 0.0001))
}
