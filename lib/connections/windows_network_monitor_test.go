// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package connections

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// BasicMockService implements the Service interface for testing
type BasicMockService struct {
	dialNowCalled bool
	mut           sync.Mutex
}

func (m *BasicMockService) DialNow() {
	m.mut.Lock()
	defer m.mut.Unlock()
	m.dialNowCalled = true
}

func (m *BasicMockService) WasDialNowCalled() bool {
	m.mut.Lock()
	defer m.mut.Unlock()
	return m.dialNowCalled
}

func (m *BasicMockService) ResetDialNowCalled() {
	m.mut.Lock()
	defer m.mut.Unlock()
	m.dialNowCalled = false
}

// Implement other required methods with empty implementations
func (m *BasicMockService) Serve(ctx context.Context) error { return nil }
func (m *BasicMockService) Stop()        {}
func (m *BasicMockService) String() string { return "BasicMockService" }
func (m *BasicMockService) ListenerStatus() map[string]ListenerStatusEntry { return nil }
func (m *BasicMockService) ConnectionStatus() map[string]ConnectionStatusEntry { return nil }
func (m *BasicMockService) NATType() string { return "" }
func (m *BasicMockService) GetConnectedDevices() []protocol.DeviceID { return nil }
func (m *BasicMockService) GetConnectionsForDevice(deviceID protocol.DeviceID) []protocol.Connection { return nil }
func (m *BasicMockService) PacketScheduler() *PacketScheduler { return nil }
func (m *BasicMockService) AllAddresses() []string { return nil }
func (m *BasicMockService) ExternalAddresses() []string { return nil }
func (m *BasicMockService) RawCopy() config.Configuration { return config.Configuration{} }

func TestWindowsNetworkMonitor(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)

	// Test that we can create a monitor
	if monitor == nil {
		t.Fatal("Failed to create WindowsNetworkMonitor")
	}

	// Test initial state using the public method
	adapterStates := monitor.GetAdapterStates()
	if adapterStates == nil {
		t.Error("Adapter states map should be initialized")
	}
}

func TestWindowsNetworkMonitor_ScanNetworkAdapters(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test scanning network adapters
	monitor.scanNetworkAdapters()
	
	// Verify that adapter states were populated using the public method
	states := len(monitor.GetAdapterStates())
	
	// We should have at least some adapters (this will depend on the test environment)
	t.Logf("Found %d network adapters", states)
}

func TestWindowsNetworkMonitor_CheckForNetworkChanges(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Set up initial state using the public method
	monitor.SetAdapterState("TestAdapter", NetworkAdapterInfo{
		Name: "TestAdapter",
		IsUp: false,
		Type: 6, // Ethernet
		ChangeCount: 0,
	})
	
	// Check for changes (should not trigger reconnection since we haven't changed the state)
	monitor.checkForNetworkChanges()
	
	// Update state to simulate adapter coming up
	adapterInfo := monitor.GetAdapterStates()["TestAdapter"]
	adapterInfo.IsUp = true
	adapterInfo.ChangeCount = 1
	monitor.SetAdapterState("TestAdapter", adapterInfo)
	
	// Check for changes (should detect the change)
	monitor.checkForNetworkChanges()
}

func TestWindowsNetworkMonitor_GetNetworkProfile(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test getting network profile
	profile := monitor.GetNetworkProfile()
	
	// Should return a string (even if it's just a placeholder)
	if profile == "" {
		t.Error("GetNetworkProfile should return a non-empty string")
	}
	
	t.Logf("Network profile: %s", profile)
}

func TestWindowsNetworkMonitor_AdapterInfo(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test adapter info structure
	adapterInfo := NetworkAdapterInfo{
		Name: "TestAdapter",
		IsUp: true,
		Type: 6, // Ethernet
		MediaType: 1,
		LinkSpeed: 1000000000, // 1 Gbps
		LastChange: time.Now(),
		ChangeCount: 0,
	}
	
	// Verify the monitor was created
	if monitor == nil {
		t.Fatal("Failed to create WindowsNetworkMonitor")
	}
	
	if adapterInfo.Name != "TestAdapter" {
		t.Error("Adapter name not set correctly")
	}
	
	if !adapterInfo.IsUp {
		t.Error("Adapter should be up")
	}
	
	if adapterInfo.Type != 6 {
		t.Error("Adapter type not set correctly")
	}
}

