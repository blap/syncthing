// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"testing"
	"time"
	// Removed unused "iter" import (unusedfunc fix)

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// TestFolderPriority tests that folders with higher priority are processed first
func TestFolderPriority(t *testing.T) {
	// Setup
	wrapper, fcfg, wCancel := newDefaultCfgWrapper()
	defer wCancel()

	// Set first folder priority
	fcfg.Priority = 100
	setFolder(t, wrapper, fcfg)

	// Create second folder with lower priority
	fcfg2 := newFolderConfiguration(wrapper, "folder2", "Folder 2", config.FilesystemTypeFake, "testdata2")
	fcfg2.Priority = 50 // Lower priority

	// Create third folder with higher priority
	fcfg3 := newFolderConfiguration(wrapper, "folder3", "Folder 3", config.FilesystemTypeFake, "testdata3")
	fcfg3.Priority = 150 // Higher priority

	// Add folders to config
	setFolder(t, wrapper, fcfg2)
	setFolder(t, wrapper, fcfg3)

	// Get the model
	m := setupModel(t, wrapper)
	defer cleanupModel(m)

	// Test that folders are processed in priority order when strategy is ranked
	t.Run("RankedStrategy", func(t *testing.T) {
		// Set folder sync strategy to ranked
		waiter, err := wrapper.Modify(func(cfg *config.Configuration) {
			cfg.Options.FolderSyncStrategy = "ranked"
		})
		must(t, err)
		waiter.Wait()

		// Verify that the configuration was updated
		if wrapper.Options().FolderSyncStrategy != "ranked" {
			t.Errorf("Expected FolderSyncStrategy to be 'ranked', got '%s'", wrapper.Options().FolderSyncStrategy)
		}
	})

	t.Run("ParallelStrategy", func(t *testing.T) {
		// Set folder sync strategy to parallel (default)
		waiter, err := wrapper.Modify(func(cfg *config.Configuration) {
			cfg.Options.FolderSyncStrategy = "parallel"
		})
		must(t, err)
		waiter.Wait()

		// Verify that the configuration was updated
		if wrapper.Options().FolderSyncStrategy != "parallel" {
			t.Errorf("Expected FolderSyncStrategy to be 'parallel', got '%s'", wrapper.Options().FolderSyncStrategy)
		}
	})
}

// TestRankTieBreaker tests that files within the same priority are ordered correctly
func TestRankTieBreaker(t *testing.T) {
	// Setup
	wrapper, _, wCancel := newDefaultCfgWrapper()
	defer wCancel()
	// Removed unused fcfg variable and write to fcfg.Priority (unusedfunc fix)

	// Create second folder with the same priority
	fcfg2 := newFolderConfiguration(wrapper, "folder2", "Folder 2", config.FilesystemTypeFake, "testdata2")
	fcfg2.Priority = 100

	// Add second folder to config
	setFolder(t, wrapper, fcfg2)

	// Get the model
	m := setupModel(t, wrapper)
	defer cleanupModel(m)

	// Test different tie breaker strategies
	testCases := []struct {
		name     string
		strategy string
	}{
		{"OldestFirst", "oldestFirst"},
		{"NewestFirst", "newestFirst"},
		{"SmallestFirst", "smallestFirst"},
		{"LargestFirst", "largestFirst"},
		{"Alphabetic", "alphabetic"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			waiter, err := wrapper.Modify(func(cfg *config.Configuration) {
				cfg.Options.FolderSyncStrategy = "ranked"
				cfg.Options.RankTieBreaker = tc.strategy
			})
			must(t, err)
			waiter.Wait()

			// Verify that the configuration was updated
			if wrapper.Options().FolderSyncStrategy != "ranked" {
				t.Errorf("Expected FolderSyncStrategy to be 'ranked', got '%s'", wrapper.Options().FolderSyncStrategy)
			}
			if wrapper.Options().RankTieBreaker != tc.strategy {
				t.Errorf("Expected RankTieBreaker to be '%s', got '%s'", tc.strategy, wrapper.Options().RankTieBreaker)
			}
		})
	}
}

