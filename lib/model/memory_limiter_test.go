// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"testing"
)

func TestMemoryLimiter_RequestMemory(t *testing.T) {
	// Create a new memory limiter
	ml := NewMemoryLimiter()

	// Test requesting memory with no limit
	if !ml.RequestMemory("component1", 100) {
		t.Error("RequestMemory should succeed with no limit")
	}

	// Set a limit
	ml.SetLimit(200)

	// Test requesting memory within limit
	if !ml.RequestMemory("component2", 50) {
		t.Error("RequestMemory should succeed within limit")
	}

	// Test requesting memory that would exceed limit
	if ml.RequestMemory("component3", 200) {
		t.Error("RequestMemory should fail when exceeding limit")
	}

	// Test requesting memory that fits within remaining limit
	// We've used 50 so far, so we have 150 remaining
	if !ml.RequestMemory("component4", 100) {
		t.Error("RequestMemory should succeed when within remaining limit")
	}
}

func TestMemoryLimiter_ReleaseMemory(t *testing.T) {
	// Create a new memory limiter with a limit
	ml := NewMemoryLimiter()
	ml.SetLimit(200)

	// Request some memory
	if !ml.RequestMemory("component1", 100) {
		t.Fatal("RequestMemory should succeed")
	}

	// Check current usage
	if ml.GetCurrentUsage() != 100 {
		t.Errorf("Expected current usage to be 100, got %d", ml.GetCurrentUsage())
	}

	// Release some memory
	ml.ReleaseMemory("component1", 50)

	// Check current usage
	if ml.GetCurrentUsage() != 50 {
		t.Errorf("Expected current usage to be 50, got %d", ml.GetCurrentUsage())
	}

	// Release remaining memory
	ml.ReleaseMemory("component1", 50)

	// Check current usage
	if ml.GetCurrentUsage() != 0 {
		t.Errorf("Expected current usage to be 0, got %d", ml.GetCurrentUsage())
	}
}

func TestMemoryLimiter_GetComponentUsage(t *testing.T) {
	// Create a new memory limiter
	ml := NewMemoryLimiter()

	// Request memory for components
	ml.RequestMemory("component1", 100)
	ml.RequestMemory("component2", 200)

	// Check component usage
	if ml.GetComponentUsage("component1") != 100 {
		t.Errorf("Expected component1 usage to be 100, got %d", ml.GetComponentUsage("component1"))
	}

	if ml.GetComponentUsage("component2") != 200 {
		t.Errorf("Expected component2 usage to be 200, got %d", ml.GetComponentUsage("component2"))
	}

	// Check usage for non-existent component
	if ml.GetComponentUsage("component3") != 0 {
		t.Errorf("Expected component3 usage to be 0, got %d", ml.GetComponentUsage("component3"))
	}
}

func TestMemoryLimiter_GetComponents(t *testing.T) {
	// Create a new memory limiter
	ml := NewMemoryLimiter()

	// Request memory for components
	ml.RequestMemory("component1", 100)
	ml.RequestMemory("component2", 200)

	// Get all components
	components := ml.GetComponents()

	// Check that we have the right components
	if len(components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(components))
	}

	if components["component1"] != 100 {
		t.Errorf("Expected component1 usage to be 100, got %d", components["component1"])
	}

	if components["component2"] != 200 {
		t.Errorf("Expected component2 usage to be 200, got %d", components["component2"])
	}
}

func TestMemoryLimiter_IsMemoryAvailable(t *testing.T) {
	// Create a new memory limiter with a limit
	ml := NewMemoryLimiter()
	ml.SetLimit(200)

	// Test with no usage
	if !ml.IsMemoryAvailable(100) {
		t.Error("Memory should be available when no usage")
	}

	// Request some memory
	ml.RequestMemory("component1", 100)

	// Test with available memory
	if !ml.IsMemoryAvailable(50) {
		t.Error("Memory should be available when within limit")
	}

	// Test with insufficient memory
	if ml.IsMemoryAvailable(150) {
		t.Error("Memory should not be available when exceeding limit")
	}

	// Test with no limit
	ml.SetLimit(0)
	if !ml.IsMemoryAvailable(1000) {
		t.Error("Memory should always be available with no limit")
	}
}

func TestMemoryLimiter_SetLimit(t *testing.T) {
	// Create a new memory limiter
	ml := NewMemoryLimiter()

	// Check initial limit (should be 0 for no limit)
	if ml.GetLimit() != 0 {
		t.Errorf("Expected initial limit to be 0, got %d", ml.GetLimit())
	}

	// Set a limit
	ml.SetLimit(1000)

	// Check the limit was set
	if ml.GetLimit() != 1000 {
		t.Errorf("Expected limit to be 1000, got %d", ml.GetLimit())
	}

	// Set limit to 0 (no limit)
	ml.SetLimit(0)

	// Check the limit was set to 0
	if ml.GetLimit() != 0 {
		t.Errorf("Expected limit to be 0, got %d", ml.GetLimit())
	}
}
