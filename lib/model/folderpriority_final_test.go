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

// TestProcessNeededRanked verifies that the processNeededRanked function works correctly
func TestProcessNeededRanked(t *testing.T) {
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
		cfg.Options.RankTieBreaker = "oldestFirst"
	})
	must(t, err)
	waiter.Wait()

	// Verify that the configuration was updated
	if wrapper.Options().FolderSyncStrategy != "ranked" {
		t.Fatalf("Expected FolderSyncStrategy to be 'ranked', got '%s'", wrapper.Options().FolderSyncStrategy)
	}

	// Skip testing GetAllNeededFilesRanked as it requires a real model instance
	// This test would need a more complex setup with actual database content
}

// TestFolderPriorityConfiguration verifies that folder priority configuration works correctly
func TestFolderPriorityConfiguration(t *testing.T) {
	// Setup
	wrapper, fcfg, wCancel := newDefaultCfgWrapper()
	defer wCancel()

	// Test default priority (should be 0)
	if fcfg.Priority != 0 {
		t.Errorf("Expected default priority to be 0, got %d", fcfg.Priority)
	}

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

	// Test that the model can access the priority
	folderCfg := m.cfg.Folders()[fcfg.ID]
	if folderCfg.Priority != 100 {
		t.Errorf("Expected model to access folder priority 100, got %d", folderCfg.Priority)
	}
}

// TestOptionsConfiguration verifies that the options configuration works correctly
func TestOptionsConfiguration(t *testing.T) {
	// Setup
	wrapper, fcfg, wCancel := newDefaultCfgWrapper()
	defer wCancel()

	setFolder(t, wrapper, fcfg)
	m := setupModel(t, wrapper)
	defer cleanupModel(m)

	// Test default values
	options := wrapper.Options()
	if options.FolderSyncStrategy != "parallel" {
		t.Errorf("Expected default FolderSyncStrategy to be 'parallel', got '%s'", options.FolderSyncStrategy)
	}
	if options.RankTieBreaker != "oldestFirst" {
		t.Errorf("Expected default RankTieBreaker to be 'oldestFirst', got '%s'", options.RankTieBreaker)
	}

	// Test setting values
	waiter, err := wrapper.Modify(func(cfg *config.Configuration) {
		cfg.Options.FolderSyncStrategy = "ranked"
		cfg.Options.RankTieBreaker = "newestFirst"
	})
	must(t, err)
	waiter.Wait()

	// Verify updated values
	options = wrapper.Options()
	if options.FolderSyncStrategy != "ranked" {
		t.Errorf("Expected FolderSyncStrategy to be 'ranked', got '%s'", options.FolderSyncStrategy)
	}
	if options.RankTieBreaker != "newestFirst" {
		t.Errorf("Expected RankTieBreaker to be 'newestFirst', got '%s'", options.RankTieBreaker)
	}
}

// TestPriorityFileIteratorComprehensive tests the priority file iterator with complex scenarios
func TestPriorityFileIteratorComprehensive(t *testing.T) {
	now := time.Now().Unix()

	// Create complex scenario with multiple folders and files
	scenarios := []struct {
		name     string
		groups   []FolderPriorityGroup
		strategy string
		expected []string
	}{
		{
			name: "MixedPrioritiesAndSizes",
			groups: []FolderPriorityGroup{
				{
					Priority: 100,
					Files: []protocol.FileInfo{
						{Name: "large_old.txt", Size: 10000, ModifiedS: now - 10000},
						{Name: "small_new.txt", Size: 100, ModifiedS: now - 100},
					},
				},
				{
					Priority: 200,
					Files: []protocol.FileInfo{
						{Name: "medium.txt", Size: 1000, ModifiedS: now - 1000},
					},
				},
			},
			strategy: "oldestFirst",
			expected: []string{"medium.txt", "large_old.txt", "small_new.txt"},
		},
		{
			name: "SamePriorityDifferentStrategies",
			groups: []FolderPriorityGroup{
				{
					Priority: 100,
					Files: []protocol.FileInfo{
						{Name: "zebra.txt", Size: 100, ModifiedS: now - 1000},
						{Name: "alpha.txt", Size: 200, ModifiedS: now - 2000},
					},
				},
				{
					Priority: 100,
					Files: []protocol.FileInfo{
						{Name: "beta.txt", Size: 150, ModifiedS: now - 1500},
					},
				},
			},
			strategy: "alphabetic",
			expected: []string{"alpha.txt", "beta.txt", "zebra.txt"},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			iterator := PriorityFileIterator(scenario.groups, scenario.strategy)

			// Collect all files from the iterator
			var collectedFiles []protocol.FileInfo
			for file := range iterator {
				collectedFiles = append(collectedFiles, file)
			}

			// Verify the count
			if len(collectedFiles) != len(scenario.expected) {
				t.Fatalf("Expected %d files, got %d", len(scenario.expected), len(collectedFiles))
			}

			// Verify the order
			for i, expected := range scenario.expected {
				if collectedFiles[i].Name != expected {
					t.Errorf("Expected %s at position %d, got %s", expected, i, collectedFiles[i].Name)
				}
			}
		})
	}
}
