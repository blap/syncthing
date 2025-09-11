// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

func TestRandomPortAllocation(t *testing.T) {
	// Create a test configuration
	cfg := config.New(protocol.LocalDeviceID)
	
	// Enable random ports
	cfg.Options.RawListenAddresses = []string{"default"}
	cfg.Options.RandomPortsEnabled = true
	cfg.Options.RandomPortRangeStart = 1024
	cfg.Options.RandomPortRangeEnd = 65535
	
	// Wrap the config properly
	wrapper := config.Wrap("/tmp/test-config.xml", cfg, protocol.LocalDeviceID, events.NoopLogger)
	
	// Test that getRandomPort returns a valid port when random ports are enabled
	port, err := getRandomPort(wrapper)
	if err != nil {
		t.Errorf("getRandomPort failed: %v", err)
	}
	
	// Port should be in the valid range or 0 (indicating default behavior)
	if port != 0 && (port < 1024 || port > 65535) {
		t.Errorf("getRandomPort returned invalid port: %d", port)
	}
	
	// Test with invalid range
	cfg.Options.RandomPortRangeStart = 70000
	cfg.Options.RandomPortRangeEnd = 80000
	wrapper = config.Wrap("/tmp/test-config2.xml", cfg, protocol.LocalDeviceID, events.NoopLogger)
	
	// Should still work (fallback to default behavior)
	port, err = getRandomPort(wrapper)
	if err != nil {
		t.Errorf("getRandomPort failed with invalid range: %v", err)
	}
	
	// Should be 0 (default behavior)
	if port != 0 {
		t.Errorf("getRandomPort should return 0 with invalid range, got: %d", port)
	}
	
	// Test with random ports disabled
	cfg.Options.RandomPortsEnabled = false
	cfg.Options.RandomPortRangeStart = 1024
	cfg.Options.RandomPortRangeEnd = 65535
	wrapper = config.Wrap("/tmp/test-config3.xml", cfg, protocol.LocalDeviceID, events.NoopLogger)
	
	// Should return 0
	port, err = getRandomPort(wrapper)
	if err != nil {
		t.Errorf("getRandomPort failed with random ports disabled: %v", err)
	}
	
	if port != 0 {
		t.Errorf("getRandomPort should return 0 when random ports are disabled, got: %d", port)
	}
}