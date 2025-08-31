// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// TestFolderPriorityIntegration tests the integration of folder priority functionality
func TestFolderPriorityIntegration(t *testing.T) {
	// Create mock files for different folders with different priorities
	now := time.Now()

	// Folder 1 (priority 100)
	files1 := []protocol.FileInfo{
		{Name: "file1a.txt", Size: 100, ModifiedS: now.Unix() - 1000},
		{Name: "file1b.txt", Size: 200, ModifiedS: now.Unix() - 2000},
	}

	// Folder 2 (priority 50)
	files2 := []protocol.FileInfo{
		{Name: "file2a.txt", Size: 150, ModifiedS: now.Unix() - 1500},
		{Name: "file2b.txt", Size: 250, ModifiedS: now.Unix() - 2500},
	}

	// Folder 3 (priority 150)
	files3 := []protocol.FileInfo{
		{Name: "file3a.txt", Size: 120, ModifiedS: now.Unix() - 1200},
		{Name: "file3b.txt", Size: 220, ModifiedS: now.Unix() - 2200},
	}

	// Create folder priority groups
	groups := []FolderPriorityGroup{
		{Priority: 100, Files: files1},
		{Priority: 50, Files: files2},
		{Priority: 150, Files: files3},
	}

	// Test the PriorityFileIterator
	iterator := PriorityFileIterator(groups, "oldestFirst")

	// Collect all files from the iterator
	var collectedFiles []protocol.FileInfo
	for file := range iterator {
		collectedFiles = append(collectedFiles, file)
	}

	// Should have 6 files total
	if len(collectedFiles) != 6 {
		t.Fatalf("Expected 6 files, got %d", len(collectedFiles))
	}

	// Files should be ordered by folder priority (highest first), then by modification time within each group
	// Folder 3 (priority 150): file3b.txt (2200), file3a.txt (1200)
	// Folder 1 (priority 100): file1b.txt (2000), file1a.txt (1000)
	// Folder 2 (priority 50): file2b.txt (2500), file2a.txt (1500)
	expectedOrder := []string{
		"file3b.txt", "file3a.txt", // Folder 3 (priority 150)
		"file1b.txt", "file1a.txt", // Folder 1 (priority 100)
		"file2b.txt", "file2a.txt", // Folder 2 (priority 50)
	}

	for i, expected := range expectedOrder {
		if collectedFiles[i].Name != expected {
			t.Errorf("Priority ordering failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
		}
	}
}

// TestTieBreakerIntegration tests the integration of tie-breaker functionality
func TestTieBreakerIntegration(t *testing.T) {
	now := time.Now().Unix()

	// Create two folders with the same priority
	folder1Files := []protocol.FileInfo{
		{Name: "zebra.txt", Size: 1000, ModifiedS: now - 1000},
		{Name: "alpha.txt", Size: 500, ModifiedS: now - 2000},
	}

	folder2Files := []protocol.FileInfo{
		{Name: "beta.txt", Size: 1500, ModifiedS: now - 500},
		{Name: "gamma.txt", Size: 750, ModifiedS: now - 1500},
	}

	// Create folder priority groups with same priority
	groups := []FolderPriorityGroup{
		{Priority: 100, Files: folder1Files},
		{Priority: 100, Files: folder2Files},
	}

	// Test different tie-breaker strategies
	testCases := []struct {
		name          string
		strategy      string
		expectedOrder []string
	}{
		{
			name:          "Alphabetic",
			strategy:      "alphabetic",
			expectedOrder: []string{"alpha.txt", "zebra.txt", "beta.txt", "gamma.txt"},
		},
		{
			name:          "SmallestFirst",
			strategy:      "smallestFirst",
			expectedOrder: []string{"alpha.txt", "zebra.txt", "gamma.txt", "beta.txt"},
		},
		{
			name:          "OldestFirst",
			strategy:      "oldestFirst",
			expectedOrder: []string{"alpha.txt", "zebra.txt", "gamma.txt", "beta.txt"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a copy of groups for this test
			groupsCopy := make([]FolderPriorityGroup, len(groups))
			for i := range groups {
				groupsCopy[i] = FolderPriorityGroup{
					Priority: groups[i].Priority,
					Files:    make([]protocol.FileInfo, len(groups[i].Files)),
				}
				copy(groupsCopy[i].Files, groups[i].Files)
			}

			// Test the PriorityFileIterator with this strategy
			iterator := PriorityFileIterator(groupsCopy, tc.strategy)

			// Collect all files from the iterator
			var collectedFiles []protocol.FileInfo
			for file := range iterator {
				collectedFiles = append(collectedFiles, file)
			}

			// Verify the order
			for i, expected := range tc.expectedOrder {
				if collectedFiles[i].Name != expected {
					t.Errorf("%s tie-breaker failed: expected %s at position %d, got %s",
						tc.name, expected, i, collectedFiles[i].Name)
				}
			}
		})
	}
}

// TestGroupFilesByPriorityIntegration verifies the GroupFilesByPriority function
func TestGroupFilesByPriorityIntegration(t *testing.T) {
	// Create mock files
	files := []protocol.FileInfo{
		{Name: "file1.txt", ModifiedS: 1000},
		{Name: "file2.txt", ModifiedS: 2000},
		{Name: "file3.txt", ModifiedS: 3000},
	}

	// Create a simple iterator for the files
	fileIter := func(yield func(protocol.FileInfo) bool) {
		for _, file := range files {
			if !yield(file) {
				return
			}
		}
	}

	folderCfg := config.FolderConfiguration{ID: "folder2", Priority: 50}

	// Group files by priority
	groups, err := GroupFilesByPriority(fileIter, func() error { return nil }, folderCfg.ID, folderCfg)
	if err != nil {
		t.Fatalf("GroupFilesByPriority failed: %v", err)
	}

	// Should have one group for folder2 with priority 50
	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	if groups[0].Priority != folderCfg.Priority {
		t.Errorf("Expected priority %d, got %d", folderCfg.Priority, groups[0].Priority)
	}

	if len(groups[0].Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(groups[0].Files))
	}
}
