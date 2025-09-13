// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"testing"

	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

func TestVersion0Compatibility(t *testing.T) {
	c, err := NewLocal(protocol.LocalDeviceID, ":0", &fakeAddressLister{}, events.NoopLogger)
	if err != nil {
		t.Fatal(err)
	}

	lc := c.(*localClient)
	
	// Test that version 0 is now considered compatible (for older Android devices)
	if !lc.isVersionCompatible(0) {
		t.Error("Version 0 should be compatible (for older Android devices)")
	}
	
	// Test that version 1 is still compatible
	if !lc.isVersionCompatible(1) {
		t.Error("Version 1 should be compatible")
	}
	
	// Test that current version is compatible
	if !lc.isVersionCompatible(ProtocolVersion) {
		t.Errorf("Current version %d should be compatible", ProtocolVersion)
	}
}