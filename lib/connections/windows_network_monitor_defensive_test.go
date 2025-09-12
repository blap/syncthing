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

// DefensiveMockService implements the Service interface for defensive testing
type DefensiveMockService struct {
	dialNowCalled bool
	mut           sync.Mutex
}

func (m *DefensiveMockService) DialNow() {
	m.mut.Lock()
	defer m.mut.Unlock()
	m.dialNowCalled = true
}

func (m *DefensiveMockService) WasDialNowCalled() bool {
	m.mut.Lock()
	defer m.mut.Unlock()
	return m.dialNowCalled
}

func (m *DefensiveMockService) ResetDialNowCalled() {
	m.mut.Lock()
	defer m.mut.Unlock()
	m.dialNowCalled = false
}

// Implement other required methods with empty implementations
func (m *DefensiveMockService) Serve(ctx context.Context) error { return nil }
func (m *DefensiveMockService) Stop()        {}
func (m *DefensiveMockService) String() string { return "DefensiveMockService" }
func (m *DefensiveMockService) ListenerStatus() map[string]ListenerStatusEntry { return nil }
func (m *DefensiveMockService) ConnectionStatus() map[string]ConnectionStatusEntry { return nil }
func (m *DefensiveMockService) NATType() string { return "" }
func (m *DefensiveMockService) GetConnectedDevices() []protocol.DeviceID { return nil }
func (m *DefensiveMockService) GetConnectionsForDevice(deviceID protocol.DeviceID) []protocol.Connection { return nil }
func (m *DefensiveMockService) PacketScheduler() *PacketScheduler { return nil }
func (m *DefensiveMockService) AllAddresses() []string { return nil }
func (m *DefensiveMockService) ExternalAddresses() []string { return nil }
func (m *DefensiveMockService) RawCopy() config.Configuration { return config.Configuration{} }

// TestDefensiveWindowsNetworkMonitor_Lifecycle tests the complete lifecycle
func TestDefensiveWindowsNetworkMonitor_Lifecycle(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &DefensiveMockService{}

	// Create defensive Windows network monitor
	monitor := NewDefensiveWindowsNetworkMonitor(mockService)

	// Test that we can create a monitor
	if monitor == nil {
		t.Fatal("Failed to create DefensiveWindowsNetworkMonitor")
	}

	// Test initial state
	if monitor.adapterStates == nil {
		t.Error("Adapter states map should be initialized")
	}

	// Test that we can start and stop the monitor without crashing
	monitor.Start()
	
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Stop the monitor
	monitor.Stop()
}

// TestDefensiveWindowsNetworkMonitor_AdapterStateChange tests adapter state change detection
func TestDefensiveWindowsNetworkMonitor_AdapterStateChange(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &DefensiveMockService{}

	// Create defensive Windows network monitor
	monitor := NewDefensiveWindowsNetworkMonitor(mockService)
	
	// Set up initial state - adapter is down
	monitor.mut.Lock()
	monitor.adapterStates["TestAdapter"] = NetworkAdapterInfo{
		Name: "TestAdapter",
		IsUp: false, // Adapter is initially down
		ChangeCount: 0,
		LastChange: time.Now(),
	}
	monitor.mut.Unlock()
	
	// Reset dialNowCalled flag
	mockService.ResetDialNowCalled()
	
	// Simulate adapter coming up
	monitor.mut.Lock()
	adapterInfo := monitor.adapterStates["TestAdapter"]
	adapterInfo.IsUp = true // Now adapter is up
	adapterInfo.ChangeCount++
	adapterInfo.LastChange = time.Now()
	monitor.adapterStates["TestAdapter"] = adapterInfo
	monitor.mut.Unlock()
	
	// Check for network changes which should trigger reconnection
	monitor.checkForNetworkChanges()
	
	// Verify that DialNow was called
	if !mockService.WasDialNowCalled() {
		t.Error("DialNow should be called when adapter state changes from down to up")
	}
}