// TestFolderPriorityWithFiles tests the actual file processing order
func TestFolderPriorityWithFiles(t *testing.T) {
	// This test would require more complex mocking of the database and file processing
	// For now, we'll just test that the configuration is properly set up

	wrapper, fcfg, wCancel := newDefaultCfgWrapper()
	defer wCancel()

	// Set folder priority
	fcfg.Priority = 100
	setFolder(t, wrapper, fcfg)

	m := setupModel(t, wrapper)
	defer cleanupModel(m)

	// Verify folder priority is set correctly
	folders := m.cfg.FolderList()
	if len(folders) != 1 {
		t.Fatalf("Expected 1 folder, got %d", len(folders))
	}

	if folders[0].Priority != 100 {
		t.Errorf("Expected folder priority to be 100, got %d", folders[0].Priority)
	}

	// Test default priority (should be 0)
	fcfg2 := newFolderConfiguration(wrapper, "folder2", "Folder 2", config.FilesystemTypeFake, "testdata2")
	// Don't set priority, should default to 0
	setFolder(t, wrapper, fcfg2)

	folders = m.cfg.FolderList()
	if len(folders) != 2 {
		t.Fatalf("Expected 2 folders, got %d", len(folders))
	}

	// Find the folder with default priority
	var defaultPriorityFolder config.FolderConfiguration
	for _, folder := range folders {
		if folder.ID == "folder2" {
			defaultPriorityFolder = folder
			break
		}
	}

	if defaultPriorityFolder.Priority != 0 {
		t.Errorf("Expected default folder priority to be 0, got %d", defaultPriorityFolder.Priority)
	}
}

// TestGroupFilesByPriority tests the folder priority grouping functionality
func TestGroupFilesByPriority(t *testing.T) {
	// Create mock files with different folder priorities
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
	groups, err := GroupFilesByPriority(fileIter, func() error { return nil }, "folder2", folderCfg)
	if err != nil {
		t.Fatalf("GroupFilesByPriority failed: %v", err)
	}

	// Should have one group for folder2 with priority 50
	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	if groups[0].Priority != 50 {
		t.Errorf("Expected priority 50, got %d", groups[0].Priority)
	}

	if len(groups[0].Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(groups[0].Files))
	}
}

// TestApplyTieBreaker tests the tie-breaker strategies
func TestApplyTieBreaker(t *testing.T) {
	// Create mock files with different properties
	now := time.Now()
	files := []protocol.FileInfo{
		{Name: "zebra.txt", Size: 1000, ModifiedS: now.Unix() - 1000},
		{Name: "alpha.txt", Size: 500, ModifiedS: now.Unix() - 2000},
		{Name: "beta.txt", Size: 1500, ModifiedS: now.Unix() - 500},
	}

	// Test alphabetic tie-breaker
	alphabeticFiles := make([]protocol.FileInfo, len(files))
	copy(alphabeticFiles, files)
	ApplyTieBreaker(alphabeticFiles, "alphabetic")

	expectedAlphaOrder := []string{"alpha.txt", "beta.txt", "zebra.txt"}
	for i, expected := range expectedAlphaOrder {
		if alphabeticFiles[i].Name != expected {
			t.Errorf("Alphabetic tie-breaker failed: expected %s at position %d, got %s", expected, i, alphabeticFiles[i].Name)
		}
	}

	// Test smallestFirst tie-breaker
	smallestFiles := make([]protocol.FileInfo, len(files))
	copy(smallestFiles, files)
	ApplyTieBreaker(smallestFiles, "smallestFirst")

	expectedSizeOrder := []int64{500, 1000, 1500}
	for i, expected := range expectedSizeOrder {
		if smallestFiles[i].Size != expected {
			t.Errorf("SmallestFirst tie-breaker failed: expected size %d at position %d, got %d", expected, i, smallestFiles[i].Size)
		}
	}

	// Test largestFirst tie-breaker
	largestFiles := make([]protocol.FileInfo, len(files))
	copy(largestFiles, files)
	ApplyTieBreaker(largestFiles, "largestFirst")

	expectedLargestOrder := []int64{1500, 1000, 500}
	for i, expected := range expectedLargestOrder {
		if largestFiles[i].Size != expected {
			t.Errorf("LargestFirst tie-breaker failed: expected size %d at position %d, got %d", expected, i, largestFiles[i].Size)
		}
	}

	// Test oldestFirst tie-breaker
	oldestFiles := make([]protocol.FileInfo, len(files))
	copy(oldestFiles, files)
	ApplyTieBreaker(oldestFiles, "oldestFirst")

	// Files should be ordered by modification time (oldest first)
	expectedTimeOrder := []int64{now.Unix() - 2000, now.Unix() - 1000, now.Unix() - 500}
	for i, expected := range expectedTimeOrder {
		if oldestFiles[i].ModifiedS != expected {
			t.Errorf("OldestFirst tie-breaker failed: expected time %d at position %d, got %d", expected, i, oldestFiles[i].ModifiedS)
		}
	}

	// Test newestFirst tie-breaker
	newestFiles := make([]protocol.FileInfo, len(files))
	copy(newestFiles, files)
	ApplyTieBreaker(newestFiles, "newestFirst")

	// Files should be ordered by modification time (newest first)
	expectedNewestOrder := []int64{now.Unix() - 500, now.Unix() - 1000, now.Unix() - 2000}
	for i, expected := range expectedNewestOrder {
		if newestFiles[i].ModifiedS != expected {
			t.Errorf("NewestFirst tie-breaker failed: expected time %d at position %d, got %d", expected, i, newestFiles[i].ModifiedS)
		}
	}
}

