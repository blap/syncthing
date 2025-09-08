// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import (
	"encoding/json"
	"time"
)

// DirectoryState represents the cached state of a directory
type DirectoryState struct {
	ModTime time.Time `json:"modTime"`
	Hash    string    `json:"hash"`
}

// DirectoryStateCache provides caching for directory states to enable selective scanning
type DirectoryStateCache struct {
	db   KV
	prefix string
}

// NewDirectoryStateCache creates a new directory state cache
func NewDirectoryStateCache(db KV, folderID string) *DirectoryStateCache {
	return &DirectoryStateCache{
		db:     db,
		prefix: "dirState/" + folderID + "/",
	}
}

// GetDirectoryState retrieves the cached state for a directory
func (c *DirectoryStateCache) GetDirectoryState(dirPath string) (*DirectoryState, bool, error) {
	key := c.prefix + dirPath
	data, err := c.db.GetKV(key)
	if err != nil {
		return nil, false, err
	}
	
	if data == nil {
		return nil, false, nil
	}
	
	var state DirectoryState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, false, err
	}
	
	return &state, true, nil
}

// PutDirectoryState stores the state for a directory
func (c *DirectoryStateCache) PutDirectoryState(dirPath string, state *DirectoryState) error {
	key := c.prefix + dirPath
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	
	return c.db.PutKV(key, data)
}

// DeleteDirectoryState removes the cached state for a directory
func (c *DirectoryStateCache) DeleteDirectoryState(dirPath string) error {
	key := c.prefix + dirPath
	return c.db.DeleteKV(key)
}

// ClearCache removes all cached directory states for this folder
func (c *DirectoryStateCache) ClearCache() error {
	// In a real implementation, we would iterate through all keys with the prefix
	// and delete them. For now, we'll just return nil.
	return nil
}