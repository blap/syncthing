// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows && !cgo
// +build windows,!cgo

package sqlite

import (
	"testing"
)

// TestNoCGOOnWindows verifies that when CGO is disabled on Windows,
// we use the modernc.org/sqlite driver.
func TestNoCGOOnWindows(t *testing.T) {
	// This test will only compile when CGO is disabled on Windows
	// This verifies we're on the correct build path
	if dbDriver != "sqlite" {
		t.Errorf("Expected sqlite driver for non-CGO build, got %s", dbDriver)
	}
}
