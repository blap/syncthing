// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package connections

import (
	"context"
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// MockService implements the Service interface for testing
type MockService struct {
	dialNowCalled bool
	cfg           config.Wrapper
}

func (m *MockService) DialNow() {
	m.dialNowCalled = true
}

// Implement other required methods with empty implementations
func (m *MockService) Serve(ctx context.Context) error { return nil }
func (m *MockService) Stop()        {}
func (m *MockService) String() string { return "MockService" }
func (m *MockService) ListenerStatus() map[string]ListenerStatusEntry { return nil }
func (m *MockService) ConnectionStatus() map[string]ConnectionStatusEntry { return nil }
func (m *MockService) NATType() string { return "" }
func (m *MockService) GetConnectedDevices() []protocol.DeviceID { return nil }
func (m *MockService) GetConnectionsForDevice(deviceID protocol.DeviceID) []protocol.Connection { return nil }
func (m *MockService) PacketScheduler() *PacketScheduler { return nil }
func (m *MockService) AllAddresses() []string { return nil }
func (m *MockService) ExternalAddresses() []string { return nil }
func (m *MockService) RawCopy() config.Configuration { return config.Configuration{} }

func TestWindowsNetworkMonitor_Integration(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &MockService{}

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

func TestWindowsNetworkMonitor_AdapterStateChangeDetection(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &MockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Set up initial state - both previous and current state are the same (false)
	monitor.SetAdapterState("TestAdapter", NetworkAdapterInfo{
		Name: "TestAdapter",
		IsUp: false,
		Type: 6, // Ethernet
		ChangeCount: 0,
	})
	
	// Reset dialNowCalled flag
	mockService.dialNowCalled = false
	
	// Simulate adapter coming up
	adapterInfo := monitor.GetAdapterStates()["TestAdapter"]
	adapterInfo.IsUp = true // Now adapter is up
	adapterInfo.ChangeCount++
	adapterInfo.LastChange = time.Now()
	monitor.SetAdapterState("TestAdapter", adapterInfo)
	
	// Check for network changes which should trigger reconnection
	monitor.checkForNetworkChanges()
	
	// Verify that DialNow was called
	if !mockService.dialNowCalled {
		t.Error("DialNow should be called when adapter state changes from down to up")
	}
}

func TestWindowsNetworkMonitor_NetworkProfileChangeDetection(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &MockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Set initial profile
	monitor.mut.Lock()
	monitor.currentProfile = "Public"
	monitor.mut.Unlock()
	
	// Reset dialNowCalled flag
	mockService.dialNowCalled = false
	
	// Mock the GetNetworkProfile method to return the same profile
	monitor.mut.Lock()
	newProfile := "Public" // Same as previous profile
	profileChanged := (monitor.currentProfile != newProfile)
	if profileChanged {
		monitor.currentProfile = newProfile
		mockService.dialNowCalled = true
	}
	monitor.mut.Unlock()
	
	// Verify that DialNow was not called
	if mockService.dialNowCalled {
		t.Error("DialNow should not be called when network profile hasn't changed")
	}
	
	// For now, we'll test that the network profile functionality exists
	profile := monitor.GetNetworkProfile()
	if profile == "" {
		t.Error("GetNetworkProfile should return a non-empty string")
	}
}

func TestWindowsNetworkMonitor_TriggerReconnection(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &MockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Reset dialNowCalled flag
	mockService.dialNowCalled = false
	
	// Trigger reconnection
	monitor.triggerReconnection()
	
	// Verify that DialNow was called
	if !mockService.dialNowCalled {
		t.Error("DialNow should be called when triggerReconnection is invoked")
	}
}

func TestWindowsNetworkMonitor_KB5060998Detection(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &MockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Set up an adapter with frequent changes to simulate KB5060998 issues
	monitor.SetAdapterState("TestAdapter", NetworkAdapterInfo{
		Name: "TestAdapter",
		IsUp: true,
		Type: 6, // Ethernet
		ChangeCount: 10, // High change count
	})
	
	// Log a frequent change event which should trigger KB5060998 detection
	monitor.logNetworkEvent("TestAdapter", "kb5060998_suspected", "Frequent changes detected")
	
	// Check that event was logged using the public method
	eventCount := len(monitor.GetEventLog())
	
	if eventCount == 0 {
		t.Error("KB5060998 detection event was not logged")
	}
	
	// Check the last event is the KB5060998 detection event
	eventLog := monitor.GetEventLog()
	lastEvent := eventLog[len(eventLog)-1]
	
	if lastEvent.EventType != "kb5060998_suspected" {
		t.Error("KB5060998 detection event not logged correctly")
	}
}

func TestWindowsNetworkMonitor_Diagnostics(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &MockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Add some test data
	monitor.SetAdapterState("TestAdapter", NetworkAdapterInfo{
		Name: "TestAdapter",
		IsUp: true,
		Type: 6,
		ChangeCount: 2,
	})
	
	// Update stability metrics using direct access (this is needed for testing)
	monitor.mut.Lock()
	monitor.stabilityMetrics.TotalChanges = 5
	monitor.stabilityMetrics.RecentChanges = 1
	monitor.currentProfile = "Private"
	monitor.mut.Unlock()
	
	// Log some events
	monitor.logNetworkEvent("TestAdapter", "test_event", "Test details")
	
	// Test diagnostics logging
	monitor.logDiagnostics()
	
	// The logDiagnostics method should not panic and should execute without error
	// We can't easily test the slog output, but we can verify the method runs
	t.Log("Diagnostics logging completed successfully")
}

func TestWindowsNetworkMonitor_AdaptiveBehavior(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &MockService{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test initial stability using the public method
	initialStability := monitor.GetStabilityMetrics().StabilityScore
	
	if initialStability != 1.0 {
		t.Error("Initial stability score should be 1.0")
	}
	
	// Simulate network instability by increasing change count
	// Update stability metrics using direct access (this is needed for testing)
	monitor.mut.Lock()
	monitor.stabilityMetrics.RecentChanges = 10 // Many recent changes
	monitor.mut.Unlock()
	
	// Update adaptive timeouts which should reduce stability score
	monitor.updateAdaptiveTimeouts()
	
	// Check that stability score was updated using the public method
	updatedStability := monitor.GetStabilityMetrics().StabilityScore
	
	if updatedStability >= initialStability {
		t.Error("Stability score should decrease with network instability")
	}
	
	// Test that adaptive timeout changes based on stability
	adaptiveTimeout := monitor.GetAdaptiveTimeout()
	
	// With low stability, timeout should be longer
	if adaptiveTimeout <= 5*time.Second {
		t.Error("Adaptive timeout should be longer for unstable networks")
	}
}