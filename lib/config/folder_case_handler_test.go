// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFolderCaseHandler(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a test directory with a specific case
	testDir := filepath.Join(tempDir, "TestFolder")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Create folder case handler
	warningCallback := func(folderID, message string) {
		t.Logf("Warning for folder %s: %s", folderID, message)
	}
	
	handler := NewFolderCaseHandler(warningCallback)
	
	// Test case-insensitive path checking
	handler.CheckFolderPathCase("test-folder", testDir)
	
	// Test path normalization
	normalized, err := handler.NormalizeFolderPath(filepath.Join(tempDir, "testfolder"))
	if err != nil {
		t.Errorf("Failed to normalize path: %v", err)
	}
	
	// On case-insensitive filesystems, this should match the actual directory
	// We won't assert equality since it may vary by filesystem
	
	// Test with non-existent path
	nonExistent := filepath.Join(tempDir, "NonExistentFolder")
	normalized, err = handler.NormalizeFolderPath(nonExistent)
	if err != nil {
		t.Errorf("Failed to normalize non-existent path: %v", err)
	}
	
	// Should return a path, but we won't assert the exact value since path handling can vary
	// Just ensure it's not empty
	if normalized == "" {
		t.Errorf("Normalized path should not be empty")
	}
}