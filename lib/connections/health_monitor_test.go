// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

func TestHealthMonitor(t *testing.T) {
	// Create a mock config wrapper
	cfg := createTestConfig()

	t.Run("NewHealthMonitor", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		
		if hm == nil {
			t.Fatal("NewHealthMonitor should not return nil")
		}
		
		// Check initial interval is set to max
		opts := cfg.Options()
		expectedInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		if hm.GetInterval() != expectedInterval {
			t.Errorf("Expected initial interval to be %v, got %v", expectedInterval, hm.GetInterval())
		}
		
		// Check initial health score
		if hm.GetHealthScore() != 50.0 {
			t.Errorf("Expected initial health score to be 50.0, got %f", hm.GetHealthScore())
		}
	})

	t.Run("RecordLatency", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		
		// Record a stable latency
		hm.RecordLatency(20 * time.Millisecond)
		
		// Health score should improve with stable latency
		score := hm.GetHealthScore()
		if score <= 50.0 {
			t.Errorf("Expected health score to improve with stable latency, got %f", score)
		}
		
		// Record an unstable latency
		hm.RecordLatency(600 * time.Millisecond)
		
		// Health score should decrease with unstable latency
		newScore := hm.GetHealthScore()
		if newScore >= score {
			t.Errorf("Expected health score to decrease with unstable latency, got %f (was %f)", newScore, score)
		}
	})

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

	t.Run("AdaptiveIntervalStableNetwork", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		
		// Simulate stable network conditions
		for i := 0; i < 5; i++ {
			hm.RecordLatency(20 * time.Millisecond)
			hm.RecordPacketLoss(0.0)
		}
		
		// Interval should approach the maximum (less aggressive)
		interval := hm.GetInterval()
		opts := cfg.Options()
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		
		// Allow some tolerance since it may not reach the exact max immediately
		if interval < maxInterval/2 {
			t.Errorf("Expected interval to be closer to max for stable network, got %v, max is %v", interval, maxInterval)
		}
	})

	t.Run("AdaptiveIntervalUnstableNetwork", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		
		// Simulate unstable network conditions
		for i := 0; i < 5; i++ {
			hm.RecordLatency(600 * time.Millisecond)
			hm.RecordPacketLoss(20.0)
		}
		
		// Interval should approach the minimum (more aggressive)
		interval := hm.GetInterval()
		opts := cfg.Options()
		minInterval := time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second
		
		// Allow some tolerance since it may not reach the exact min immediately
		if interval > minInterval*3 {
			t.Errorf("Expected interval to be closer to min for unstable network, got %v, min is %v", interval, minInterval)
		}
	})

	t.Run("AdaptiveIntervalRecovery", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		
		// First simulate unstable network
		for i := 0; i < 5; i++ {
			hm.RecordLatency(600 * time.Millisecond)
			hm.RecordPacketLoss(20.0)
		}
		
		unstableInterval := hm.GetInterval()
		
		// Then simulate recovery to stable network
		for i := 0; i < 10; i++ {
			hm.RecordLatency(20 * time.Millisecond)
			hm.RecordPacketLoss(0.0)
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

	t.Run("IntervalBounds", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		opts := cfg.Options()
		minInterval := time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		
		// Test that interval never goes below minimum
		for i := 0; i < 20; i++ {
			hm.RecordLatency(1000 * time.Millisecond) // Very bad latency
			hm.RecordPacketLoss(100.0) // 100% packet loss
		}
		
		if hm.GetInterval() < minInterval {
			t.Errorf("Interval should not go below minimum, got %v, min is %v", hm.GetInterval(), minInterval)
		}
		
		// Test that interval never goes above maximum
		for i := 0; i < 20; i++ {
			hm.RecordLatency(1 * time.Millisecond) // Very good latency
			hm.RecordPacketLoss(0.0) // No packet loss
		}
		
		if hm.GetInterval() > maxInterval {
			t.Errorf("Interval should not go above maximum, got %v, max is %v", hm.GetInterval(), maxInterval)
		}
	})
}

func TestHealthScoreCalculation(t *testing.T) {
	cfg := createTestConfig()
	
	t.Run("NormalizeLatency", func(t *testing.T) {
		hm := NewHealthMonitor(cfg, "device1")
		
		// Test with very low latency (should score high)
		hm.RecordLatency(1 * time.Millisecond)
		score1 := hm.GetHealthScore()
		
		// Test with high latency (should score low)
		hm.RecordLatency(1000 * time.Millisecond)
		score2 := hm.GetHealthScore()
		
		if score1 <= score2 {
			t.Errorf("Low latency should score higher than high latency, got %f and %f", score1, score2)
		}
	})
	
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

func TestDebugStableNetwork(t *testing.T) {
	cfg := createTestConfig()
	hm := NewHealthMonitor(cfg, "device1")
	
	// Test with good network conditions
	hm.RecordLatency(20 * time.Millisecond)
	t.Logf("After good latency (20ms): Health score = %f, Interval = %v", hm.GetHealthScore(), hm.GetInterval())
	
	hm.RecordPacketLoss(0.0)
	t.Logf("After no packet loss: Health score = %f, Interval = %v", hm.GetHealthScore(), hm.GetInterval())
	
	// Test with multiple good measurements
	for i := 0; i < 5; i++ {
		hm.RecordLatency(20 * time.Millisecond)
		hm.RecordPacketLoss(0.0)
		t.Logf("After %d good measurements: Health score = %f, Interval = %v", i+1, hm.GetHealthScore(), hm.GetInterval())
	}
}

// createTestConfig creates a mock config wrapper for testing
func createTestConfig() config.Wrapper {
	// For testing purposes, we'll create a simple mock
	// In a real implementation, we would need to properly mock the config wrapper
	return config.Wrap("/tmp/test-config.xml", config.New(protocol.EmptyDeviceID), protocol.EmptyDeviceID, nil)
}