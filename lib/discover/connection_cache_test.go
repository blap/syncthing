// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

func TestConnectionCache(t *testing.T) {
	// Create a cache with a short TTL for testing
	ttl := 100 * time.Millisecond
	cache := newConnectionCache(ttl)

	// Create a test device ID
	deviceID := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	addresses := []string{"tcp://192.168.1.100:22000"}

	// Test adding an entry
	cache.Add(deviceID, addresses)

	// Test retrieving an entry
	retrieved, found := cache.Get(deviceID)
	if !found {
		t.Fatal("Expected to find entry but it was not found")
	}

	if len(retrieved) != len(addresses) {
		t.Fatalf("Expected %d addresses, got %d", len(addresses), len(retrieved))
	}

	if retrieved[0] != addresses[0] {
		t.Errorf("Expected address %s, got %s", addresses[0], retrieved[0])
	}

	// Test that cache size is correct
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Test removing an entry
	cache.Remove(deviceID)
	_, found = cache.Get(deviceID)
	if found {
		t.Error("Expected not to find entry after removal but it was found")
	}

	// Test that cache size is correct after removal
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}
}

func TestConnectionCacheExpiration(t *testing.T) {
	// Create a cache with a very short TTL for testing
	ttl := 10 * time.Millisecond
	cache := newConnectionCache(ttl)

	// Create a test device ID
	deviceID := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	addresses := []string{"tcp://192.168.1.100:22000"}

	// Add an entry
	cache.Add(deviceID, addresses)

	// Wait for the entry to expire
	time.Sleep(20 * time.Millisecond)

	// Try to retrieve the expired entry
	_, found := cache.Get(deviceID)
	if found {
		t.Error("Expected not to find entry after expiration but it was found")
	}

	// Test that cache size is correct after expiration
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}
}

func TestConnectionCacheCleanExpired(t *testing.T) {
	// Create a cache with a short TTL for testing
	ttl := 10 * time.Millisecond
	cache := newConnectionCache(ttl)

	// Create test device IDs
	deviceID1 := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	deviceID2 := protocol.DeviceID{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33}
	addresses := []string{"tcp://192.168.1.100:22000"}

	// Add entries
	cache.Add(deviceID1, addresses)
	cache.Add(deviceID2, addresses)

	// Wait for entries to expire
	time.Sleep(20 * time.Millisecond)

	// Clean expired entries
	cache.CleanExpired()

	// Test that cache size is correct after cleaning
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}
}
