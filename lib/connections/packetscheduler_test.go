// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"

	"github.com/syncthing/syncthing/lib/protocol"
)

// TestPacketScheduler tests the basic functionality of the PacketScheduler
func TestPacketScheduler(t *testing.T) {
	// Given a packet scheduler
	scheduler := NewPacketScheduler()

	// When we create it
	if scheduler == nil {
		t.Error("Expected packet scheduler to be created, but it was nil")
	}

	// Then it should have default values
	deviceID := protocol.LocalDeviceID
	count := scheduler.GetConnectionCount(deviceID)
	if count != 0 {
		t.Errorf("Expected 0 connections initially, got %d", count)
	}
}

// TestPacketSchedulerAddRemove tests adding and removing connections
func TestPacketSchedulerAddRemove(t *testing.T) {
	// Given a packet scheduler and connections
	scheduler := NewPacketScheduler()
	deviceID := protocol.LocalDeviceID

	conn1 := NewEnhancedMockConnection("conn1", deviceID, 10, 90.0)
	conn2 := NewEnhancedMockConnection("conn2", deviceID, 20, 80.0)

	// When we add connections
	scheduler.AddConnection(deviceID, conn1)
	scheduler.AddConnection(deviceID, conn2)

	// Then we should have 2 connections
	count := scheduler.GetConnectionCount(deviceID)
	if count != 2 {
		t.Errorf("Expected 2 connections after adding, got %d", count)
	}

	// When we remove one connection
	scheduler.RemoveConnection(deviceID, "conn1")

	// Then we should have 1 connection
	count = scheduler.GetConnectionCount(deviceID)
	if count != 1 {
		t.Errorf("Expected 1 connection after removing one, got %d", count)
	}

	// And it should be the right connection
	conns := scheduler.GetConnections(deviceID)
	if len(conns) != 1 || conns[0].ConnectionID() != "conn2" {
		t.Errorf("Expected conn2 to remain, got %v", conns)
	}
}

// TestPacketSchedulerSelectConnection tests selecting the best connection
func TestPacketSchedulerSelectConnection(t *testing.T) {
	// Given a packet scheduler with connections of different health scores
	scheduler := NewPacketScheduler()
	deviceID := protocol.LocalDeviceID

	// Connection 1: Low health
	conn1 := NewEnhancedMockConnection("conn1", deviceID, 10, 30.0)

	// Connection 2: High health (should be selected)
	conn2 := NewEnhancedMockConnection("conn2", deviceID, 20, 90.0)

	// Connection 3: Medium health
	conn3 := NewEnhancedMockConnection("conn3", deviceID, 15, 60.0)

	scheduler.AddConnection(deviceID, conn1)
	scheduler.AddConnection(deviceID, conn2)
	scheduler.AddConnection(deviceID, conn3)

	// When we select the best connection
	selected := scheduler.SelectConnection(deviceID)

	// Then it should be the connection with the highest health score
	if selected != conn2 {
		t.Errorf("Expected conn2 to be selected (highest health), got %v", selected)
	}
}

// TestPacketSchedulerLoadBalancing tests load balancing selection
func TestPacketSchedulerLoadBalancing(t *testing.T) {
	// Given a packet scheduler with equally healthy connections
	scheduler := NewPacketScheduler()
	deviceID := protocol.LocalDeviceID

	// Three connections with equal health scores
	conn1 := NewEnhancedMockConnection("conn1", deviceID, 10, 80.0)
	conn2 := NewEnhancedMockConnection("conn2", deviceID, 10, 80.0)
	conn3 := NewEnhancedMockConnection("conn3", deviceID, 10, 80.0)

	scheduler.AddConnection(deviceID, conn1)
	scheduler.AddConnection(deviceID, conn2)
	scheduler.AddConnection(deviceID, conn3)

	// When we select connections for load balancing multiple times
	distribution := make(map[string]int)
	for i := 0; i < 300; i++ {
		selected := scheduler.SelectConnectionForLoadBalancing(deviceID)
		if selected != nil {
			distribution[selected.ConnectionID()]++
		}
	}

	// Then each connection should get approximately the same number of selections
	// With 300 selections and 3 connections, each should get around 100
	expected := 100
	tolerance := 50 // Allow 50% variance due to randomness

	for connID, count := range distribution {
		if count < expected-tolerance || count > expected+tolerance {
			t.Errorf("Connection %s was selected %d times, expected around %d (±%d)", connID, count, expected, tolerance)
		}
	}

	// Verify all three connections were selected
	if len(distribution) != 3 {
		t.Errorf("Expected all 3 connections to be selected, got %d", len(distribution))
	}
}

// TestPacketSchedulerNoConnections tests behavior with no connections
func TestPacketSchedulerNoConnections(t *testing.T) {
	// Given a packet scheduler with no connections
	scheduler := NewPacketScheduler()
	deviceID := protocol.LocalDeviceID

	// When we try to select a connection
	selected := scheduler.SelectConnection(deviceID)

	// Then it should return nil
	if selected != nil {
		t.Errorf("Expected nil when no connections available, got %v", selected)
	}

	// And when we try load balancing selection
	selected = scheduler.SelectConnectionForLoadBalancing(deviceID)

	// Then it should also return nil
	if selected != nil {
		t.Errorf("Expected nil when no connections available for load balancing, got %v", selected)
	}
}

// TestPacketSchedulerSingleConnection tests behavior with only one connection
func TestPacketSchedulerSingleConnection(t *testing.T) {
	// Given a packet scheduler with one connection
	scheduler := NewPacketScheduler()
	deviceID := protocol.LocalDeviceID

	conn := NewEnhancedMockConnection("single-conn", deviceID, 10, 75.0)
	scheduler.AddConnection(deviceID, conn)

	// When we select a connection
	selected := scheduler.SelectConnection(deviceID)

	// Then it should return that connection
	if selected != conn {
		t.Errorf("Expected single connection to be returned, got %v", selected)
	}

	// And when we select for load balancing
	selected = scheduler.SelectConnectionForLoadBalancing(deviceID)

	// Then it should also return that connection
	if selected != conn {
		t.Errorf("Expected single connection to be returned for load balancing, got %v", selected)
	}
}
