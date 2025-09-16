// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"errors"
	"testing"
)

func TestV2ProtocolNegotiation(t *testing.T) {
	tests := []struct {
		localHello     Hello
		remoteHello    Hello
		expectedProtocol string
		expectError    bool
	}{
		{
			localHello:     Hello{ClientVersion: "v2.0.0"},
			remoteHello:    Hello{ClientVersion: "v2.0.0"},
			expectedProtocol: "bep/2.0",
			expectError:    false,
		},
		{
			localHello:     Hello{ClientVersion: "v2.0.0"},
			remoteHello:    Hello{ClientVersion: "v1.0.0"},
			expectedProtocol: "bep/1.0",
			expectError:    false,
		},
		{
			localHello:     Hello{ClientVersion: "v1.0.0"},
			remoteHello:    Hello{ClientVersion: "v2.0.0"},
			expectedProtocol: "bep/1.0",
			expectError:    false,
		},
		{
			localHello:     Hello{ClientVersion: "v1.0.0"},
			remoteHello:    Hello{ClientVersion: "v1.0.0"},
			expectedProtocol: "bep/1.0",
			expectError:    false,
		},
	}

	for _, test := range tests {
		protocol, err := negotiateV2Protocol(test.localHello, test.remoteHello)
		
		if test.expectError {
			if err == nil {
				t.Errorf("Expected error for local='%s', remote='%s', but got none", 
					test.localHello.ClientVersion, test.remoteHello.ClientVersion)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("Unexpected error for local='%s', remote='%s': %v", 
				test.localHello.ClientVersion, test.remoteHello.ClientVersion, err)
			continue
		}
		
		if protocol != test.expectedProtocol {
			t.Errorf("For local='%s', remote='%s', expected protocol='%s', got '%s'", 
				test.localHello.ClientVersion, test.remoteHello.ClientVersion, 
				test.expectedProtocol, protocol)
		}
	}
}

func TestV2FeatureDetection(t *testing.T) {
	tests := []struct {
		localHello  Hello
		remoteHello Hello
		expectedFeatures V2FeatureSet
	}{
		{
			localHello:  Hello{ClientVersion: "v2.0.0"},
			remoteHello: Hello{ClientVersion: "v2.0.0"},
			expectedFeatures: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  false, // Enhanced compression from v2.1.0
				ImprovedIndexing:     false, // Improved indexing from v2.2.0
			},
		},
		{
			localHello:  Hello{ClientVersion: "v2.1.0"},
			remoteHello: Hello{ClientVersion: "v2.0.0"},
			expectedFeatures: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  false, // Only local has this
				ImprovedIndexing:     false, // Only local has this
			},
		},
		{
			localHello:  Hello{ClientVersion: "v2.0.0"},
			remoteHello: Hello{ClientVersion: "v2.2.0"},
			expectedFeatures: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  false, // Only remote has this
				ImprovedIndexing:     false, // Only remote has this
			},
		},
		{
			localHello:  Hello{ClientVersion: "v2.2.0"},
			remoteHello: Hello{ClientVersion: "v2.1.0"},
			expectedFeatures: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  true,  // Both have this (v2.1.0+)
				ImprovedIndexing:     false, // Only local has this (v2.2.0+)
			},
		},
		{
			localHello:  Hello{ClientVersion: "v2.2.0"},
			remoteHello: Hello{ClientVersion: "v2.2.0"},
			expectedFeatures: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  true,  // Both have this (v2.1.0+)
				ImprovedIndexing:     true,  // Both have this (v2.2.0+)
			},
		},
	}

	for _, test := range tests {
		features := DetectV2Features(test.localHello, test.remoteHello)
		
		if features.MultipathConnections != test.expectedFeatures.MultipathConnections {
			t.Errorf("For local='%s', remote='%s', expected MultipathConnections=%v, got %v", 
				test.localHello.ClientVersion, test.remoteHello.ClientVersion,
				test.expectedFeatures.MultipathConnections, features.MultipathConnections)
		}
		
		if features.EnhancedCompression != test.expectedFeatures.EnhancedCompression {
			t.Errorf("For local='%s', remote='%s', expected EnhancedCompression=%v, got %v", 
				test.localHello.ClientVersion, test.remoteHello.ClientVersion,
				test.expectedFeatures.EnhancedCompression, features.EnhancedCompression)
		}
		
		if features.ImprovedIndexing != test.expectedFeatures.ImprovedIndexing {
			t.Errorf("For local='%s', remote='%s', expected ImprovedIndexing=%v, got %v", 
				test.localHello.ClientVersion, test.remoteHello.ClientVersion,
				test.expectedFeatures.ImprovedIndexing, features.ImprovedIndexing)
		}
	}
}

func TestV2ErrorHandling(t *testing.T) {
	// Test that recoverable errors are identified correctly
	recoverableErrors := []error{
		errors.New("timeout occurred"),
		errors.New("temporary network issue"),
		errors.New("EOF"),
	}
	
	for _, err := range recoverableErrors {
		if !isErrorRecoverable(err) {
			t.Errorf("Expected error '%v' to be recoverable, but it was not", err)
		}
	}
	
	// Test that non-recoverable errors are identified correctly
	nonRecoverableErrors := []error{
		errors.New("permanent failure"),
		errors.New("invalid configuration"),
		errors.New("authentication failed"),
	}
	
	for _, err := range nonRecoverableErrors {
		if isErrorRecoverable(err) {
			t.Errorf("Expected error '%v' to be non-recoverable, but it was", err)
		}
	}
}

func TestV2FeatureParsing(t *testing.T) {
	tests := []struct {
		version  string
		expected V2FeatureSet
	}{
		{
			version: "v2.0.0",
			expected: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  false,
				ImprovedIndexing:     false,
			},
		},
		{
			version: "v2.1.0",
			expected: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  true,
				ImprovedIndexing:     false,
			},
		},
		{
			version: "v2.2.0",
			expected: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  true,
				ImprovedIndexing:     true,
			},
		},
		{
			version: "v3.0.0",
			expected: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  true,
				ImprovedIndexing:     true,
			},
		},
		{
			version: "invalid",
			expected: V2FeatureSet{
				MultipathConnections: false,
				EnhancedCompression:  false,
				ImprovedIndexing:     false,
			},
		},
	}

	for _, test := range tests {
		features := parseV2Features(test.version)
		
		if features.MultipathConnections != test.expected.MultipathConnections {
			t.Errorf("For version='%s', expected MultipathConnections=%v, got %v", 
				test.version, test.expected.MultipathConnections, features.MultipathConnections)
		}
		
		if features.EnhancedCompression != test.expected.EnhancedCompression {
			t.Errorf("For version='%s', expected EnhancedCompression=%v, got %v", 
				test.version, test.expected.EnhancedCompression, features.EnhancedCompression)
		}
		
		if features.ImprovedIndexing != test.expected.ImprovedIndexing {
			t.Errorf("For version='%s', expected ImprovedIndexing=%v, got %v", 
				test.version, test.expected.ImprovedIndexing, features.ImprovedIndexing)
		}
	}
}