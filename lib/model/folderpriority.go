// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"iter"
	"sort"

	"github.com/syncthing/syncthing/internal/itererr"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// FolderPriorityGroup represents a group of files with the same priority
type FolderPriorityGroup struct {
	Priority int
	Files    []protocol.FileInfo
}

// GroupFilesByPriority groups files by their folder priority
func GroupFilesByPriority(files iter.Seq[protocol.FileInfo], errFn func() error, folderID string, folderCfg config.FolderConfiguration) ([]FolderPriorityGroup, error) {
	// Group files by priority - in this case all files are from the same folder
	priority := folderCfg.Priority

	// Collect all files
	var fileSlice []protocol.FileInfo
	for file, err := range itererr.Zip(files, errFn) {
		if err != nil {
			return nil, err
		}
		fileSlice = append(fileSlice, file)
	}

	// Create a single group for this folder
	groups := []FolderPriorityGroup{
		{
			Priority: priority,
			Files:    fileSlice,
		},
	}

	return groups, nil
}

// ApplyTieBreaker sorts files within the same priority group according to the tie-breaker strategy
func ApplyTieBreaker(files []protocol.FileInfo, strategy string) {
	switch strategy {
	case "alphabetic":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name < files[j].Name
		})
	case "smallestFirst":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size < files[j].Size
		})
	case "largestFirst":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size > files[j].Size
		})
	case "oldestFirst":
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime().Before(files[j].ModTime())
		})
	case "newestFirst":
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime().After(files[j].ModTime())
		})
	default:
		// Default to oldestFirst if unknown strategy
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime().Before(files[j].ModTime())
		})
	}
}

// PriorityFileIterator creates an iterator that yields files in priority order
func PriorityFileIterator(groups []FolderPriorityGroup, tieBreaker string) iter.Seq[protocol.FileInfo] {
	return func(yield func(protocol.FileInfo) bool) {
		// Sort groups by priority (highest first)
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Priority > groups[j].Priority
		})

		// Merge groups with the same priority
		var mergedGroups []FolderPriorityGroup
		for _, group := range groups {
			if len(mergedGroups) > 0 && mergedGroups[len(mergedGroups)-1].Priority == group.Priority {
				// Merge with the last group
				lastGroup := &mergedGroups[len(mergedGroups)-1]
				lastGroup.Files = append(lastGroup.Files, group.Files...)
			} else {
				// Add as a new group
				mergedGroups = append(mergedGroups, group)
			}
		}

		for _, group := range mergedGroups {
			// Apply tie-breaker within the group
			ApplyTieBreaker(group.Files, tieBreaker)

			// Yield all files in this group
			for _, file := range group.Files {
				if !yield(file) {
					return
				}
			}
		}
	}
}

// GetAllNeededFilesRanked gets all needed files from all folders, grouped by priority
func GetAllNeededFilesRanked(m *model, tieBreaker string) ([]FolderPriorityGroup, error) {
	// Get all folders
	folders := m.cfg.FolderList()

	// Create priority groups for each folder
	var groups []FolderPriorityGroup

	for _, folder := range folders {
		// Get needed files for this folder
		files, errFn := m.sdb.AllNeededGlobalFiles(folder.ID, protocol.LocalDeviceID, folder.Order, 0, 0)
		if files == nil {
			continue
		}

		// Group files by priority
		folderGroups, err := GroupFilesByPriority(files, errFn, folder.ID, folder)
		if err != nil {
			return nil, err
		}

		// Add to overall groups
		groups = append(groups, folderGroups...)
	}

	// Sort groups by priority (highest first)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Priority > groups[j].Priority
	})

	// Apply tie-breaker within each group
	for i := range groups {
		ApplyTieBreaker(groups[i].Files, tieBreaker)
	}

	return groups, nil
}
