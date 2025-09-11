// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"testing"

	"github.com/syncthing/syncthing/lib/protocol"
)

func TestCheckPathOverlaps(t *testing.T) {
	// Test cases for folder path overlap detection
	testCases := []struct {
		name          string
		folder        FolderConfiguration
		allFolders    []FolderConfiguration
		expectError   bool
		errorContains string
	}{
		{
			name: "no overlap",
			folder: FolderConfiguration{
				ID:   "folder1",
				Path: "/home/user/folder1",
			},
			allFolders: []FolderConfiguration{
				{
					ID:   "folder2",
					Path: "/home/user/folder2",
				},
			},
			expectError: false,
		},
		{
			name: "same path",
			folder: FolderConfiguration{
				ID:   "folder1",
				Path: "/home/user/folder",
			},
			allFolders: []FolderConfiguration{
				{
					ID:   "folder2",
					Path: "/home/user/folder",
				},
			},
			expectError:   true,
			errorContains: "is the same as folder",
		},
		{
			name: "nested folders - parent contains child",
			folder: FolderConfiguration{
				ID:   "folder1",
				Path: "/home/user/folder",
			},
			allFolders: []FolderConfiguration{
				{
					ID:   "folder2",
					Path: "/home/user/folder/subfolder",
				},
			},
			expectError: false,
		},
		{
			name: "nested folders - child in parent",
			folder: FolderConfiguration{
				ID:   "folder1",
				Path: "/home/user/folder/subfolder",
			},
			allFolders: []FolderConfiguration{
				{
					ID:   "folder2",
					Path: "/home/user/folder",
				},
			},
			expectError: false,
		},
		{
			name: "self exclusion",
			folder: FolderConfiguration{
				ID:   "folder1",
				Path: "/home/user/folder1",
			},
			allFolders: []FolderConfiguration{
				{
					ID:   "folder1",
					Path: "/home/user/folder1",
				},
				{
					ID:   "folder2",
					Path: "/home/user/folder2",
				},
			},
			expectError: false,
		},
		{
			name: "empty path skip",
			folder: FolderConfiguration{
				ID:   "folder1",
				Path: "/home/user/folder1",
			},
			allFolders: []FolderConfiguration{
				{
					ID:   "folder2",
					Path: "",
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.folder.checkPathOverlaps(tc.allFolders)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.errorContains != "" && err.Error() != tc.errorContains && !containsString(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain '%s', but got '%s'", tc.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestNestedFolderConfiguration tests that nested folder configurations are now allowed
func TestNestedFolderConfiguration(t *testing.T) {
	// Create a configuration with nested folders
	cfg := Configuration{
		Version: CurrentVersion,
		Folders: []FolderConfiguration{
			{
				ID:   "parent",
				Path: "/home/user/Syncthing",
			},
			{
				ID:   "child",
				Path: "/home/user/Syncthing/tea",
			},
		},
	}

	// This should not return an error with our changes
	err := cfg.prepare(protocol.EmptyDeviceID)
	if err != nil {
		t.Errorf("Expected no error for nested folder configuration, but got: %v", err)
	}
}

// TestOriginalErrorCase tests the specific case mentioned in the issue
func TestOriginalErrorCase(t *testing.T) {
	// This is the exact case from the issue:
	// Folder "ahqrm-5jgc7": "D:\Syncthing\Syncthing"
	// Folder "p6qu4-ks3vs": "D:\Syncthing\Syncthing\tea"
	cfg := Configuration{
		Version: CurrentVersion,
		Folders: []FolderConfiguration{
			{
				ID:   "ahqrm-5jgc7",
				Path: "D:\\Syncthing\\Syncthing",
			},
			{
				ID:   "p6qu4-ks3vs",
				Path: "D:\\Syncthing\\Syncthing\\tea",
			},
		},
	}

	// This should not return an error with our changes
	err := cfg.prepare(protocol.EmptyDeviceID)
	if err != nil {
		t.Errorf("Expected no error for the original error case, but got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) < len(s) && contains(s, substr))
}

// Simple contains implementation for testing
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}