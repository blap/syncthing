// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// connectionCache stores successful connection information for devices
type connectionCache struct {
	entries map[protocol.DeviceID]connectionCacheEntry
	mut     sync.RWMutex
	ttl     time.Duration
}

// connectionCacheEntry stores connection information for a device
type connectionCacheEntry struct {
	Addresses []string  `json:"addresses"`
	when      time.Time // When was this entry created
}

// newConnectionCache creates a new connection cache with the specified TTL
func newConnectionCache(ttl time.Duration) *connectionCache {
	return &connectionCache{
		entries: make(map[protocol.DeviceID]connectionCacheEntry),
		ttl:     ttl,
	}
}

// Add adds a new entry to the cache
func (c *connectionCache) Add(id protocol.DeviceID, addresses []string) {
	c.mut.Lock()
	defer c.mut.Unlock()

	c.entries[id] = connectionCacheEntry{
		Addresses: addresses,
		when:      time.Now(),
	}
}

// Get retrieves addresses for a device from the cache if they exist and are still valid
func (c *connectionCache) Get(id protocol.DeviceID) ([]string, bool) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	entry, exists := c.entries[id]
	if !exists {
		return nil, false
	}

	// Check if the entry is still valid
	if time.Since(entry.when) > c.ttl {
		// Entry has expired, remove it
		c.mut.RUnlock()
		c.mut.Lock()
		delete(c.entries, id)
		c.mut.Unlock()
		c.mut.RLock()
		return nil, false
	}

	return entry.Addresses, true
}

// Remove removes an entry from the cache
func (c *connectionCache) Remove(id protocol.DeviceID) {
	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.entries, id)
}

// CleanExpired removes all expired entries from the cache
func (c *connectionCache) CleanExpired() {
	c.mut.Lock()
	defer c.mut.Unlock()

	now := time.Now()
	for id, entry := range c.entries {
		if now.Sub(entry.when) > c.ttl {
			delete(c.entries, id)
		}
	}
}

// Size returns the number of entries in the cache
func (c *connectionCache) Size() int {
	c.mut.RLock()
	defer c.mut.RUnlock()

	return len(c.entries)
}
