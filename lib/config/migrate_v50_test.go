// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"testing"
)

func TestMigrateToConfigV50(t *testing.T) {
	// Test that migrateToConfigV50 properly sets v2.0 defaults
	
	// Create a configuration with pre-v2.0 settings (version 49)
	cfg := Configuration{
		Version: 49,
		Options: OptionsConfiguration{
			// These should be updated by the migration if not already set
			ProtocolFallbackEnabled: false,
			TransferChunkSizeBytes: 0,
		},
	}
	
	// Apply the v50 migration
	migrateToConfigV50(&cfg)
	
	// Verify that essential v2.0 settings are properly set
	if !cfg.Options.ProtocolFallbackEnabled {
		t.Error("Expected ProtocolFallbackEnabled to be true after v50 migration")
	}
	
	if cfg.Options.ProtocolFallbackThreshold != 3 {
		t.Errorf("Expected ProtocolFallbackThreshold to be 3, got %d", cfg.Options.ProtocolFallbackThreshold)
	}
	
	if cfg.Options.TransferChunkSizeBytes != 1048576 {
		t.Errorf("Expected TransferChunkSizeBytes to be 1048576, got %d", cfg.Options.TransferChunkSizeBytes)
	}
	
	// Test that existing values are not overridden if already set
	cfgExisting := Configuration{
		Version: 49,
		Options: OptionsConfiguration{
			// These should NOT be overridden by the migration
			ProtocolFallbackEnabled: true,
			ProtocolFallbackThreshold: 5,    // This should NOT be updated to 3
			TransferChunkSizeBytes: 2097152, // This should NOT be updated (2MB)
			PreferredProtocols: []string{"tcp", "relay"}, // This should NOT be updated
		},
	}
	
	migrateToConfigV50(&cfgExisting)
	
	// Verify that existing values are preserved
	if cfgExisting.Options.ProtocolFallbackThreshold != 5 {
		t.Errorf("Expected ProtocolFallbackThreshold to remain 5, got %d", cfgExisting.Options.ProtocolFallbackThreshold)
	}
	
	if cfgExisting.Options.TransferChunkSizeBytes != 2097152 {
		t.Errorf("Expected TransferChunkSizeBytes to remain 2097152, got %d", cfgExisting.Options.TransferChunkSizeBytes)
	}
	
	expectedProtocols := []string{"tcp", "relay"}
	if len(cfgExisting.Options.PreferredProtocols) != len(expectedProtocols) {
		t.Errorf("Expected %d preferred protocols to remain unchanged, got %d", len(expectedProtocols), len(cfgExisting.Options.PreferredProtocols))
	}
	
	for i, protocol := range expectedProtocols {
		if cfgExisting.Options.PreferredProtocols[i] != protocol {
			t.Errorf("Expected protocol at index %d to remain '%s', got '%s'", i, protocol, cfgExisting.Options.PreferredProtocols[i])
		}
	}
	
	// Test that defaults are applied when values are zero/empty
	cfgDefaults := Configuration{
		Version: 49,
		Options: OptionsConfiguration{
			// Zero/empty values that should get defaults
			ProtocolFallbackEnabled: false,
			ProtocolFallbackThreshold: 0,
			TransferChunkSizeBytes: 0,
			PreferredProtocols: []string{}, // Empty slice
		},
	}
	
	migrateToConfigV50(&cfgDefaults)
	
	// Verify that defaults are applied
	if !cfgDefaults.Options.ProtocolFallbackEnabled {
		t.Error("Expected ProtocolFallbackEnabled to be set to true for defaults")
	}
	
	if cfgDefaults.Options.ProtocolFallbackThreshold != 3 {
		t.Errorf("Expected ProtocolFallbackThreshold to be set to 3 for defaults, got %d", cfgDefaults.Options.ProtocolFallbackThreshold)
	}
	
	if cfgDefaults.Options.TransferChunkSizeBytes != 1048576 {
		t.Errorf("Expected TransferChunkSizeBytes to be set to 1048576 for defaults, got %d", cfgDefaults.Options.TransferChunkSizeBytes)
	}
	
	expectedDefaultProtocols := []string{"quic", "tcp", "relay"}
	if len(cfgDefaults.Options.PreferredProtocols) != len(expectedDefaultProtocols) {
		t.Errorf("Expected %d preferred protocols to be set for defaults, got %d", len(expectedDefaultProtocols), len(cfgDefaults.Options.PreferredProtocols))
	}
	
	for i, protocol := range expectedDefaultProtocols {
		if cfgDefaults.Options.PreferredProtocols[i] != protocol {
			t.Errorf("Expected protocol at index %d to be set to '%s' for defaults, got '%s'", i, protocol, cfgDefaults.Options.PreferredProtocols[i])
		}
	}
}