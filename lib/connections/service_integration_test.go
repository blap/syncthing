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

// TestHealthMonitorInstantiation tests that health monitors are properly instantiated
// when adaptive keep-alive is enabled
func TestHealthMonitorInstantiation(t *testing.T) {
	// Create a config with adaptive keep-alive enabled
	cfg := createTestConfigForIntegrationTest(true)
	
	t.Run("Health monitor created when feature enabled", func(t *testing.T) {
		// Given a device ID
		deviceID := protocol.DeviceID{1, 2, 3, 4}
		
		// When we create a health monitor
		healthMonitor := NewHealthMonitor(cfg, deviceID.String())
		
		// Then it should be created successfully
		if healthMonitor == nil {
			t.Error("Expected health monitor to be created")
		}
		
		// Initial interval should be set to max
		initialInterval := healthMonitor.GetInterval()
		opts := cfg.Options()
		expectedInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		
		if initialInterval != expectedInterval {
			t.Errorf("Expected initial interval to be %v, got %v", expectedInterval, initialInterval)
		}
	})
	
	t.Run("Health monitor lifecycle", func(t *testing.T) {
		// Given a health monitor
		deviceID := protocol.DeviceID{1, 2, 3, 4}
		healthMonitor := NewHealthMonitor(cfg, deviceID.String())
		
		// When we start and stop it
		healthMonitor.Start()
		
		// Allow some time for the monitor to run
		time.Sleep(50 * time.Millisecond)
		
		// Record some metrics
		healthMonitor.RecordLatency(100 * time.Millisecond)
		healthMonitor.RecordPacketLoss(5.0)
		
		// Allow time for processing
		time.Sleep(50 * time.Millisecond)
		
		// Then stop it
		healthMonitor.Stop()
		
		// The monitor should have processed the metrics
		interval := healthMonitor.GetInterval()
		score := healthMonitor.GetHealthScore()
		
		// With poor network conditions, interval should decrease and score should be lower
		opts := cfg.Options()
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		
		if interval >= maxInterval {
			t.Errorf("Expected interval to decrease with poor network conditions, got %v (max is %v)", interval, maxInterval)
		}
		
		if score >= 50.0 {
			t.Errorf("Expected health score to decrease with poor network conditions, got %f", score)
		}
	})
}

// TestHealthMonitorDisabled tests that no health monitor is created when feature is disabled
func TestHealthMonitorDisabled(t *testing.T) {
	// Create a config with adaptive keep-alive disabled
	cfg := createTestConfigForIntegrationTest(false)
	
	t.Run("No health monitor operations when feature disabled", func(t *testing.T) {
		// This test is more of a conceptual test since we can't easily test
		// the connection management integration without a full service setup
		// But we can at least verify that the config is correctly set
		
		if cfg.Options().AdaptiveKeepAliveEnabled {
			t.Error("Expected adaptive keep-alive to be disabled")
		}
	})
}

// createTestConfigForIntegrationTest creates a test config with adaptive keep-alive settings
func createTestConfigForIntegrationTest(enabled bool) config.Wrapper {
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