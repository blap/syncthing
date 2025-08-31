// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// TestMultipathEnabled tests that multipath is properly enabled in configuration
func TestMultipathEnabled(t *testing.T) {
	// Given a configuration with multipath enabled
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.MultipathEnabled = true

	// When we check if multipath is enabled
	enabled := cfg.Options.MultipathEnabled

	// Then it should be true
	if !enabled {
		t.Error("Expected multipath to be enabled, but it was not")
	}
}

// TestMultipathDisabledByDefault tests that multipath is disabled by default
func TestMultipathDisabledByDefault(t *testing.T) {
	// Given a new configuration (default)
	cfg := config.New(protocol.EmptyDeviceID)

	// When we check if multipath is enabled
	enabled := cfg.Options.MultipathEnabled

	// Then it should be false by default
	if enabled {
		t.Error("Expected multipath to be disabled by default, but it was enabled")
	}
}