// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"sync"
)

// MemoryLimiter controls memory usage across multiple components
type MemoryLimiter struct {
	// Total memory limit in bytes
	limit int64

	// Current memory usage
	currentUsage int64

	// Map of component IDs to their memory usage
	componentUsage map[string]int64
	mu             sync.RWMutex
}

// NewMemoryLimiter creates a new memory limiter with no limit
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{
		limit:          0, // No limit by default
		componentUsage: make(map[string]int64),
	}
}

// SetLimit sets the total memory limit in bytes (0 for no limit)
func (ml *MemoryLimiter) SetLimit(limit int64) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.limit = limit
}

// GetLimit returns the current memory limit
func (ml *MemoryLimiter) GetLimit() int64 {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	return ml.limit
}

// GetCurrentUsage returns the current total memory usage
func (ml *MemoryLimiter) GetCurrentUsage() int64 {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	return ml.currentUsage
}

// RequestMemory requests memory allocation for a component
// Returns true if allocation is allowed, false otherwise
func (ml *MemoryLimiter) RequestMemory(componentID string, size int64) bool {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	// If no limit is set, allow allocation
	if ml.limit <= 0 {
		ml.componentUsage[componentID] = size
		ml.currentUsage += size
		return true
	}

	// Check if this allocation would exceed the limit
	if ml.currentUsage+size > ml.limit {
		return false
	}

	// Allocate memory
	ml.componentUsage[componentID] = size
	ml.currentUsage += size
	return true
}

// ReleaseMemory releases previously allocated memory for a component
func (ml *MemoryLimiter) ReleaseMemory(componentID string, size int64) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	currentUsage := ml.componentUsage[componentID]
	if currentUsage <= size {
		delete(ml.componentUsage, componentID)
		ml.currentUsage -= currentUsage
	} else {
		ml.componentUsage[componentID] = currentUsage - size
		ml.currentUsage -= size
	}
}

// GetComponentUsage returns the memory usage for a specific component
func (ml *MemoryLimiter) GetComponentUsage(componentID string) int64 {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	return ml.componentUsage[componentID]
}

// GetComponents returns a list of all components with their memory usage
func (ml *MemoryLimiter) GetComponents() map[string]int64 {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	// Create a copy of the map
	result := make(map[string]int64, len(ml.componentUsage))
	for k, v := range ml.componentUsage {
		result[k] = v
	}

	return result
}

// IsMemoryAvailable checks if a certain amount of memory is available
func (ml *MemoryLimiter) IsMemoryAvailable(size int64) bool {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	if ml.limit <= 0 {
		return true
	}

	return ml.currentUsage+size <= ml.limit
}
