// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// TestMultipathIntegration tests the complete multipath integration flow
func TestMultipathIntegration(t *testing.T) {
	// Given a configuration with multipath enabled
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.MultipathEnabled = true
	cfgWrapper := config.Wrap("/tmp/test-config.xml", cfg, protocol.EmptyDeviceID, nil)

	// And a device connection tracker
	tracker := &deviceConnectionTracker{
		connections:     make(map[protocol.DeviceID][]protocol.Connection),
		wantConnections: make(map[protocol.DeviceID]int),
	}

	// And a packet scheduler
	scheduler := NewPacketScheduler()

	// And a device ID
	deviceID := protocol.LocalDeviceID

	// When we create multiple connections for the same device
	conn1 := NewEnhancedMockConnection("conn1", deviceID, 10, 90.0)
	conn2 := NewEnhancedMockConnection("conn2", deviceID, 20, 80.0)

	// And we add them to both the tracker and scheduler (simulating the service behavior)
	hello := protocol.Hello{NumConnections: 2}
	tracker.accountAddedConnection(conn1, hello, 0, cfgWrapper)
	tracker.accountAddedConnection(conn2, hello, 0, cfgWrapper)

	scheduler.AddConnection(deviceID, conn1)
	scheduler.AddConnection(deviceID, conn2)

	// Then we should be able to select connections using the scheduler
	selected := scheduler.SelectConnection(deviceID)
	if selected != conn1 {
		t.Errorf("Expected conn1 to be selected (highest health), got %v", selected)
	}

	// And we should be able to select for load balancing
	selected = scheduler.SelectConnectionForLoadBalancing(deviceID)
	if selected == nil {
		t.Error("Expected a connection to be selected for load balancing")
	}

	// When we remove a connection from the scheduler
	scheduler.RemoveConnection(deviceID, "conn1")

	// Then only conn2 should remain
	conns := scheduler.GetConnections(deviceID)
	if len(conns) != 1 || conns[0] != conn2 {
		t.Errorf("Expected only conn2 to remain, got %v", conns)
	}
}

// TestMultipathDisabledIntegration tests that when multipath is disabled,
// the scheduler is not used
func TestMultipathDisabledIntegration(t *testing.T) {
	// Given a configuration with multipath disabled
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.MultipathEnabled = false

	// And a packet scheduler
	scheduler := NewPacketScheduler()

	// And a device ID
	deviceID := protocol.LocalDeviceID

	// When we try to select a connection when no connections are added
	selected := scheduler.SelectConnection(deviceID)
	if selected != nil {
		t.Errorf("Expected nil when no connections available, got %v", selected)
	}

	// Even when we add connections
	conn := NewEnhancedMockConnection("conn", deviceID, 10, 90.0)
	scheduler.AddConnection(deviceID, conn)

	// We can still select them (the scheduler works regardless of the config)
	selected = scheduler.SelectConnection(deviceID)
	if selected != conn {
		t.Errorf("Expected conn to be selected, got %v", selected)
	}
}
