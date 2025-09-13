// Copyright (C) 2015 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"context"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/thejerf/suture/v4"
)

// A cachedFinder is a Finder with associated cache timeouts.
type cachedFinder struct {
	// Embedded fields should be listed first
	Finder
	cacheTime    time.Duration
	negCacheTime time.Duration
	cache        *cache
	token        *suture.ServiceToken
}

// An error may implement cachedError, in which case it will be interrogated
// to see how long we should cache the error. This overrides the default
// negative cache time.
type cachedError interface {
	CacheFor() time.Duration
}

// A cache can be embedded wherever useful

type cache struct {
	entries map[protocol.DeviceID]CacheEntry
	mut     sync.Mutex
	// Statistics for cache performance monitoring
	hits   int64
	misses int64
	// Automatic cleanup
	cleanupInterval time.Duration
	cleanupCtx      context.Context
	cleanupCancel   context.CancelFunc
	cleanupRunning  bool
}

func newCache() *cache {
	ctx, cancel := context.WithCancel(context.Background())
	c := &cache{
		entries:         make(map[protocol.DeviceID]CacheEntry),
		cleanupInterval: 10 * time.Minute, // Default cleanup interval
		cleanupCtx:      ctx,
		cleanupCancel:   cancel,
	}
	c.startCleanup()
	return c
}

// startCleanup starts the automatic cache cleanup goroutine
func (c *cache) startCleanup() {
	c.mut.Lock()
	defer c.mut.Unlock()
	
	if c.cleanupRunning {
		return // Already running
	}
	
	c.cleanupRunning = true
	go c.cleanupLoop()
}

// stopCleanup stops the automatic cache cleanup goroutine
func (c *cache) stopCleanup() {
	c.mut.Lock()
	defer c.mut.Unlock()
	
	if !c.cleanupRunning {
		return // Not running
	}
	
	c.cleanupRunning = false
	if c.cleanupCancel != nil {
		c.cleanupCancel()
	}
}

// CleanupExpired removes expired entries from the cache
func (c *cache) CleanupExpired(defaultTTL time.Duration) int {
	c.mut.Lock()
	defer c.mut.Unlock()
	
	removed := 0
	now := time.Now()
	
	for id, ce := range c.entries {
		ttl := defaultTTL
		// Use custom TTL if specified in the entry
		if !ce.validUntil.IsZero() {
			ttl = ce.validUntil.Sub(ce.when)
		}
		
		if now.Sub(ce.when) > ttl {
			delete(c.entries, id)
			removed++
		}
	}
	
	return removed
}

// Close stops the automatic cache cleanup and releases resources
func (c *cache) Close() error {
	c.stopCleanup()
	return nil
}

// cleanupLoop runs the periodic cache cleanup
func (c *cache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Clean up expired entries with default TTL of 1 hour
			removed := c.CleanupExpired(1 * time.Hour)
			if removed > 0 {
				// Log cleanup activity
			}
		case <-c.cleanupCtx.Done():
			return
		}
	}
}

// SetCleanupInterval sets the interval for automatic cache cleanup
func (c *cache) SetCleanupInterval(interval time.Duration) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.cleanupInterval = interval
}

func (c *cache) Set(id protocol.DeviceID, ce CacheEntry) {
	c.mut.Lock()
	c.entries[id] = ce
	c.mut.Unlock()
}

func (c *cache) Get(id protocol.DeviceID) (CacheEntry, bool) {
	c.mut.Lock()
	ce, ok := c.entries[id]
	if ok {
		c.hits++
	} else {
		c.misses++
	}
	c.mut.Unlock()
	return ce, ok
}

// GetWithTTL checks if a cache entry exists and is still valid based on TTL
func (c *cache) GetWithTTL(id protocol.DeviceID, ttl time.Duration) (CacheEntry, bool) {
	c.mut.Lock()
	ce, ok := c.entries[id]
	if !ok {
		c.misses++
		c.mut.Unlock()
		return ce, false
	}
	
	c.hits++
	// Check if entry is still valid based on TTL
	if time.Since(ce.when) > ttl {
		// Entry expired, remove it
		delete(c.entries, id)
		c.hits-- // Adjust hit count since it's actually a miss
		c.misses++
		c.mut.Unlock()
		return CacheEntry{}, false
	}
	c.mut.Unlock()
	return ce, true
}

// GetStats returns cache hit/miss statistics
func (c *cache) GetStats() (hits, misses int64) {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.hits, c.misses
}

// ResetStats resets cache hit/miss statistics
func (c *cache) ResetStats() {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.hits = 0
	c.misses = 0
}

func (c *cache) Cache() map[protocol.DeviceID]CacheEntry {
	c.mut.Lock()
	m := make(map[protocol.DeviceID]CacheEntry, len(c.entries))
	for k, v := range c.entries {
		m[k] = v
	}
	c.mut.Unlock()
	return m
}
