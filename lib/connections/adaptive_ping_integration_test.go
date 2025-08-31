// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections_test

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections"
	"github.com/syncthing/syncthing/lib/protocol"
)

// TestAdaptivePingSender tests the adaptive ping sender functionality
func TestAdaptivePingSender(t *testing.T) {
	// Create mock config with adaptive keep-alive enabled
	cfg := createTestConfigWithAdaptiveKeepAlive(true)

	t.Run("Adaptive interval used when feature enabled", func(t *testing.T) {
		// Given a health monitor with adaptive keep-alive enabled
		healthMonitor := connections.NewHealthMonitor(cfg, "device1")

		// When we simulate stable network conditions
		for i := 0; i < 5; i++ {
			healthMonitor.RecordLatency(20 * time.Millisecond)
			healthMonitor.RecordPacketLoss(0.0)
		}

		// Allow time for health monitor to update
		time.Sleep(100 * time.Millisecond)

		// Then the interval should approach the maximum (less aggressive)
		stableInterval := healthMonitor.GetInterval()
		opts := cfg.Options()
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second

		if stableInterval < maxInterval/2 {
			t.Errorf("Expected interval to be closer to max for stable network, got %v, max is %v", stableInterval, maxInterval)
		}

		// When we simulate unstable network conditions
		for i := 0; i < 5; i++ {
			healthMonitor.RecordLatency(600 * time.Millisecond)
			healthMonitor.RecordPacketLoss(20.0)
		}

		// Allow time for health monitor to update
		time.Sleep(100 * time.Millisecond)

		// Then the interval should approach the minimum (more aggressive)
		unstableInterval := healthMonitor.GetInterval()
		minInterval := time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second

		if unstableInterval > minInterval*3 {
			t.Errorf("Expected interval to be closer to min for unstable network, got %v, min is %v", unstableInterval, minInterval)
		}

		// Verify that the interval changed based on network conditions
		if unstableInterval >= stableInterval {
			t.Errorf("Expected interval to decrease with unstable network, got %v (was %v)", unstableInterval, stableInterval)
		}
	})

	t.Run("Fixed behavior when feature disabled", func(t *testing.T) {
		// Given a config with adaptive keep-alive disabled
		cfgDisabled := createTestConfigWithAdaptiveKeepAlive(false)

		// When we create a health monitor
		healthMonitor := connections.NewHealthMonitor(cfgDisabled, "device1")

		// The health monitor still exists but won't be used by the protocol layer
		// This test verifies the health monitor functions correctly regardless of config

		initialInterval := healthMonitor.GetInterval()
		initialScore := healthMonitor.GetHealthScore()

		// Simulate network activity
		healthMonitor.RecordLatency(100 * time.Millisecond)
		healthMonitor.RecordPacketLoss(5.0)

		time.Sleep(50 * time.Millisecond)

		// The health monitor should still update its values
		newInterval := healthMonitor.GetInterval()
		newScore := healthMonitor.GetHealthScore()

		// Values should have changed
		if newInterval == initialInterval && newScore == initialScore {
			t.Error("Expected health monitor to update values even when feature is conceptually disabled")
		}
	})

	t.Run("Interval updates during connection lifetime", func(t *testing.T) {
		// Given a health monitor
		cfg := createTestConfigWithAdaptiveKeepAlive(true)
		healthMonitor := connections.NewHealthMonitor(cfg, "device1")

		// Record initial interval
		initialInterval := healthMonitor.GetInterval()

		// When we simulate network degradation
		healthMonitor.RecordLatency(1000 * time.Millisecond)
		healthMonitor.RecordPacketLoss(50.0)

		// Allow time for update
		time.Sleep(100 * time.Millisecond)

		degradedInterval := healthMonitor.GetInterval()

		// When we simulate network recovery
		for i := 0; i < 10; i++ {
			healthMonitor.RecordLatency(10 * time.Millisecond)
			healthMonitor.RecordPacketLoss(0.0)
		}

		// Allow time for update
		time.Sleep(100 * time.Millisecond)

		recoveredInterval := healthMonitor.GetInterval()

		// Then intervals should change appropriately
		if degradedInterval >= initialInterval {
			t.Errorf("Expected interval to decrease with degraded network, got %v (was %v)", degradedInterval, initialInterval)
		}

		if recoveredInterval <= degradedInterval {
			t.Errorf("Expected interval to increase with recovered network, got %v (was %v)", recoveredInterval, degradedInterval)
		}

		// Final interval should be closer to max than degraded interval
		opts := cfg.Options()
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		if recoveredInterval < maxInterval/2 {
			t.Errorf("Expected recovered interval to be closer to max, got %v, max is %v", recoveredInterval, maxInterval)
		}
	})
}

// TestLatencyMeasurement tests that latency measurements are properly recorded
func TestLatencyMeasurement(t *testing.T) {
	cfg := createTestConfigWithAdaptiveKeepAlive(true)
	healthMonitor := connections.NewHealthMonitor(cfg, "device1")

	// Record initial health score
	initialScore := healthMonitor.GetHealthScore()

	// When we record low latency
	healthMonitor.RecordLatency(50 * time.Millisecond)

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	newScore := healthMonitor.GetHealthScore()

	// Then score should improve with better latency
	if newScore <= initialScore {
		t.Errorf("Expected health score to improve with better latency, got %f (was %f)", newScore, initialScore)
	}

	// When we record high latency
	healthMonitor.RecordLatency(800 * time.Millisecond)

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	worseScore := healthMonitor.GetHealthScore()

	// Then score should decrease with worse latency
	if worseScore >= newScore {
		t.Errorf("Expected health score to decrease with worse latency, got %f (was %f)", worseScore, newScore)
	}
}

// TestPacketLossMeasurement tests that packet loss measurements are properly recorded
func TestPacketLossMeasurement(t *testing.T) {
	cfg := createTestConfigWithAdaptiveKeepAlive(true)
	healthMonitor := connections.NewHealthMonitor(cfg, "device1")

	// Record initial health score
	initialScore := healthMonitor.GetHealthScore()

	// When we record no packet loss
	healthMonitor.RecordPacketLoss(0.0)

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	newScore := healthMonitor.GetHealthScore()

	// Then score should improve with no packet loss
	if newScore <= initialScore {
		t.Errorf("Expected health score to improve with no packet loss, got %f (was %f)", newScore, initialScore)
	}

	// When we record high packet loss
	healthMonitor.RecordPacketLoss(50.0)

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	worseScore := healthMonitor.GetHealthScore()

	// Then score should decrease with high packet loss
	if worseScore >= newScore {
		t.Errorf("Expected health score to decrease with high packet loss, got %f (was %f)", worseScore, newScore)
	}
}

// createTestConfigWithAdaptiveKeepAlive creates a test config with adaptive keep-alive settings
func createTestConfigWithAdaptiveKeepAlive(enabled bool) config.Wrapper {
	// Create a temporary config file for testing
	cfg := config.New(protocol.EmptyDeviceID)

	// Set adaptive keep-alive options
	cfg.Options.AdaptiveKeepAliveEnabled = enabled
	cfg.Options.AdaptiveKeepAliveMinS = 20
	cfg.Options.AdaptiveKeepAliveMaxS = 120

	// For testing purposes, we'll create a simple wrapper
	// In a real implementation, we would need to properly create a config wrapper
	return config.Wrap("/tmp/test-config.xml", cfg, protocol.EmptyDeviceID, nil)
}
