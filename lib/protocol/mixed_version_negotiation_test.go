// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"fmt"
	"testing"
)

func TestMixedVersionFeatureNegotiation(t *testing.T) {
	tests := []struct {
		name             string
		localHello       Hello
		remoteHello      Hello
		expectedProtocol string
		expectedFeatures V2FeatureSet
	}{
		{
			name:             "Both v2.0 devices",
			localHello:       Hello{ClientVersion: "v2.0.0"},
			remoteHello:      Hello{ClientVersion: "v2.0.0"},
			expectedProtocol: "bep/2.0",
			expectedFeatures: V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  false, // Enhanced compression from v2.1.0
				ImprovedIndexing:     false, // Improved indexing from v2.2.0
			},
		},
		{
			name:             "v2.0 and v1.0 devices",
			localHello:       Hello{ClientVersion: "v2.0.0"},
			remoteHello:      Hello{ClientVersion: "v1.0.0"},
			expectedProtocol: "bep/1.0",
			expectedFeatures: V2FeatureSet{
				MultipathConnections: false,
				EnhancedCompression:  false,
				ImprovedIndexing:     false,
			},
		},
		{
			name:             "v1.0 and v2.0 devices",
			localHello:       Hello{ClientVersion: "v1.0.0"},
			remoteHello:      Hello{ClientVersion: "v2.0.0"},
			expectedProtocol: "bep/1.0",
			expectedFeatures: V2FeatureSet{
				MultipathConnections: false,
				EnhancedCompression:  false,
				ImprovedIndexing:     false,
			},
		},
		{
			name:             "Both v1.0 devices",
			localHello:       Hello{ClientVersion: "v1.0.0"},
			remoteHello:      Hello{ClientVersion: "v1.0.0"},
			expectedProtocol: "bep/1.0",
			expectedFeatures: V2FeatureSet{
				MultipathConnections: false,
				EnhancedCompression:  false,
				ImprovedIndexing:     false,
			},
		},
		{
			name:             "v2.2.0 and v2.1.0 devices",
			localHello:       Hello{ClientVersion: "v2.2.0"},
			remoteHello:      Hello{ClientVersion: "v2.1.0"},
			expectedProtocol: "bep/2.0",
			expectedFeatures: V2FeatureSet{
				MultipathConnections: true,  // Both support this
				EnhancedCompression:  true,  // Both support this (v2.1.0+)
				ImprovedIndexing:     false, // Only local supports this (v2.2.0+)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			protocol, features, err := NegotiateFeaturesForMixedVersions(test.localHello, test.remoteHello)
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if protocol != test.expectedProtocol {
				t.Errorf("Expected protocol '%s', got '%s'", test.expectedProtocol, protocol)
			}
			
			if features.MultipathConnections != test.expectedFeatures.MultipathConnections {
				t.Errorf("Expected MultipathConnections=%v, got %v", 
					test.expectedFeatures.MultipathConnections, features.MultipathConnections)
			}
			
			if features.EnhancedCompression != test.expectedFeatures.EnhancedCompression {
				t.Errorf("Expected EnhancedCompression=%v, got %v", 
					test.expectedFeatures.EnhancedCompression, features.EnhancedCompression)
			}
			
			if features.ImprovedIndexing != test.expectedFeatures.ImprovedIndexing {
				t.Errorf("Expected ImprovedIndexing=%v, got %v", 
					test.expectedFeatures.ImprovedIndexing, features.ImprovedIndexing)
			}
		})
	}
}

func TestGracefulProtocolFallback(t *testing.T) {
	localHello := Hello{ClientVersion: "v2.0.0"}
	remoteHello := Hello{ClientVersion: "v1.0.0"}
	initialError := fmt.Errorf("test error")
	
	protocol, err := GracefulProtocolFallback(localHello, remoteHello, initialError)
	
	if err != nil {
		t.Errorf("Unexpected error from fallback: %v", err)
	}
	
	if protocol != "bep/1.0" {
		t.Errorf("Expected fallback protocol 'bep/1.0', got '%s'", protocol)
	}
}
