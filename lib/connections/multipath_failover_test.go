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

// selectBestConnection selects the best connection based on health scores
func selectBestConnection(connections []protocol.Connection) protocol.Connection {
	if len(connections) == 0 {
		return nil
	}

	// Use the same logic as in PacketScheduler
	scheduler := NewPacketScheduler()
	return scheduler.selectBestConnection(connections)
}

// selectConnectionForLoadBalancing selects a connection for load balancing
// based on health scores and selection history
func selectConnectionForLoadBalancing(connections []protocol.Connection, seed int) protocol.Connection {
	if len(connections) == 0 {
		return nil
	}

	// Use the same logic as in PacketScheduler but with a fixed seed for testing
	scheduler := NewPacketScheduler()
	// For testing purposes, we'll use a fixed seed to make tests deterministic
	scheduler.randSource.Seed(int64(seed))
	return scheduler.selectConnectionWeighted(connections)
}

// TestMultipathFailoverScenario tests a complete multipath failover scenario
func TestMultipathFailoverScenario(t *testing.T) {
	// Given a device with multiple network paths
	deviceID := protocol.LocalDeviceID
	
	// Connection 1: LAN connection (healthy, low latency)
	lanConn := NewEnhancedMockConnection("lan-conn", deviceID, 10, 95.0)
	lanConn.SetLatency(10 * time.Millisecond)
	
	// Connection 2: Relay connection (slower, higher latency)
	relayConn := NewEnhancedMockConnection("relay-conn", deviceID, 30, 70.0)
	relayConn.SetLatency(100 * time.Millisecond)
	
	// Connection 3: WiFi connection (medium health)
	wifiConn := NewEnhancedMockConnection("wifi-conn", deviceID, 20, 80.0)
	wifiConn.SetLatency(50 * time.Millisecond)
	
	// A tracker to manage these connections
	tracker := &deviceConnectionTracker{
		connections:     make(map[protocol.DeviceID][]protocol.Connection),
		wantConnections: make(map[protocol.DeviceID]int),
	}
	
	// Config with multipath enabled
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.MultipathEnabled = true
	
	// When we add all connections to the tracker
	hello := protocol.Hello{NumConnections: 3}
	tracker.accountAddedConnection(lanConn, hello, 0)
	tracker.accountAddedConnection(relayConn, hello, 0)
	tracker.accountAddedConnection(wifiConn, hello, 0)
	
	// Initially, LAN should be selected as the best connection
	connections := tracker.connections[deviceID]
	bestConn := selectBestConnection(connections)
	if bestConn != lanConn {
		t.Errorf("Expected LAN connection to be selected initially, got %v", bestConn)
	}
	
	// When LAN connection fails (simulate by setting health to 0)
	lanConn.SetHealthScore(0.0)
	
	// Then WiFi should be selected as the new best connection
	bestConn = selectBestConnection(connections)
	if bestConn != wifiConn {
		t.Errorf("Expected WiFi connection to be selected after LAN failed, got %v", bestConn)
	}
	
	// When WiFi also degrades (simulate by reducing health)
	wifiConn.SetHealthScore(40.0)
	
	// Then Relay should be selected as the new best connection
	bestConn = selectBestConnection(connections)
	if bestConn != relayConn {
		t.Errorf("Expected Relay connection to be selected after WiFi degraded, got %v", bestConn)
	}
	
	// When LAN recovers (simulate by restoring health)
	lanConn.SetHealthScore(95.0)
	
	// Then LAN should be selected again as the best connection
	bestConn = selectBestConnection(connections)
	if bestConn != lanConn {
		t.Errorf("Expected LAN connection to be selected after recovery, got %v", bestConn)
	}
}

// TestMultipathLoadBalancingScenario tests a load balancing scenario with multiple healthy paths
func TestMultipathLoadBalancingScenario(t *testing.T) {
	// Given a device with multiple equally healthy connections
	deviceID := protocol.LocalDeviceID
	
	// Connection 1: Path A (healthy)
	connA := NewEnhancedMockConnection("path-a", deviceID, 10, 90.0)
	
	// Connection 2: Path B (equally healthy)
	connB := NewEnhancedMockConnection("path-b", deviceID, 10, 90.0)
	
	// Connection 3: Path C (also healthy)
	connC := NewEnhancedMockConnection("path-c", deviceID, 10, 90.0)
	
	// A tracker to manage these connections
	tracker := &deviceConnectionTracker{
		connections:     make(map[protocol.DeviceID][]protocol.Connection),
		wantConnections: make(map[protocol.DeviceID]int),
	}
	
	// Config with multipath enabled
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.MultipathEnabled = true
	
	// When we add all connections to the tracker
	hello := protocol.Hello{NumConnections: 3}
	tracker.accountAddedConnection(connA, hello, 0)
	tracker.accountAddedConnection(connB, hello, 0)
	tracker.accountAddedConnection(connC, hello, 0)
	
	// And we simulate sending many packets
	connections := tracker.connections[deviceID]
	distribution := make(map[string]int)
	
	// Send 300 packets
	for i := 0; i < 300; i++ {
		selected := selectConnectionForLoadBalancing(connections, i)
		distribution[selected.ConnectionID()]++
	}
	
	// Then packets should be distributed approximately equally
	// With 300 packets and 3 connections, each should get around 100
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
}