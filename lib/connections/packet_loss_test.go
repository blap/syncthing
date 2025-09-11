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

func TestPacketLossTracking(t *testing.T) {
	// Create a mock config wrapper
	cfg := createTestConfig()

	t.Run("RecordPacketLoss", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Record no packet loss
		hm.RecordPacketLoss(0.0)

		// Health score should improve with no packet loss
		score := hm.GetHealthScore()
		if score <= 50.0 {
			t.Errorf("Expected health score to improve with no packet loss, got %f", score)
		}

		// Record high packet loss
		hm.RecordPacketLoss(50.0)

		// Health score should decrease with high packet loss
		newScore := hm.GetHealthScore()
		if newScore >= score {
			t.Errorf("Expected health score to decrease with high packet loss, got %f (was %f)", newScore, score)
		}
	})

	t.Run("AdaptiveIntervalWithPacketLoss", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Simulate network with packet loss
		for i := 0; i < 5; i++ {
			hm.RecordLatency(20 * time.Millisecond)
			hm.RecordPacketLoss(20.0) // 20% packet loss
		}

		// Interval should be more aggressive with packet loss
		interval := hm.GetInterval()
		opts := cfg.Options()
		minInterval := time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second

		// With packet loss, interval should be closer to minimum
		if interval > minInterval*3 {
			t.Errorf("Expected interval to be more aggressive with packet loss, got %v, min is %v", interval, minInterval)
		}
	})

	t.Run("AdaptiveIntervalRecoveryFromPacketLoss", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// First simulate network with packet loss
		for i := 0; i < 5; i++ {
			hm.RecordLatency(20 * time.Millisecond)
			hm.RecordPacketLoss(50.0) // 50% packet loss
		}

		unstableInterval := hm.GetInterval()

		// Then simulate recovery to stable network
		for i := 0; i < 10; i++ {
			hm.RecordLatency(10 * time.Millisecond)
			hm.RecordPacketLoss(0.0) // No packet loss
		}

		recoveredInterval := hm.GetInterval()
		opts := cfg.Options()
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second

		// Interval should increase after recovery
		if recoveredInterval <= unstableInterval {
			t.Errorf("Expected interval to increase after recovery, got %v (was %v)", recoveredInterval, unstableInterval)
		}

		// Should approach the maximum interval
		if recoveredInterval < maxInterval/2 {
			t.Errorf("Expected recovered interval to be closer to max, got %v, max is %v", recoveredInterval, maxInterval)
		}
	})

	t.Run("PacketLossBounds", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		opts := cfg.Options()
		minInterval := time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second

		// Test with 100% packet loss (worst case)
		for i := 0; i < 20; i++ {
			hm.RecordLatency(1000 * time.Millisecond) // Bad latency
			hm.RecordPacketLoss(100.0)              // 100% packet loss
		}

		// Interval should be closer to minimum with 100% packet loss and bad latency
		if hm.GetInterval() > minInterval*3 {
			t.Errorf("Interval should be closer to minimum with 100%% packet loss and bad latency, got %v, min is %v", hm.GetInterval(), minInterval)
		}

		// Test with 0% packet loss (best case)
		for i := 0; i < 20; i++ {
			hm.RecordLatency(10 * time.Millisecond) // Very good latency
			hm.RecordPacketLoss(0.0)               // No packet loss
		}

		// Interval should approach maximum with 0% packet loss
		if hm.GetInterval() < maxInterval/2 {
			t.Errorf("Interval should be closer to maximum with 0%% packet loss, got %v, max is %v", hm.GetInterval(), maxInterval)
		}
	})
}

func TestNormalizePacketLoss(t *testing.T) {
	cfg := createTestConfig()

	t.Run("NormalizePacketLoss", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")

		// Test with no packet loss (should score high)
		hm.RecordPacketLoss(0.0)
		score1 := hm.GetHealthScore()

		// Test with high packet loss (should score low)
		hm.RecordPacketLoss(50.0)
		score2 := hm.GetHealthScore()

		if score1 <= score2 {
			t.Errorf("Low packet loss should score higher than high packet loss, got %f and %f", score1, score2)
		}
	})
}

// This file uses the createTestConfig function defined in health_monitor_test.go