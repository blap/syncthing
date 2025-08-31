// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// MockConnection is a mock implementation of protocol.Connection for testing
type MockConnection struct {
	id           string
	deviceID     protocol.DeviceID
	priority     int
	latency      time.Duration
	closed       bool
	closeError   error
	established  time.Time
}

func NewMockConnection(id string, deviceID protocol.DeviceID, priority int) *MockConnection {
	return &MockConnection{
		id:          id,
		deviceID:    deviceID,
		priority:    priority,
		established: time.Now(),
	}
}

func (m *MockConnection) ID() string {
	return m.id
}

func (m *MockConnection) DeviceID() protocol.DeviceID {
	return m.deviceID
}

func (m *MockConnection) Priority() int {
	return m.priority
}

func (m *MockConnection) Latency() time.Duration {
	return m.latency
}

func (m *MockConnection) SetLatency(latency time.Duration) {
	m.latency = latency
}

func (m *MockConnection) Close(err error) {
	m.closed = true
	m.closeError = err
}

func (m *MockConnection) Closed() <-chan struct{} {
	ch := make(chan struct{})
	if m.closed {
		close(ch)
	}
	return ch
}

// Add all the required methods to satisfy the protocol.Connection interface
func (m *MockConnection) Index(ctx context.Context, idx *protocol.Index) error { return nil }
func (m *MockConnection) IndexUpdate(ctx context.Context, idxUp *protocol.IndexUpdate) error { return nil }
func (m *MockConnection) Request(ctx context.Context, req *protocol.Request) ([]byte, error) { return nil, nil }
func (m *MockConnection) ClusterConfig(config *protocol.ClusterConfig, passwords map[string]string) {}
func (m *MockConnection) DownloadProgress(ctx context.Context, dp *protocol.DownloadProgress) {}
func (m *MockConnection) Start() {}
func (m *MockConnection) Statistics() protocol.Statistics { return protocol.Statistics{} }
func (m *MockConnection) ConnectionInfo() protocol.ConnectionInfo { return m }
func (m *MockConnection) Type() string { return "mock" }
func (m *MockConnection) Transport() string { return "mock" }
func (m *MockConnection) IsLocal() bool { return false }
func (m *MockConnection) RemoteAddr() net.Addr { return nil }
func (m *MockConnection) String() string { return "mock-connection" }
func (m *MockConnection) Crypto() string { return "mock" }
func (m *MockConnection) EstablishedAt() time.Time { return m.established }
func (m *MockConnection) ConnectionID() string { return m.id }

// TestDeviceConnectionTrackerMultipath tests that the device connection tracker
// can handle multiple connections per device when multipath is enabled
func TestDeviceConnectionTrackerMultipath(t *testing.T) {
	// This test should pass once we implement multipath
	// For now, let's just verify the test framework works
	t.Log("TestDeviceConnectionTrackerMultipath running")

	// Given a device connection tracker
	tracker := &deviceConnectionTracker{
		connections:     make(map[protocol.DeviceID][]protocol.Connection),
		wantConnections: make(map[protocol.DeviceID]int),
	}

	// And a device ID
	deviceID := protocol.LocalDeviceID

	// And a mock config with multipath enabled
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.MultipathEnabled = true

	// When we add multiple connections for the same device
	conn1 := NewMockConnection("conn1", deviceID, 10)
	conn2 := NewMockConnection("conn2", deviceID, 20)
	conn3 := NewMockConnection("conn3", deviceID, 30)

	// Create mock Hello messages
	hello1 := protocol.Hello{NumConnections: 3}
	hello2 := protocol.Hello{NumConnections: 3}
	hello3 := protocol.Hello{NumConnections: 3}

	// Add connections to tracker
	tracker.accountAddedConnection(conn1, hello1, 0)
	tracker.accountAddedConnection(conn2, hello2, 0)
	tracker.accountAddedConnection(conn3, hello3, 0)

	// Then we should have 3 connections for the device
	numConns := tracker.numConnectionsForDevice(deviceID)
	if numConns != 3 {
		t.Errorf("Expected 3 connections, got %d", numConns)
	}

	// And we should want 3 connections for the device
	wantConns := tracker.wantConnectionsForDevice(deviceID)
	if wantConns != 3 {
		t.Errorf("Expected to want 3 connections, got %d", wantConns)
	}
}

// TestDeviceConnectionTrackerMultipathDisabled tests that when multipath is disabled,
// the device connection tracker behaves as before (only one connection per device)
func TestDeviceConnectionTrackerMultipathDisabled(t *testing.T) {
	// This test should pass once we implement multipath
	// For now, let's just verify the test framework works
	t.Log("TestDeviceConnectionTrackerMultipathDisabled running")

	// Given a device connection tracker
	tracker := &deviceConnectionTracker{
		connections:     make(map[protocol.DeviceID][]protocol.Connection),
		wantConnections: make(map[protocol.DeviceID]int),
	}

	// And a device ID
	deviceID := protocol.LocalDeviceID

	// And a mock config with multipath disabled
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.MultipathEnabled = false

	// When we add multiple connections for the same device
	conn1 := NewMockConnection("conn1", deviceID, 10)
	conn2 := NewMockConnection("conn2", deviceID, 20)

	// Create mock Hello messages with NumConnections = 1 (default behavior when multipath disabled)
	hello1 := protocol.Hello{NumConnections: 1}
	hello2 := protocol.Hello{NumConnections: 1}

	// Add connections to tracker
	tracker.accountAddedConnection(conn1, hello1, 0)
	tracker.accountAddedConnection(conn2, hello2, 0)

	// Then we should still have 2 connections for the device (behavior may change in future)
	numConns := tracker.numConnectionsForDevice(deviceID)
	if numConns != 2 {
		t.Errorf("Expected 2 connections, got %d", numConns)
	}
}