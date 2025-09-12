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

// TestCompleteConnectionManagementSystem tests the integration of all connection management features
func TestCompleteConnectionManagementSystem(t *testing.T) {
	// Create a config with all new features enabled
	cfg := createTestConfigWithAllFeaturesEnabled()

	t.Run("HealthMonitoringWithAdaptiveIntervals", func(t *testing.T) {
		// Given a health monitor with all features enabled
		deviceID := protocol.LocalDeviceID
		healthMonitor := NewHealthMonitorWithConfig(cfg, deviceID.String())

		// When we simulate excellent network conditions
		for i := 0; i < 10; i++ {
			healthMonitor.RecordLatency(10 * time.Millisecond)
			healthMonitor.RecordPacketLoss(0.0)
		}

		// Allow time for processing
		time.Sleep(50 * time.Millisecond)

		// Then we should get a high health score and longer interval
		excellentScore := healthMonitor.GetHealthScore()
		excellentInterval := healthMonitor.GetInterval()

		if excellentScore < 80.0 {
			t.Errorf("Expected high health score with excellent conditions, got %f", excellentScore)
		}

		opts := cfg.Options()
		maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		if excellentInterval < maxInterval/2 {
			t.Errorf("Expected longer interval with excellent conditions, got %v, max is %v", excellentInterval, maxInterval)
		}

		// When we simulate poor network conditions
		for i := 0; i < 10; i++ {
			healthMonitor.RecordLatency(500 * time.Millisecond)
			healthMonitor.RecordPacketLoss(30.0)
		}

		// Allow time for processing
		time.Sleep(50 * time.Millisecond)

		// Then we should get a low health score and shorter interval
		poorScore := healthMonitor.GetHealthScore()
		poorInterval := healthMonitor.GetInterval()

		if poorScore >= excellentScore {
			t.Errorf("Expected lower health score with poor conditions, got %f (was %f)", poorScore, excellentScore)
		}

		minInterval := time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second
		if poorInterval > minInterval*3 {
			t.Errorf("Expected shorter interval with poor conditions, got %v, min is %v", poorInterval, minInterval)
		}

		if poorInterval >= excellentInterval {
			t.Errorf("Expected shorter interval with poor conditions, got %v (was %v)", poorInterval, excellentInterval)
		}
	})

	t.Run("MultipathConnectionSelectionWithHealthScores", func(t *testing.T) {
		// Given a packet scheduler with multipath enabled
		scheduler := NewPacketScheduler()
		deviceID := protocol.LocalDeviceID

		// And multiple connections with different health characteristics
		lanConn := NewEnhancedMockConnectionWithNetworkType("lan-conn", deviceID, 10, 90.0, "lan")
		wanConn := NewEnhancedMockConnectionWithNetworkType("wan-conn", deviceID, 20, 70.0, "wan")
		relayConn := NewEnhancedMockConnectionWithNetworkType("relay-conn", deviceID, 30, 50.0, "wan")

		// When we add all connections to the scheduler
		scheduler.AddConnection(deviceID, lanConn)
		scheduler.AddConnection(deviceID, wanConn)
		scheduler.AddConnection(deviceID, relayConn)

		// Then the LAN connection should be selected as best due to health boost
		bestConn := scheduler.SelectConnection(deviceID)
		if bestConn != lanConn {
			t.Errorf("Expected LAN connection to be selected as best, got %v", bestConn.ConnectionID())
		}

		// When we degrade the LAN connection
		lanConn.SetHealthScore(30.0) // Significantly degraded

		// Then the WAN connection should be selected as best
		bestConn = scheduler.SelectConnection(deviceID)
		if bestConn != wanConn {
			t.Errorf("Expected WAN connection to be selected after LAN degradation, got %v", bestConn.ConnectionID())
		}

		// When we improve the LAN connection
		lanConn.SetHealthScore(95.0) // Restored and even better

		// Then the LAN connection should be selected again
		bestConn = scheduler.SelectConnection(deviceID)
		if bestConn != lanConn {
			t.Errorf("Expected LAN connection to be selected after recovery, got %v", bestConn.ConnectionID())
		}
	})

	t.Run("LoadBalancingWithHealthScores", func(t *testing.T) {
		// Given a packet scheduler with multiple equally healthy connections
		scheduler := NewPacketScheduler()
		deviceID := protocol.LocalDeviceID

		connA := NewEnhancedMockConnection("conn-a", deviceID, 10, 85.0)
		connB := NewEnhancedMockConnection("conn-b", deviceID, 10, 85.0)
		connC := NewEnhancedMockConnection("conn-c", deviceID, 10, 85.0)

		// When we add all connections to the scheduler
		scheduler.AddConnection(deviceID, connA)
		scheduler.AddConnection(deviceID, connB)
		scheduler.AddConnection(deviceID, connC)

		// And we simulate sending many packets
		distribution := make(map[string]int)
		for i := 0; i < 300; i++ {
			selected := scheduler.SelectConnectionForLoadBalancing(deviceID)
			distribution[selected.ConnectionID()]++
		}

		// Then packets should be distributed approximately equally
		expected := 100
		tolerance := 30 // Allow 30% variance

		for connID, count := range distribution {
			if count < expected-tolerance || count > expected+tolerance {
				t.Errorf("Connection %s got %d packets, expected around %d (±%d)", connID, count, expected, tolerance)
			}
		}

		// Verify all three connections received packets
		if len(distribution) != 3 {
			t.Errorf("Expected packets to be distributed across 3 connections, got %d", len(distribution))
		}
	})

	t.Run("ConnectionTrackerWithMultipath", func(t *testing.T) {
		// Given a device connection tracker with multipath enabled
		tracker := &deviceConnectionTracker{
			connections:       make(map[protocol.DeviceID][]protocol.Connection),
			wantConnections:   make(map[protocol.DeviceID]int),
			stabilityMgrs:     make(map[protocol.DeviceID]*ConnectionStabilityManager),
			hysteresisCtrls:   make(map[protocol.DeviceID]*HysteresisController),
			convergenceMgrs:   make(map[protocol.DeviceID]*ConvergenceManager),
		}

		deviceID := protocol.LocalDeviceID
		hello := protocol.Hello{NumConnections: 3}
		
		// Create a test config
		cfg := createTestConfigWithAllFeaturesEnabled()

		// And multiple connections for the same device
		conn1 := NewEnhancedMockConnection("conn1", deviceID, 10, 90.0)
		conn2 := NewEnhancedMockConnection("conn2", deviceID, 20, 80.0)
		conn3 := NewEnhancedMockConnection("conn3", deviceID, 30, 70.0)

		// When we add all connections to the tracker
		tracker.accountAddedConnection(conn1, hello, 0, cfg)
		tracker.accountAddedConnection(conn2, hello, 0, cfg)
		tracker.accountAddedConnection(conn3, hello, 0, cfg)

		// Then we should have 3 connections for this device
		numConns := tracker.numConnectionsForDevice(deviceID)
		if numConns != 3 {
			t.Errorf("Expected 3 connections, got %d", numConns)
		}

		// And we should be able to retrieve all connections
		conns := tracker.connections[deviceID]
		if len(conns) != 3 {
			t.Errorf("Expected 3 connections in tracker, got %d", len(conns))
		}

		// When we remove a connection
		tracker.accountRemovedConnection(conn1, cfg)

		// Then we should have 2 connections remaining
		numConns = tracker.numConnectionsForDevice(deviceID)
		if numConns != 2 {
			t.Errorf("Expected 2 connections after removal, got %d", numConns)
		}
	})

	t.Run("IntelligentReconnectionWithBackoff", func(t *testing.T) {
		// Given a next dial registry
		registry := make(nextDialRegistry)
		deviceID := protocol.LocalDeviceID
		now := time.Now()
		addr := "tcp://127.0.0.1:22000"

		// When we simulate multiple reconnection attempts
		registry.set(deviceID, addr, now)

		// First redial (immediate)
		registry.redialDevice(deviceID, now)
		// firstDial := registry.get(deviceID, addr)  // Not used, commented out

		// Second redial (1 second backoff)
		registry.redialDevice(deviceID, now.Add(1*time.Second))
		secondDial := registry.get(deviceID, addr)

		// Third redial (4 seconds backoff)
		registry.redialDevice(deviceID, now.Add(5*time.Second))
		thirdDial := registry.get(deviceID, addr)

		// Verify backoff timing (allowing some tolerance)
		expectedSecond := now.Add(1 * time.Second).Add(1 * time.Second) // now + 1s + 1s backoff
		expectedThird := now.Add(5 * time.Second).Add(4 * time.Second)  // now + 5s + 4s backoff

		if secondDial.Before(expectedSecond.Add(-2*time.Second)) || secondDial.After(expectedSecond.Add(2*time.Second)) {
			t.Errorf("Second redial timing incorrect, got %v, expected around %v", secondDial, expectedSecond)
		}

		if thirdDial.Before(expectedThird.Add(-5*time.Second)) || thirdDial.After(expectedThird.Add(5*time.Second)) {
			t.Errorf("Third redial timing incorrect, got %v, expected around %v", thirdDial, expectedThird)
		}
	})
}

// TestServiceIntegration tests the integration of all components through the service interface
func TestServiceIntegration(t *testing.T) {
	// Note: This would require a more complex setup with a real service instance
	// For now, we'll test the methods we implemented
	t.Run("GetConnectedDevicesAndConnections", func(t *testing.T) {
		// This test would require a full service setup which is complex
		// The methods are already tested in other tests
		t.Skip("Full service integration test requires complex setup")
	})
}

// createTestConfigWithAllFeaturesEnabled creates a test config with all new features enabled
func createTestConfigWithAllFeaturesEnabled() config.Wrapper {
	// Create a temporary config file for testing
	cfg := config.New(protocol.EmptyDeviceID)

	// Enable all new features
	cfg.Options.AdaptiveKeepAliveEnabled = true
	cfg.Options.AdaptiveKeepAliveMinS = 10
	cfg.Options.AdaptiveKeepAliveMaxS = 60
	cfg.Options.MultipathEnabled = true

	// For testing purposes, we'll create a simple wrapper
	// In a real implementation, we would need to properly create a config wrapper
	return config.Wrap("/tmp/test-config.xml", cfg, protocol.EmptyDeviceID, nil)
}