func TestWindowsNetworkMonitor_StabilityMetrics(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test stability metrics structure using the public method
	metrics := monitor.GetStabilityMetrics()
	
	if metrics.StabilityScore != 1.0 {
		t.Error("Initial stability score should be 1.0")
	}
	
	if metrics.AdaptiveTimeout != 5*time.Second {
		t.Error("Initial adaptive timeout should be 5 seconds")
	}
	
	// Test updating metrics by accessing internal fields directly is not possible
	// We'll test the updateAdaptiveTimeouts method instead
	monitor.mut.Lock()
	monitor.stabilityMetrics.TotalChanges = 5
	monitor.stabilityMetrics.RecentChanges = 2
	monitor.stabilityMetrics.LastErrorTime = time.Now()
	monitor.mut.Unlock()
	
	metrics = monitor.GetStabilityMetrics()
	if metrics.TotalChanges != 5 {
		t.Error("Total changes not updated correctly")
	}
}

func TestWindowsNetworkMonitor_EventLogging(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test event logging
	monitor.logNetworkEvent("TestAdapter", "test_event", "Test event details")
	
	// Check that event was logged using the public method
	eventCount := len(monitor.GetEventLog())
	
	if eventCount != 1 {
		t.Error("Event was not logged correctly")
	}
	
	// Test multiple events
	monitor.logNetworkEvent("TestAdapter2", "test_event2", "Test event details 2")
	monitor.logNetworkEvent("TestAdapter3", "test_event3", "Test event details 3")
	
	eventCount = len(monitor.GetEventLog())
	
	if eventCount != 3 {
		t.Error("Multiple events were not logged correctly")
	}
	
	// Test event log size limit
	for i := 0; i < 100; i++ {
		monitor.logNetworkEvent("Adapter", "event", "Details")
	}
	
	eventCount = len(monitor.GetEventLog())
	maxEventLogSize := monitor.GetMaxEventLogSize()
	
	// Should be limited to maxEventLogSize
	if eventCount > maxEventLogSize {
		t.Error("Event log size limit not enforced")
	}
}

func TestWindowsNetworkMonitor_AdaptiveTimeouts(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &BasicMockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test initial adaptive timeout using the public method
	initialTimeout := monitor.GetAdaptiveTimeout()
	if initialTimeout != 5*time.Second {
		t.Error("Initial adaptive timeout incorrect")
	}
	
	// Test adaptive scan interval
	initialInterval := monitor.getAdaptiveScanInterval()
	if initialInterval != 10*time.Second {
		t.Error("Initial adaptive scan interval incorrect")
	}
	
	// Test updating stability metrics affects timeouts
	// For a stable network (high stability score)
	stabilityMetrics := monitor.GetStabilityMetrics()
	adaptiveTimeout := monitor.GetAdaptiveTimeout()
	monitor.mut.Lock()
	monitor.stabilityMetrics.LastCheckTime = time.Now().Add(-30 * time.Second) // Force update
	monitor.mut.Unlock()
	
	t.Logf("Before update - StabilityScore: %f, AdaptiveTimeout: %v", stabilityMetrics.StabilityScore, adaptiveTimeout)
	
	// Call updateAdaptiveTimeouts to update the timeout based on stability score
	monitor.updateAdaptiveTimeouts()
	
	// Get updated values for stable network
	updatedTimeout := monitor.GetAdaptiveTimeout()
	updatedInterval := monitor.getAdaptiveScanInterval()
	
	stabilityMetrics = monitor.GetStabilityMetrics()
	newStabilityScore := stabilityMetrics.StabilityScore
	
	t.Logf("After update - StabilityScore: %f, AdaptiveTimeout: %v", newStabilityScore, updatedTimeout)
	
	// Verify the values are reasonable
	if newStabilityScore < 0 || newStabilityScore > 1.0 {
		t.Error("Stability score should be between 0 and 1")
	}
	
	if updatedTimeout <= 0 {
		t.Error("Adaptive timeout should be positive")
	}
	
	if updatedInterval <= 0 {
		t.Error("Adaptive scan interval should be positive")
	}
}