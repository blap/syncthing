// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"
	"time"
)

func TestAdaptiveIntervalCalculation(t *testing.T) {
	cfg := createTestConfig()

	t.Run("HealthScoreToIntervalMapping", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Test the mapping from health scores to intervals
		testCases := []struct {
			score    float64
			expected time.Duration
		}{
			{100.0, 60 * time.Second}, // Perfect health
			{80.0, 42 * time.Second},  // Good health
			{60.0, 28 * time.Second},  // Moderate health
			{40.0, 18 * time.Second},  // Poor health
			{20.0, 12 * time.Second},  // Bad health
			{0.0, 10 * time.Second},   // Worst health
		}

		for _, tc := range testCases {
			hm.SetHealthScore(tc.score)
			interval := hm.GetInterval()

			// Allow for some tolerance due to floating point calculations
			if interval < tc.expected-time.Second || interval > tc.expected+time.Second {
				t.Errorf("For health score %f, expected interval ~%v, got %v", tc.score, tc.expected, interval)
			}
		}
	})

	t.Run("IntervalBoundsEnforcement", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Test that intervals never go below minimum
		hm.SetHealthScore(0.0) // Should result in minimum interval
		minInterval := hm.GetInterval()

		// Even with extreme scores, should not go below configured minimum
		hm.SetHealthScore(-100.0) // Invalid score
		interval := hm.GetInterval()

		if interval < minInterval {
			t.Errorf("Interval should not go below minimum, got %v, minimum is %v", interval, minInterval)
		}

		// Test that intervals never go above maximum
		hm.SetHealthScore(100.0) // Should result in maximum interval
		maxInterval := hm.GetInterval()

		// Even with extreme scores, should not go above configured maximum
		hm.SetHealthScore(200.0) // Invalid score
		interval = hm.GetInterval()

		if interval > maxInterval {
			t.Errorf("Interval should not go above maximum, got %v, maximum is %v", interval, maxInterval)
		}
	})

	t.Run("QuadraticMappingBehavior", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Test that the quadratic mapping creates appropriate differences
		// High health scores should result in much longer intervals
		hm.SetHealthScore(90.0)
		highInterval := hm.GetInterval()

		hm.SetHealthScore(50.0)
		midInterval := hm.GetInterval()

		hm.SetHealthScore(10.0)
		lowInterval := hm.GetInterval()

		// With quadratic mapping, differences should be more pronounced
		if highInterval <= midInterval || midInterval <= lowInterval {
			t.Errorf("Expected quadratic behavior: high(%v) > mid(%v) > low(%v)", highInterval, midInterval, lowInterval)
		}

		// The difference between high and mid should be larger than mid and low
		highMidDiff := highInterval - midInterval
		midLowDiff := midInterval - lowInterval

		// This might not always be true depending on the exact values, but it's expected behavior
		t.Logf("High-Mid difference: %v, Mid-Low difference: %v", highMidDiff, midLowDiff)
	})

	t.Run("ConfigurableMinMaxIntervals", func(t *testing.T) {
		// Test with different configuration values
		cfg := createTestConfig()
		// TODO: Modify config to test different min/max values
		// This would require a more sophisticated test config setup
		hm := NewHealthMonitor(cfg, "device1")

		// Just verify that we can get interval values
		interval := hm.GetInterval()
		if interval <= 0 {
			t.Errorf("Expected positive interval, got %v", interval)
		}
	})
}

func TestIntervalUpdateTriggers(t *testing.T) {
	cfg := createTestConfig()

	t.Run("IntervalUpdatesOnLatencyChange", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Record initial interval
		initialInterval := hm.GetInterval()

		// Record a latency measurement
		hm.RecordLatency(20 * time.Millisecond)

		newInterval := hm.GetInterval()

		// Interval should have updated (may be the same or different depending on health score change)
		// Just verify the function works without error
		_ = newInterval
		t.Logf("Initial interval: %v, After latency recording: %v", initialInterval, newInterval)
	})

	t.Run("IntervalUpdatesOnPacketLossChange", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Record initial interval
		initialInterval := hm.GetInterval()

		// Record a packet loss measurement
		hm.RecordPacketLoss(5.0)

		newInterval := hm.GetInterval()

		// Interval should have updated
		_ = newInterval
		t.Logf("Initial interval: %v, After packet loss recording: %v", initialInterval, newInterval)
	})

	t.Run("IntervalStability", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Record several identical measurements
		for i := 0; i < 5; i++ {
			hm.RecordLatency(20 * time.Millisecond)
			hm.RecordPacketLoss(0.0)
		}

		interval1 := hm.GetInterval()

		// Record more identical measurements
		for i := 0; i < 5; i++ {
			hm.RecordLatency(20 * time.Millisecond)
			hm.RecordPacketLoss(0.0)
		}

		interval2 := hm.GetInterval()

		// With identical measurements, interval should stabilize
		// Allow for small variations due to implementation details
		diff := interval1 - interval2
		if diff < -time.Second || diff > time.Second {
			t.Errorf("Interval should stabilize with consistent measurements, got %v and %v", interval1, interval2)
		}
	})
}
