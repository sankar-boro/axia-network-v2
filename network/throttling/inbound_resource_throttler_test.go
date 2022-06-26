// Copyright (C) 2019-2022, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package throttling

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"

	"github.com/sankar-boro/axia/ids"
	"github.com/sankar-boro/axia/snow/networking/tracker"
	"github.com/sankar-boro/axia/snow/validators"
	"github.com/sankar-boro/axia/utils/math/meter"
	"github.com/sankar-boro/axia/utils/resource"
	"github.com/sankar-boro/axia/utils/timer/mockable"
)

func TestNewSystemThrottler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	assert := assert.New(t)
	reg := prometheus.NewRegistry()
	clock := mockable.Clock{}
	clock.Set(time.Now())
	vdrs := validators.NewSet()
	resourceTracker, err := tracker.NewResourceTracker(reg, resource.NoUsage, meter.ContinuousFactory{}, time.Second)
	assert.NoError(err)
	cpuTracker := resourceTracker.CPUTracker()

	config := SystemThrottlerConfig{
		Clock:           clock,
		MaxRecheckDelay: time.Second,
	}
	targeter := tracker.NewMockTargeter(ctrl)
	throttlerIntf, err := NewSystemThrottler("", reg, config, vdrs, cpuTracker, targeter)
	assert.NoError(err)
	throttler, ok := throttlerIntf.(*systemThrottler)
	assert.True(ok)
	assert.EqualValues(clock, config.Clock)
	assert.EqualValues(time.Second, config.MaxRecheckDelay)
	assert.EqualValues(cpuTracker, throttler.tracker)
	assert.EqualValues(targeter, throttler.targeter)
}

func TestSystemThrottler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	assert := assert.New(t)

	// Setup
	mockTracker := tracker.NewMockTracker(ctrl)
	maxRecheckDelay := 100 * time.Millisecond
	config := SystemThrottlerConfig{
		MaxRecheckDelay: maxRecheckDelay,
	}
	vdrs := validators.NewSet()
	vdrID, nonVdrID := ids.GenerateTestNodeID(), ids.GenerateTestNodeID()
	err := vdrs.AddWeight(vdrID, 1)
	assert.NoError(err)
	targeter := tracker.NewMockTargeter(ctrl)
	throttler, err := NewSystemThrottler("", prometheus.NewRegistry(), config, vdrs, mockTracker, targeter)
	assert.NoError(err)

	// Case: Actual usage <= target usage; should return immediately
	// for both validator and non-validator
	targeter.EXPECT().TargetUsage(vdrID).Return(1.0).Times(1)
	mockTracker.EXPECT().Usage(vdrID, gomock.Any()).Return(0.9).Times(1)

	throttler.Acquire(context.Background(), vdrID)

	targeter.EXPECT().TargetUsage(nonVdrID).Return(1.0).Times(1)
	mockTracker.EXPECT().Usage(nonVdrID, gomock.Any()).Return(0.9).Times(1)

	throttler.Acquire(context.Background(), nonVdrID)

	// Case: Actual usage > target usage; we should wait.
	// In the first loop iteration inside acquire,
	// say the actual usage exceeds the target.
	targeter.EXPECT().TargetUsage(vdrID).Return(0.0).Times(1)
	mockTracker.EXPECT().Usage(vdrID, gomock.Any()).Return(1.0).Times(1)
	// Note we'll only actually wait [maxRecheckDelay]. We set [timeUntilAtDiskTarget]
	// much larger to assert that the min recheck frequency is honored below.
	timeUntilAtDiskTarget := 100 * maxRecheckDelay
	mockTracker.EXPECT().TimeUntilUsage(vdrID, gomock.Any(), gomock.Any()).Return(timeUntilAtDiskTarget).Times(1)

	// The second iteration, say the usage is OK.
	targeter.EXPECT().TargetUsage(vdrID).Return(1.0).Times(1)
	mockTracker.EXPECT().Usage(vdrID, gomock.Any()).Return(0.0).Times(1)

	onAcquire := make(chan struct{})

	// Check for validator
	go func() {
		throttler.Acquire(context.Background(), vdrID)
		onAcquire <- struct{}{}
	}()
	// Make sure the min re-check frequency is honored
	select {
	// Use 5*maxRecheckDelay and not just maxRecheckDelay to give a buffer
	// and avoid flakiness. If the min re-check freq isn't honored,
	// we'll wait [timeUntilAtDiskTarget].
	case <-time.After(5 * maxRecheckDelay):
		assert.FailNow("should have returned after about [maxRecheckDelay]")
	case <-onAcquire:
	}

	targeter.EXPECT().TargetUsage(nonVdrID).Return(0.0).Times(1)
	mockTracker.EXPECT().Usage(nonVdrID, gomock.Any()).Return(1.0).Times(1)

	mockTracker.EXPECT().TimeUntilUsage(nonVdrID, gomock.Any(), gomock.Any()).Return(timeUntilAtDiskTarget).Times(1)

	targeter.EXPECT().TargetUsage(nonVdrID).Return(1.0).Times(1)
	mockTracker.EXPECT().Usage(nonVdrID, gomock.Any()).Return(0.0).Times(1)

	// Check for non-validator
	go func() {
		throttler.Acquire(context.Background(), nonVdrID)
		onAcquire <- struct{}{}
	}()
	// Make sure the min re-check frequency is honored
	select {
	// Use 5*maxRecheckDelay and not just maxRecheckDelay to give a buffer
	// and avoid flakiness. If the min re-check freq isn't honored,
	// we'll wait [timeUntilAtDiskTarget].
	case <-time.After(5 * maxRecheckDelay):
		assert.FailNow("should have returned after about [maxRecheckDelay]")
	case <-onAcquire:
	}
}

func TestSystemThrottlerContextCancel(t *testing.T) {
	assert := assert.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	mockTracker := tracker.NewMockTracker(ctrl)
	maxRecheckDelay := 10 * time.Second
	config := SystemThrottlerConfig{
		MaxRecheckDelay: maxRecheckDelay,
	}
	vdrs := validators.NewSet()
	vdrID := ids.GenerateTestNodeID()
	err := vdrs.AddWeight(vdrID, 1)
	assert.NoError(err)
	targeter := tracker.NewMockTargeter(ctrl)
	throttler, err := NewSystemThrottler("", prometheus.NewRegistry(), config, vdrs, mockTracker, targeter)
	assert.NoError(err)

	// Case: Actual usage > target usage; we should wait.
	// Mock the tracker so that the first loop iteration inside acquire,
	// it says the actual usage exceeds the target.
	// There should be no second iteration because we've already returned.
	targeter.EXPECT().TargetUsage(vdrID).Return(0.0).Times(1)
	mockTracker.EXPECT().Usage(vdrID, gomock.Any()).Return(1.0).Times(1)
	mockTracker.EXPECT().TimeUntilUsage(vdrID, gomock.Any(), gomock.Any()).Return(maxRecheckDelay).Times(1)
	onAcquire := make(chan struct{})
	// Pass a canceled context into Acquire so that it returns immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	go func() {
		throttler.Acquire(ctx, vdrID)
		onAcquire <- struct{}{}
	}()
	select {
	case <-onAcquire:
	case <-time.After(maxRecheckDelay / 2):
		// Make sure Acquire returns well before the second check (i.e. "immediately")
		assert.Fail("should have returned immediately")
	}
}
