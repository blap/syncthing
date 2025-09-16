// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"errors"
	"testing"
	
	"github.com/syncthing/syncthing/lib/protocol"
)

func TestV2ConnectionHandlerCreation(t *testing.T) {
	features := protocol.V2FeatureSet{
		MultipathConnections: true,
		EnhancedCompression:  true,
		ImprovedIndexing:     true,
	}
	
	monitor := protocol.NewProtocolHealthMonitor()
	handler := NewV2ConnectionHandler(features, monitor)
	
	if handler == nil {
		t.Fatal("Failed to create V2ConnectionHandler")
	}
	
	// Check that features are correctly set
	gotFeatures := handler.GetV2Features()
	if gotFeatures.MultipathConnections != features.MultipathConnections {
		t.Errorf("Expected MultipathConnections=%v, got %v", 
			features.MultipathConnections, gotFeatures.MultipathConnections)
	}
	
	if gotFeatures.EnhancedCompression != features.EnhancedCompression {
		t.Errorf("Expected EnhancedCompression=%v, got %v", 
			features.EnhancedCompression, gotFeatures.EnhancedCompression)
	}
	
	if gotFeatures.ImprovedIndexing != features.ImprovedIndexing {
		t.Errorf("Expected ImprovedIndexing=%v, got %v", 
			features.ImprovedIndexing, gotFeatures.ImprovedIndexing)
	}
}

func TestV2ErrorHandler(t *testing.T) {
	features := protocol.V2FeatureSet{
		MultipathConnections: true,
		EnhancedCompression:  true,
		ImprovedIndexing:     true,
	}
	
	monitor := protocol.NewProtocolHealthMonitor()
	handler := NewV2ConnectionHandler(features, monitor)
	
	// Test handling of recoverable errors
	recoverableErr := errors.New("timeout occurred")
	localHello := protocol.Hello{ClientVersion: "v2.0.0"}
	remoteHello := protocol.Hello{ClientVersion: "v2.0.0"}
	
	// The handler should return the same error (since our recovery is a no-op in tests)
	resultErr := handler.HandleV2Error(recoverableErr, localHello, remoteHello)
	if resultErr == nil {
		t.Error("Expected error to be returned, got nil")
	}
	
	// Test handling of non-recoverable errors
	nonRecoverableErr := errors.New("permanent failure")
	resultErr = handler.HandleV2Error(nonRecoverableErr, localHello, remoteHello)
	if resultErr == nil {
		t.Error("Expected error to be returned, got nil")
	}
}

func TestV2FeatureSet(t *testing.T) {
	// Test different feature combinations
	testCases := []struct {
		name     string
		features protocol.V2FeatureSet
	}{
		{
			name: "All features enabled",
			features: protocol.V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  true,
				ImprovedIndexing:     true,
			},
		},
		{
			name: "Only multipath enabled",
			features: protocol.V2FeatureSet{
				MultipathConnections: true,
				EnhancedCompression:  false,
				ImprovedIndexing:     false,
			},
		},
		{
			name: "No features enabled",
			features: protocol.V2FeatureSet{
				MultipathConnections: false,
				EnhancedCompression:  false,
				ImprovedIndexing:     false,
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			monitor := protocol.NewProtocolHealthMonitor()
			handler := NewV2ConnectionHandler(tc.features, monitor)
			
			gotFeatures := handler.GetV2Features()
			if gotFeatures.MultipathConnections != tc.features.MultipathConnections {
				t.Errorf("Expected MultipathConnections=%v, got %v", 
					tc.features.MultipathConnections, gotFeatures.MultipathConnections)
			}
			
			if gotFeatures.EnhancedCompression != tc.features.EnhancedCompression {
				t.Errorf("Expected EnhancedCompression=%v, got %v", 
					tc.features.EnhancedCompression, gotFeatures.EnhancedCompression)
			}
			
			if gotFeatures.ImprovedIndexing != tc.features.ImprovedIndexing {
				t.Errorf("Expected ImprovedIndexing=%v, got %v", 
					tc.features.ImprovedIndexing, gotFeatures.ImprovedIndexing)
			}
		})
	}
}