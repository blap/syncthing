// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// TestConnectionPoolManagement tests basic connection pool management functionality
func TestConnectionPoolManagement(t *testing.T) {
	// Given a connection pool manager
	manager := NewConnectionPoolManager(3, 10, 30*time.Minute)
	deviceID := protocol.LocalDeviceID
	
	// When we create a pool
	pool := manager.GetPool(deviceID)
	
	// Then we should be able to add and retrieve connections
	if pool == nil {
		t.Error("Expected pool to be created")
	}
	
	// Test adding a connection
	mockConn := NewEnhancedMockConnection("mock1", deviceID, 10, 50.0)
	added := pool.AddConnection(mockConn)
	if !added {
		t.Error("Expected connection to be added to pool")
	}
	
	// Test getting a connection
	retrieved := pool.GetConnection(RoundRobinStrategy)
	if retrieved == nil {
		t.Error("Expected connection to be retrieved from pool")
	}
	
	// Test round-robin behavior with multiple requests
	retrieved1 := pool.GetConnection(RoundRobinStrategy)
	retrieved2 := pool.GetConnection(RoundRobinStrategy)
	retrieved3 := pool.GetConnection(RoundRobinStrategy)
	
	// With only one connection, all requests should return the same connection
	if retrieved1 != retrieved2 || retrieved2 != retrieved3 {
		t.Error("Round-robin should return same connection when only one connection exists")
	}
	
	// Test returning connection to pool updates usage tracking
	pool.ReturnConnection(retrieved1)
}

// TestPoolAllocationStrategies tests different pool allocation strategies
func TestPoolAllocationStrategies(t *testing.T) {
	// Given a connection pool manager with different allocation strategies
	manager := NewConnectionPoolManager(3, 10, 30*time.Minute)
	deviceID := protocol.LocalDeviceID
	pool := manager.GetPool(deviceID)
	
	// Add some mock connections
	conn1 := NewEnhancedMockConnection("mock1", deviceID, 10, 30.0)	// Low health
	conn2 := NewEnhancedMockConnection("mock2", deviceID, 50, 60.0)	// Medium health
	conn3 := NewEnhancedMockConnection("mock3", deviceID, 90, 90.0)	// High health
	
	pool.AddConnection(conn1)
	pool.AddConnection(conn2)
	pool.AddConnection(conn3)
	
	// When we request connections using different strategies
	// Test round-robin strategy
	selected1 := pool.GetConnection(RoundRobinStrategy)
	if selected1 == nil {
		t.Error("Expected connection with round-robin strategy")
	}
	
	// Test health-based strategy (should select highest health)
	selected2 := pool.GetConnection(HealthBasedStrategy)
	if selected2 == nil {
		t.Error("Expected connection with health-based strategy")
	}
	
	// Test random strategy
	selected3 := pool.GetConnection(RandomStrategy)
	if selected3 == nil {
		t.Error("Expected connection with random strategy")
	}
	
	// Test least-used strategy
	selected4 := pool.GetConnection(LeastUsedStrategy)
	if selected4 == nil {
		t.Error("Expected connection with least-used strategy")
	}
	
	// Then we should get appropriate connections based on the strategy
	// For health-based strategy, we expect the highest health connection
	if selected2 != nil && selected2.ConnectionID() != "mock3" {
		t.Errorf("Expected health-based strategy to select mock3, got %s", selected2.ConnectionID())
	}
}

// TestResourceCleanup tests that resources are properly cleaned up
func TestResourceCleanup(t *testing.T) {
	// Given a connection pool manager with connections
	manager := NewConnectionPoolManager(3, 10, 30*time.Minute)
	deviceID := protocol.LocalDeviceID
	pool := manager.GetPool(deviceID)
	
	// Add some mock connections
	mockConn := NewEnhancedMockConnection("mock1", deviceID, 10, 50.0)
	pool.AddConnection(mockConn)
	
	// Test usage tracking
	pool.ReturnConnection(mockConn)
	
	// When connections expire or are explicitly closed
	// Test closing the pool
	pool.Close()
	
	// Then resources should be properly cleaned up
	// Try to get a connection - should return nil
	retrieved := pool.GetConnection(RoundRobinStrategy)
	if retrieved != nil {
		t.Error("Expected no connection after pool is closed")
	}
}

// TestPoolExhaustion tests handling of pool exhaustion scenarios
func TestPoolExhaustion(t *testing.T) {
	// Given a connection pool manager with a small pool size
	manager := NewConnectionPoolManager(2, 2, 30*time.Minute) // Max size of 2
	deviceID := protocol.LocalDeviceID
	pool := manager.GetPool(deviceID)
	
	// When we request more connections than the pool can provide
	conn1 := NewEnhancedMockConnection("mock1", deviceID, 10, 30.0)
	conn2 := NewEnhancedMockConnection("mock2", deviceID, 20, 50.0)
	conn3 := NewEnhancedMockConnection("mock3", deviceID, 30, 70.0) // This should not be added
	
	added1 := pool.AddConnection(conn1)
	added2 := pool.AddConnection(conn2)
	added3 := pool.AddConnection(conn3) // Should fail
	
	// Then appropriate handling should occur (blocking, error, or creating new connections)
	if !added1 || !added2 {
		t.Error("First two connections should be added successfully")
	}
	if added3 {
		t.Error("Third connection should not be added due to pool size limit")
	}
	
	// Test that we can still get connections from the pool
	retrieved := pool.GetConnection(RoundRobinStrategy)
	if retrieved == nil {
		t.Error("Expected to be able to retrieve connection from pool")
	}
}