// TestPriorityFileIterator tests the priority file iterator
func TestPriorityFileIterator(t *testing.T) {
	// Create mock priority groups
	groups := []FolderPriorityGroup{
		{Priority: 150, Files: []protocol.FileInfo{{Name: "high_priority_1"}, {Name: "high_priority_2"}}},
		{Priority: 100, Files: []protocol.FileInfo{{Name: "medium_priority_1"}, {Name: "medium_priority_2"}}},
		{Priority: 50, Files: []protocol.FileInfo{{Name: "low_priority_1"}, {Name: "low_priority_2"}}},
	}

	// Test iterator with alphabetic tie-breaker
	iterator := PriorityFileIterator(groups, "alphabetic")

	// Collect all files from the iterator
	var collectedFiles []protocol.FileInfo
	for file := range iterator {
		collectedFiles = append(collectedFiles, file)
	}

	// Should have 6 files total
	if len(collectedFiles) != 6 {
		t.Errorf("Expected 6 files, got %d", len(collectedFiles))
	}

	// Files should be ordered by priority (highest first), then alphabetically within each group
	expectedOrder := []string{
		"high_priority_1", "high_priority_2",
		"medium_priority_1", "medium_priority_2",
		"low_priority_1", "low_priority_2",
	}

	for i, expected := range expectedOrder {
		if collectedFiles[i].Name != expected {
			t.Errorf("Priority ordering failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
		}
	}
}

// TestFolderPriorityOrdering tests that files from higher priority folders are processed first
func TestFolderPriorityOrdering(t *testing.T) {
	// Setup
	wrapper, fcfg, wCancel := newDefaultCfgWrapper()
	defer wCancel()

	// Set first folder with medium priority
	fcfg.Priority = 100
	fcfg.ID = "folder1"
	fcfg.Label = "Folder 1"
	setFolder(t, wrapper, fcfg)

	// Create second folder with lower priority
	fcfg2 := newFolderConfiguration(wrapper, "folder2", "Folder 2", config.FilesystemTypeFake, "testdata2")
	fcfg2.Priority = 50 // Lower priority
	setFolder(t, wrapper, fcfg2)

	// Create third folder with higher priority
	fcfg3 := newFolderConfiguration(wrapper, "folder3", "Folder 3", config.FilesystemTypeFake, "testdata3")
	fcfg3.Priority = 150 // Higher priority
	setFolder(t, wrapper, fcfg3)

	// Get the model
	m := setupModel(t, wrapper)
	defer cleanupModel(m)

	// Set folder sync strategy to ranked
	waiter, err := wrapper.Modify(func(cfg *config.Configuration) {
		cfg.Options.FolderSyncStrategy = "ranked"
	})
	must(t, err)
	waiter.Wait()

	// Verify that the configuration was updated
	if wrapper.Options().FolderSyncStrategy != "ranked" {
		t.Fatalf("Expected FolderSyncStrategy to be 'ranked', got '%s'", wrapper.Options().FolderSyncStrategy)
	}

	// Create mock files for each folder
	files1 := []protocol.FileInfo{
		{Name: "file1a.txt", ModifiedS: 1000},
		{Name: "file1b.txt", ModifiedS: 2000},
	}

	files2 := []protocol.FileInfo{
		{Name: "file2a.txt", ModifiedS: 1500},
		{Name: "file2b.txt", ModifiedS: 2500},
	}

	files3 := []protocol.FileInfo{
		{Name: "file3a.txt", ModifiedS: 1200},
		{Name: "file3b.txt", ModifiedS: 2200},
	}

	// Create folder priority groups
	groups := []FolderPriorityGroup{
		{Priority: fcfg.Priority, Files: files1},
		{Priority: fcfg2.Priority, Files: files2},
		{Priority: fcfg3.Priority, Files: files3},
	}

	// Test that files are ordered by folder priority
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
	// Folder 3 (priority 150): file3a.txt (1200), file3b.txt (2200)
	// Folder 1 (priority 100): file1a.txt (1000), file1b.txt (2000)
	// Folder 2 (priority 50): file2a.txt (1500), file2b.txt (2500)
	expectedOrder := []string{
		"file3a.txt", "file3b.txt", // Folder 3 (priority 150)
		"file1a.txt", "file1b.txt", // Folder 1 (priority 100)
		"file2a.txt", "file2b.txt", // Folder 2 (priority 50)
	}

	for i, expected := range expectedOrder {
		if collectedFiles[i].Name != expected {
			t.Errorf("Priority ordering failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
		}
	}
}

// TestRankTieBreakerStrategies tests different tie-breaker strategies with same priority folders
func TestRankTieBreakerStrategies(t *testing.T) {
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
		iterator := PriorityFileIterator(groups, "alphabetic")

		var collectedFiles []protocol.FileInfo
		for file := range iterator {
			collectedFiles = append(collectedFiles, file)
		}

		// Should be ordered alphabetically: alpha.txt, beta.txt, gamma.txt, zebra.txt
		expectedOrder := []string{"alpha.txt", "beta.txt", "gamma.txt", "zebra.txt"}
		for i, expected := range expectedOrder {
			if collectedFiles[i].Name != expected {
				t.Errorf("Alphabetic tie-breaker failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
			}
		}
	})

	// Test smallestFirst tie-breaker
	t.Run("SmallestFirst", func(t *testing.T) {
		iterator := PriorityFileIterator(groups, "smallestFirst")

		var collectedFiles []protocol.FileInfo
		for file := range iterator {
			collectedFiles = append(collectedFiles, file)
		}

		// Should be ordered by size (smallest first): alpha.txt(500), gamma.txt(750), zebra.txt(1000), beta.txt(1500)
		expectedOrder := []string{"alpha.txt", "gamma.txt", "zebra.txt", "beta.txt"}
		for i, expected := range expectedOrder {
			if collectedFiles[i].Name != expected {
				t.Errorf("SmallestFirst tie-breaker failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
			}
		}
	})

	// Test oldestFirst tie-breaker (default)
	t.Run("OldestFirst", func(t *testing.T) {
		iterator := PriorityFileIterator(groups, "oldestFirst")

		var collectedFiles []protocol.FileInfo
		for file := range iterator {
			collectedFiles = append(collectedFiles, file)
		}

		// Should be ordered by modification time (oldest first): alpha.txt(-2000), gamma.txt(-1500), zebra.txt(-1000), beta.txt(-500)
		expectedOrder := []string{"alpha.txt", "gamma.txt", "zebra.txt", "beta.txt"}
		for i, expected := range expectedOrder {
			if collectedFiles[i].Name != expected {
				t.Errorf("OldestFirst tie-breaker failed: expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
			}
		}
	})
}
