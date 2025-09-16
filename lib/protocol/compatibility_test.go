// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"testing"
)

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		version     string
		expectedMajor int
		expectedMinor int
		expectedPatch int
		expectError   bool
	}{
		{"2.0.0", 2, 0, 0, false},
		{"v2.0.0", 2, 0, 0, false},
		{"2.0", 2, 0, 0, false},
		{"v2.0", 2, 0, 0, false},
		{"1.2.3", 1, 2, 3, false},
		{"v1.2.3", 1, 2, 3, false},
		{"2.0.0-beta", 2, 0, 0, false},
		{"v2.0.0-beta", 2, 0, 0, false},
		{"invalid", 0, 0, 0, true},
		{"", 0, 0, 0, true},
	}

	for _, test := range tests {
		major, minor, patch, err := parseSemVer(test.version)
		
		if test.expectError {
			if err == nil {
				t.Errorf("Expected error for version '%s', but got none", test.version)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("Unexpected error for version '%s': %v", test.version, err)
			continue
		}
		
		if major != test.expectedMajor || minor != test.expectedMinor || patch != test.expectedPatch {
			t.Errorf("For version '%s', expected %d.%d.%d, got %d.%d.%d", 
				test.version, test.expectedMajor, test.expectedMinor, test.expectedPatch,
				major, minor, patch)
		}
	}
}

func TestIsV2Client(t *testing.T) {
	tests := []struct {
		version string
		isV2    bool
	}{
		{"v2.0.0", true},
		{"2.0.0", true},
		{"v2.0", true},
		{"2.0", true},
		{"v2.0-beta", true},
		{"v2.0-rc.1", true},
		{"syncthing v2.0", true},
		{"v1.2.3", false},
		{"1.2.3", false},
		{"", false},
		{"v1.9.9", false},
		{"v2.0.0-beta", true},
		{"2.0.1", true},
	}

	for _, test := range tests {
		result := isV2Client(test.version)
		if result != test.isV2 {
			t.Errorf("For version '%s', expected %v, got %v", test.version, test.isV2, result)
		}
	}
}

func TestCheckCompatibility(t *testing.T) {
	tests := []struct {
		localVersion  string
		remoteVersion string
		compatible    bool
		preferred     string
	}{
		{"v2.0", "v2.0", true, "bep/2.0"},
		{"v2.0", "v1.0", true, "bep/1.0"},
		{"v1.0", "v2.0", true, "bep/1.0"},
		{"v1.0", "v1.0", true, "bep/1.0"},
		{"v3.0", "v1.0", true, "bep/1.0"}, // Fallback case
	}

	for _, test := range tests {
		compatible, preferred := CheckCompatibility(test.localVersion, test.remoteVersion)
		if compatible != test.compatible {
			t.Errorf("For local='%s', remote='%s', expected compatible=%v, got %v", 
				test.localVersion, test.remoteVersion, test.compatible, compatible)
		}
		if preferred != test.preferred {
			t.Errorf("For local='%s', remote='%s', expected preferred='%s', got '%s'", 
				test.localVersion, test.remoteVersion, test.preferred, preferred)
		}
	}
}

func TestProtocolHealthMonitor(t *testing.T) {
	monitor := NewProtocolHealthMonitor()
	
	// Test recording successful attempts
	monitor.RecordProtocolAttempt("bep/2.0", true, nil)
	monitor.RecordProtocolAttempt("bep/2.0", true, nil)
	monitor.RecordProtocolAttempt("bep/2.0", false, nil)
	
	// Test selecting optimal protocol with no data
	// This should default to bep/2.0
	optimal := monitor.SelectOptimalProtocol(DeviceID{})
	if optimal != "bep/2.0" {
		t.Errorf("Expected 'bep/2.0' with no data, got '%s'", optimal)
	}
	
	// Test with some data
	monitor.RecordProtocolAttempt("bep/1.0", true, nil)
	monitor.RecordProtocolAttempt("bep/1.0", true, nil)
	monitor.RecordProtocolAttempt("bep/1.0", true, nil)
	monitor.RecordProtocolAttempt("bep/1.0", false, nil)
	
	// With 75% success rate for bep/1.0 and 66% for bep/2.0, but threshold is 80%
	// so it should still default to bep/2.0
	optimal = monitor.SelectOptimalProtocol(DeviceID{})
	if optimal != "bep/2.0" {
		t.Errorf("Expected 'bep/2.0' with low success rates, got '%s'", optimal)
	}
}