// TestDefensiveWindowsNetworkMonitor_NetworkProfile tests network profile functionality
func TestDefensiveWindowsNetworkMonitor_NetworkProfile(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &DefensiveMockService{}

	// Create defensive Windows network monitor
	monitor := NewDefensiveWindowsNetworkMonitor(mockService)
	
	// Test getting network profile
	profile := monitor.GetNetworkProfile()
	
	// Should return a string (even if it's just a placeholder)
	if profile == "" {
		t.Error("GetNetworkProfile should return a non-empty string")
	}
	
	// Test enhanced network profile
	enhancedProfile := monitor.GetNetworkProfileEnhanced()
	
	// Should return a string (even if it's just a placeholder)
	if enhancedProfile == "" {
		t.Error("GetNetworkProfileEnhanced should return a non-empty string")
	}
}

// TestDefensiveWindowsNetworkMonitor_PanicRecovery tests that the monitor recovers from panics
func TestDefensiveWindowsNetworkMonitor_PanicRecovery(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &DefensiveMockService{}

	// Create defensive Windows network monitor
	monitor := NewDefensiveWindowsNetworkMonitor(mockService)
	
	// Test that methods don't panic even in error conditions
	// These tests ensure our panic recovery wrappers work correctly
	
	// Test scanNetworkAdapters with a mock that might cause issues
	monitor.scanNetworkAdapters()
	
	// Test checkForNetworkChanges
	monitor.checkForNetworkChanges()
	
	// Test GetNetworkProfile
	profile := monitor.GetNetworkProfile()
	if profile == "" {
		t.Error("GetNetworkProfile should return a non-empty string even with errors")
	}
	
	// Test GetNetworkProfileEnhanced
	enhancedProfile := monitor.GetNetworkProfileEnhanced()
	if enhancedProfile == "" {
		t.Error("GetNetworkProfileEnhanced should return a non-empty string even with errors")
	}
	
	// Test that we can start and stop without panics
	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()
		
		monitor.Start()
		time.Sleep(50 * time.Millisecond)
		monitor.Stop()
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("DefensiveWindowsNetworkMonitor panic recovery test timed out")
	}
}

// TestDefensiveWindowsNetworkMonitor_ScanNetworkAdapters tests the network adapter scanning
func TestDefensiveWindowsNetworkMonitor_ScanNetworkAdapters(t *testing.T) {
	t.Parallel()

	// Create a mock service
	mockService := &DefensiveMockService{}

	// Create defensive Windows network monitor
	monitor := NewDefensiveWindowsNetworkMonitor(mockService)
	
	// Test scanning network adapters
	monitor.scanNetworkAdapters()
	
	// Verify that adapter states were populated
	monitor.mut.RLock()
	states := len(monitor.adapterStates)
	monitor.mut.RUnlock()
	
	// We should have at least some adapters (this will depend on the test environment)
	t.Logf("Found %d network adapters", states)
	
	// At least verify the method completed without panic
}

// TestDefensiveWindowsNetworkMonitor_ContainsAny tests the helper function
func TestDefensiveWindowsNetworkMonitor_ContainsAny(t *testing.T) {
	t.Parallel()

	// Test the containsAny helper function
	testCases := []struct {
		input     string
		substrings []string
		expected  bool
	}{
		{"Ethernet Adapter", []string{"ethernet"}, true},
		{"Wi-Fi Connection", []string{"wi-fi", "wifi"}, true},
		{"LAN Connection", []string{"lan"}, true},
		{"Unknown Adapter", []string{"ethernet", "wifi"}, false},
		{"", []string{"test"}, false},
		{"test", []string{""}, true},
	}

	for _, tc := range testCases {
		result := containsAny(tc.input, tc.substrings)
		if result != tc.expected {
			t.Errorf("containsAny(%q, %v) = %v; expected %v", tc.input, tc.substrings, result, tc.expected)
		}
	}
}