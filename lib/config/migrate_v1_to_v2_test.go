// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"testing"
)

func TestFullMigrationFromV1ToV2(t *testing.T) {
	// Test the full migration process from a typical v1 configuration to v2
	
	// Create a typical v1 configuration (version 30, representing an older Syncthing version)
	cfg := Configuration{
		Version: 30,
		Options: OptionsConfiguration{
			// Typical v1 settings that should be updated for v2 compatibility
			ProtocolFallbackEnabled: false,
			TransferChunkSizeBytes: 0,
			PreferredProtocols: []string{}, // Empty in v1
			// Other v1 settings
			ReconnectIntervalS: 60,
			GlobalAnnEnabled: true,
			LocalAnnEnabled: true,
		},
		// Typical v1 folder configuration
		Folders: []FolderConfiguration{
			{
				ID:   "test-folder",
				Path: "/some/path",
				Type: FolderTypeSendReceive,
				// v1 default settings
				RescanIntervalS: 3600,
				FSWatcherEnabled: false, // Often disabled in v1
			},
		},
		// Typical v1 device configuration
		Devices: []DeviceConfiguration{
			{
				Name: "test-device",
				// v1 default settings
				Compression: CompressionMetadata,
				MaxSendKbps: 0,
				MaxRecvKbps: 0,
			},
		},
	}
	
	// Apply all migrations up to current version
	migrationsMut.Lock()
	migrations.apply(&cfg)
	migrationsMut.Unlock()
	
	// Verify that the configuration has been updated to v2 defaults
	if cfg.Version != CurrentVersion {
		t.Errorf("Expected version to be %d after migration, got %d", CurrentVersion, cfg.Version)
	}
	
	// Verify essential v2.0 compatibility settings are applied
	if !cfg.Options.ProtocolFallbackEnabled {
		t.Error("Expected ProtocolFallbackEnabled to be true after v1->v2 migration for compatibility")
	}
	
	// Verify protocol fallback threshold is set
	if cfg.Options.ProtocolFallbackThreshold != 3 {
		t.Errorf("Expected ProtocolFallbackThreshold to be 3, got %d", cfg.Options.ProtocolFallbackThreshold)
	}
	
	// Verify preferred protocols are set for better connectivity
	if len(cfg.Options.PreferredProtocols) == 0 {
		t.Error("Expected preferred protocols to be set after v1->v2 migration")
	}
	
	expectedProtocols := []string{"quic", "tcp", "relay"}
	if len(cfg.Options.PreferredProtocols) >= len(expectedProtocols) {
		// Check if the expected protocols are included (order might vary)
		foundQuic := false
		foundTcp := false
		foundRelay := false
		
		for _, protocol := range cfg.Options.PreferredProtocols {
			switch protocol {
			case "quic":
				foundQuic = true
			case "tcp":
				foundTcp = true
			case "relay":
				foundRelay = true
			}
		}
		
		if !foundQuic {
			t.Error("Expected 'quic' to be in preferred protocols after v1->v2 migration")
		}
		if !foundTcp {
			t.Error("Expected 'tcp' to be in preferred protocols after v1->v2 migration")
		}
		if !foundRelay {
			t.Error("Expected 'relay' to be in preferred protocols after v1->v2 migration")
		}
	}
	
	// Verify transfer chunk size is set for better performance
	if cfg.Options.TransferChunkSizeBytes != 1048576 {
		t.Errorf("Expected TransferChunkSizeBytes to be 1048576, got %d", cfg.Options.TransferChunkSizeBytes)
	}
}