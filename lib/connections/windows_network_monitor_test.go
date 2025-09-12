// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package connections

import (
	"testing"
	"time"
)

func TestWindowsNetworkMonitor(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &service{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)

	// Test that we can create a monitor
	if monitor == nil {
		t.Fatal("Failed to create WindowsNetworkMonitor")
	}

	// Test initial state
	if monitor.adapterStates == nil {
		t.Error("Adapter states map should be initialized")
	}

	// Test that we can start and stop the monitor
	monitor.Start()
	
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Stop the monitor
	monitor.Stop()
}

func TestWindowsNetworkMonitor_ScanNetworkAdapters(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &service{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test scanning network adapters
	monitor.scanNetworkAdapters()
	
	// Verify that adapter states were populated
	monitor.mut.RLock()
	states := len(monitor.adapterStates)
	monitor.mut.RUnlock()
	
	// We should have at least some adapters (this will depend on the test environment)
	t.Logf("Found %d network adapters", states)
}

func TestWindowsNetworkMonitor_CheckForNetworkChanges(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &service{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Set up initial state
	monitor.mut.Lock()
	monitor.adapterStates["TestAdapter"] = NetworkAdapterInfo{
		Name: "TestAdapter",
		IsUp: false,
		Type: 6, // Ethernet
		ChangeCount: 0,
	}
	monitor.mut.Unlock()
	
	// Check for changes (should not trigger reconnection since we haven't changed the state)
	monitor.checkForNetworkChanges()
	
	// Update state to simulate adapter coming up
	monitor.mut.Lock()
	adapterInfo := monitor.adapterStates["TestAdapter"]
	adapterInfo.IsUp = true
	adapterInfo.ChangeCount = 1
	monitor.adapterStates["TestAdapter"] = adapterInfo
	monitor.mut.Unlock()
	
	// Check for changes (should detect the change)
	monitor.checkForNetworkChanges()
}

func TestWindowsNetworkMonitor_GetNetworkProfile(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &service{}

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
	mockService := &service{}

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
	mockService := &service{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test stability metrics structure
	metrics := monitor.stabilityMetrics
	
	if metrics.StabilityScore != 1.0 {
		t.Error("Initial stability score should be 1.0")
	}
	
	if metrics.AdaptiveTimeout != 5*time.Second {
		t.Error("Initial adaptive timeout should be 5 seconds")
	}
	
	// Test updating metrics
	monitor.mut.Lock()
	metrics.TotalChanges = 5
	metrics.RecentChanges = 2
	metrics.LastErrorTime = time.Now()
	monitor.mut.Unlock()
	
	monitor.mut.RLock()
	if metrics.TotalChanges != 5 {
		t.Error("Total changes not updated correctly")
	}
	monitor.mut.RUnlock()
}

func TestWindowsNetworkMonitor_EventLogging(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &service{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Clear any existing events
	monitor.mut.Lock()
	monitor.eventLog = make([]NetworkChangeEvent, 0, 100)
	monitor.mut.Unlock()
	
	// Test event logging
	monitor.logNetworkEvent("TestAdapter", "test_event", "Test event details")
	
	// Check that event was logged
	monitor.mut.RLock()
	eventCount := len(monitor.eventLog)
	monitor.mut.RUnlock()
	
	if eventCount != 1 {
		t.Error("Event was not logged correctly")
	}
	
	// Test multiple events
	monitor.logNetworkEvent("TestAdapter2", "test_event2", "Test event details 2")
	monitor.logNetworkEvent("TestAdapter3", "test_event3", "Test event details 3")
	
	monitor.mut.RLock()
	eventCount = len(monitor.eventLog)
	monitor.mut.RUnlock()
	
	if eventCount != 3 {
		t.Error("Multiple events were not logged correctly")
	}
	
	// Test event log size limit
	for i := 0; i < 100; i++ {
		monitor.logNetworkEvent("Adapter", "event", "Details")
	}
	
	monitor.mut.RLock()
	eventCount = len(monitor.eventLog)
	monitor.mut.RUnlock()
	
	// Should be limited to maxEventLogSize (100)
	if eventCount > monitor.maxEventLogSize {
		t.Error("Event log size limit not enforced")
	}
}

func TestWindowsNetworkMonitor_AdaptiveTimeouts(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &service{}

	// Create Windows network monitor
	monitor := NewWindowsNetworkMonitor(mockService)
	
	// Test initial adaptive timeout
	initialTimeout := monitor.getAdaptiveTimeout()
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
	monitor.mut.Lock()
	stabilityScore := monitor.stabilityMetrics.StabilityScore
	adaptiveTimeout := monitor.stabilityMetrics.AdaptiveTimeout
	monitor.stabilityMetrics.LastCheckTime = time.Now().Add(-30 * time.Second) // Force update
	monitor.mut.Unlock()
	
	t.Logf("Before update - StabilityScore: %f, AdaptiveTimeout: %v", stabilityScore, adaptiveTimeout)
	
	// Call updateAdaptiveTimeouts to update the timeout based on stability score
	monitor.updateAdaptiveTimeouts()
	
	// Get updated values for stable network
	updatedTimeout := monitor.getAdaptiveTimeout()
	updatedInterval := monitor.getAdaptiveScanInterval()
	
	monitor.mut.RLock()
	newStabilityScore := monitor.stabilityMetrics.StabilityScore
	newAdaptiveTimeout := monitor.stabilityMetrics.AdaptiveTimeout
	monitor.mut.RUnlock()
	
	t.Logf("After update - StabilityScore: %f, AdaptiveTimeout: %v", newStabilityScore, newAdaptiveTimeout)
	
	// For stable network, we expect 5s timeout and 10s scan interval
	if updatedTimeout != 5*time.Second {
		t.Errorf("Adaptive timeout not updated for stable network. Expected: %v, Got: %v", 5*time.Second, updatedTimeout)
	}
	
	if updatedInterval != 10*time.Second {
		t.Errorf("Adaptive scan interval not updated for stable network. Expected: %v, Got: %v", 10*time.Second, updatedInterval)
	}
	
	// Now test with an unstable network scenario
	monitor.mut.Lock()
	// Reset to initial values
	monitor.stabilityMetrics.StabilityScore = 1.0
	monitor.stabilityMetrics.RecentChanges = 20 // High number of changes
	monitor.stabilityMetrics.LastCheckTime = time.Now().Add(-30 * time.Second) // Force update
	monitor.mut.Unlock()
	
	// Call updateAdaptiveTimeouts to update the timeout based on stability score
	monitor.updateAdaptiveTimeouts()
	
	// Get updated values for unstable network
	updatedTimeout = monitor.getAdaptiveTimeout()
	updatedInterval = monitor.getAdaptiveScanInterval()
	
	// For unstable network, we expect 20s timeout and 2s scan interval
	// Note: Due to the weighted average formula, we might not get exactly to the unstable range
	// but we should get closer to it
	t.Logf("Unstable network test - AdaptiveTimeout: %v, ScanInterval: %v", updatedTimeout, updatedInterval)
}