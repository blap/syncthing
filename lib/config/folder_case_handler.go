// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FolderCaseHandler handles case sensitivity issues with folder paths
type FolderCaseHandler struct {
	warningCallback func(folderID, message string)
}

// NewFolderCaseHandler creates a new folder case handler
func NewFolderCaseHandler(warningCallback func(folderID, message string)) *FolderCaseHandler {
	return &FolderCaseHandler{
		warningCallback: warningCallback,
	}
}

// CheckFolderPathCase checks if a folder path has case sensitivity issues
func (fch *FolderCaseHandler) CheckFolderPathCase(folderID, folderPath string) {
	// Normalize the path for comparison
	normalizedPath := filepath.Clean(folderPath)
	
	// Check if the path exists with the exact case
	if _, err := os.Stat(normalizedPath); err != nil {
		// Path doesn't exist with exact case, check if it exists with different case
		dir := filepath.Dir(normalizedPath)
		base := filepath.Base(normalizedPath)
		
		// Check if parent directory exists
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			return
		}
		
		// Look for entries with different case
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), base) && entry.Name() != base {
				// Found a case mismatch
				message := "Folder path has case mismatch. Filesystem has '" + entry.Name() + "' but config specifies '" + base + "'"
				slog.Warn("Case sensitivity issue detected", 
					"folder", folderID, 
					"message", message)
				
				if fch.warningCallback != nil {
					fch.warningCallback(folderID, message)
				}
				break
			}
		}
	}
}

// NormalizeFolderPath normalizes a folder path to match the actual case on disk
func (fch *FolderCaseHandler) NormalizeFolderPath(folderPath string) (string, error) {
	// Normalize the path components to match actual case on disk
	normalizedPath := filepath.Clean(folderPath)
	
	// Split the path into components
	components := strings.Split(normalizedPath, string(filepath.Separator))
	
	// Start with the root
	currentPath := ""
	
	for i, component := range components {
		if component == "" {
			// Handle root component
			if i == 0 {
				currentPath = string(filepath.Separator)
			}
			continue
		}
		
		// For the first component on Windows, it might be a drive letter
		if i == 0 && len(component) >= 2 && component[1] == ':' {
			currentPath = component
			continue
		}
		
		// Check if current path exists
		if _, err := os.Stat(currentPath); err != nil {
			// If the current path doesn't exist, we can't normalize further
			// Return the remaining path as-is
			remaining := strings.Join(components[i:], string(filepath.Separator))
			if currentPath == "" {
				return remaining, nil
			}
			return filepath.Join(currentPath, remaining), nil
		}
		
		// Read directory entries
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			// If we can't read the directory, return the path as-is
			return folderPath, nil
		}
		
		// Look for the component with case-insensitive matching
		found := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), component) {
				// Use the actual case from the filesystem
				if currentPath == "" {
					currentPath = entry.Name()
				} else {
					currentPath = filepath.Join(currentPath, entry.Name())
				}
				found = true
				break
			}
		}
		
		// If not found, use the original component
		if !found {
			if currentPath == "" {
				currentPath = component
			} else {
				currentPath = filepath.Join(currentPath, component)
			}
		}
	}
	
	return currentPath, nil
}