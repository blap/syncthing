// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"sort"
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// TestFolderPriorityOrderingPriority tests that files from higher priority folders are processed first
func TestFolderPriorityOrderingPriority(t *testing.T) {
	// Create mock files for different folders with different priorities
	now := time.Now()

	// Folder 1 (priority 100)
	files1 := []protocol.FileInfo{
		{Name: "file1a.txt", ModifiedS: now.Unix() - 1000},
		{Name: "file1b.txt", ModifiedS: now.Unix() - 2000},
	}

	// Folder 2 (priority 50)
	files2 := []protocol.FileInfo{
		{Name: "file2a.txt", ModifiedS: now.Unix() - 1500},
		{Name: "file2b.txt", ModifiedS: now.Unix() - 2500},
	}

	// Folder 3 (priority 150)
	files3 := []protocol.FileInfo{
		{Name: "file3a.txt", ModifiedS: now.Unix() - 1200},
		{Name: "file3b.txt", ModifiedS: now.Unix() - 2200},
	}

	// Create folder priority groups
	groups := []FolderPriorityGroup{
		{Priority: 100, Files: files1},
		{Priority: 50, Files: files2},
		{Priority: 150, Files: files3},
	}

	// Sort groups by priority (highest first)
	for i := range groups {
		ApplyTieBreaker(groups[i].Files, "oldestFirst")
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Priority > groups[j].Priority
	})

	// Collect all files from the groups
	var collectedFiles []protocol.FileInfo
	for _, group := range groups {
		for _, file := range group.Files {
			collectedFiles = append(collectedFiles, file)
		}
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

// TestTieBreakerStrategies tests different tie-breaker strategies with same priority folders
func TestTieBreakerStrategies(t *testing.T) {
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

	// Test alphabetic tie-breaker
	t.Run("Alphabetic", func(t *testing.T) {
		// Sort groups by priority (highest first)
		for i := range groups {
			ApplyTieBreaker(groups[i].Files, "alphabetic")
		}

		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Priority > groups[j].Priority
		})

		// Collect all files from the groups
		var collectedFiles []protocol.FileInfo
		for _, group := range groups {
			for _, file := range group.Files {
				collectedFiles = append(collectedFiles, file)
			}
		}

		// Should be ordered alphabetically within groups: alpha.txt, zebra.txt (folder1), beta.txt, gamma.txt (folder2)
		expectedOrder := []string{"alpha.txt", "zebra.txt", "beta.txt", "gamma.txt"}
		for i, expected := range expectedOrder {
			if collectedFiles[i].Name != expected {
				t.Errorf("Alphabetic tie-breaker failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
			}
		}
	})

	// Test smallestFirst tie-breaker
	t.Run("SmallestFirst", func(t *testing.T) {
		// Sort groups by priority (highest first)
		for i := range groups {
			ApplyTieBreaker(groups[i].Files, "smallestFirst")
		}

		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Priority > groups[j].Priority
		})

		// Collect all files from the groups
		var collectedFiles []protocol.FileInfo
		for _, group := range groups {
			for _, file := range group.Files {
				collectedFiles = append(collectedFiles, file)
			}
		}

		// Should be ordered by size (smallest first) within groups: alpha.txt(500), zebra.txt(1000) (folder1), gamma.txt(750), beta.txt(1500) (folder2)
		expectedOrder := []string{"alpha.txt", "zebra.txt", "gamma.txt", "beta.txt"}
		for i, expected := range expectedOrder {
			if collectedFiles[i].Name != expected {
				t.Errorf("SmallestFirst tie-breaker failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
			}
		}
	})

	// Test oldestFirst tie-breaker (default)
	t.Run("OldestFirst", func(t *testing.T) {
		// Sort groups by priority (highest first)
		for i := range groups {
			ApplyTieBreaker(groups[i].Files, "oldestFirst")
		}

		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Priority > groups[j].Priority
		})

		// Collect all files from the groups
		var collectedFiles []protocol.FileInfo
		for _, group := range groups {
			for _, file := range group.Files {
				collectedFiles = append(collectedFiles, file)
			}
		}

		// Should be ordered by modification time (oldest first) within groups: alpha.txt(-2000), zebra.txt(-1000) (folder1), gamma.txt(-1500), beta.txt(-500) (folder2)
		expectedOrder := []string{"alpha.txt", "zebra.txt", "gamma.txt", "beta.txt"}
		for i, expected := range expectedOrder {
			if collectedFiles[i].Name != expected {
				t.Errorf("OldestFirst tie-breaker failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
			}
		}
	})
}
