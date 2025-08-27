// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows && cgo
// +build windows,cgo

package sqlite

import (
	"testing"
)

// TestCGOEnabledOnWindows verifies that when CGO is enabled on Windows,
// we use the mattn/go-sqlite3 driver which doesn't have the libc dependency.
func TestCGOEnabledOnWindows(t *testing.T) {
	// This test will only compile when CGO is enabled on Windows
	// If this test compiles and runs, it means we're using the CGO path
	// which should avoid the modernc.org/libc issue
	if dbDriver != "sqlite3" {
		t.Errorf("Expected sqlite3 driver for CGO build, got %s", dbDriver)
	}
}
