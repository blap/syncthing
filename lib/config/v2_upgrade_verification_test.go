// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"testing"
)

// TestV2UpgradeCompatibility verifies that the v2.0 migration addresses
// the original issue of connection problems when upgrading from v1.x to v2.x
func TestV2UpgradeCompatibility(t *testing.T) {
	// Simulate a typical v1.x configuration that would cause connection issues
	v1Config := Configuration{
		Version: 30, // Typical v1.x config version
		Options: OptionsConfiguration{
			// These settings in v1.x could cause connection issues with v2.x devices
			ProtocolFallbackEnabled: false,     // No fallback to older protocols
			ProtocolFallbackThreshold: 0,       // No threshold set
			TransferChunkSizeBytes: 0,          // No optimized chunk size
			PreferredProtocols: []string{},     // No preferred protocols set
			AdaptiveKeepAliveEnabled: false,    // No adaptive keep-alive
		},
	}
	
	// Apply all migrations including our v2.0 migration
	migrationsMut.Lock()
	migrations.apply(&v1Config)
	migrationsMut.Unlock()
	
	// Verify that essential v2.0 compatibility settings are now enabled
	// These settings address the original connection issues
	
	// Protocol fallback should be enabled to handle compatibility between versions
	if !v1Config.Options.ProtocolFallbackEnabled {
		t.Error("Protocol fallback should be enabled for v1/v2 compatibility")
	}
	
	// A reasonable fallback threshold should be set
	if v1Config.Options.ProtocolFallbackThreshold != 3 {
		t.Errorf("Expected protocol fallback threshold of 3, got %d", v1Config.Options.ProtocolFallbackThreshold)
	}
	
	// Transfer chunk size should be optimized for better performance
	if v1Config.Options.TransferChunkSizeBytes != 1048576 {
		t.Errorf("Expected transfer chunk size of 1MB (1048576 bytes), got %d", v1Config.Options.TransferChunkSizeBytes)
	}
	
	// Modern protocols should be preferred for better connectivity
	if len(v1Config.Options.PreferredProtocols) == 0 {
		t.Error("Preferred protocols should be set for better connectivity")
	}
	
	// Adaptive keep-alive should be enabled for connection stability
	if !v1Config.Options.AdaptiveKeepAliveEnabled {
		t.Error("Adaptive keep-alive should be enabled for connection stability")
	}
	
	// The configuration should now be at the current version
	if v1Config.Version != CurrentVersion {
		t.Errorf("Expected configuration to be upgraded to version %d, got %d", CurrentVersion, v1Config.Version)
	}
	
	t.Log("v2.0 upgrade compatibility migration successfully applied all essential settings")
	t.Log("This should resolve connection issues when upgrading from v1.x to v2.x")
}