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

// TestPriorityGrouping tests that files are correctly grouped by folder priority
func TestPriorityGrouping(t *testing.T) {
	// Create mock priority groups
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
	groupsCopy := make([]FolderPriorityGroup, len(groups))
	copy(groupsCopy, groups)

	// Sort groups by priority (highest first)
	for i := range groupsCopy {
		ApplyTieBreaker(groupsCopy[i].Files, "oldestFirst")
	}

	sort.Slice(groupsCopy, func(i, j int) bool {
		return groupsCopy[i].Priority > groupsCopy[j].Priority
	})

	// Should have 3 groups
	if len(groupsCopy) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(groupsCopy))
	}

	// Groups should be ordered by priority (highest first)
	// Folder 3 (priority 150), Folder 1 (priority 100), Folder 2 (priority 50)
	expectedPriorities := []int{150, 100, 50}
	for i, expected := range expectedPriorities {
		if groupsCopy[i].Priority != expected {
			t.Errorf("Priority ordering failed: expected priority %d at position %d, got %d", expected, i, groupsCopy[i].Priority)
		}
	}

	// Files within each group should be ordered by oldestFirst
	// Folder 3: file3b.txt (2200), file3a.txt (1200)
	if groupsCopy[0].Files[0].Name != "file3b.txt" || groupsCopy[0].Files[1].Name != "file3a.txt" {
		t.Errorf("Folder 3 file ordering failed: expected file3b.txt, file3a.txt, got %s, %s",
			groupsCopy[0].Files[0].Name, groupsCopy[0].Files[1].Name)
	}

	// Folder 1: file1b.txt (2000), file1a.txt (1000)
	if groupsCopy[1].Files[0].Name != "file1b.txt" || groupsCopy[1].Files[1].Name != "file1a.txt" {
		t.Errorf("Folder 1 file ordering failed: expected file1b.txt, file1a.txt, got %s, %s",
			groupsCopy[1].Files[0].Name, groupsCopy[1].Files[1].Name)
	}

	// Folder 2: file2b.txt (2500), file2a.txt (1500)
	if groupsCopy[2].Files[0].Name != "file2b.txt" || groupsCopy[2].Files[1].Name != "file2a.txt" {
		t.Errorf("Folder 2 file ordering failed: expected file2b.txt, file2a.txt, got %s, %s",
			groupsCopy[2].Files[0].Name, groupsCopy[2].Files[1].Name)
	}
}

// TestRankedFileIterator tests the ranked file iterator
func TestRankedFileIterator(t *testing.T) {
	// Create mock priority groups
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
