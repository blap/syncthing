// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// TestCompleteEnhancedConnectionManagement tests all the enhanced connection management features together
func TestCompleteEnhancedConnectionManagement(t *testing.T) {
	// Setup
	deviceID := protocol.LocalDeviceID
	cfg := config.New(protocol.EmptyDeviceID)

	// Create all the enhanced components
	healthMonitor := NewHealthMonitorWithConfig(config.Wrap("/tmp/test-config.xml", cfg, protocol.EmptyDeviceID, nil), deviceID.String())
	packetScheduler := NewPacketScheduler()
	connectionPoolManager := NewConnectionPoolManager(3, 10, 30*time.Minute)
	connectionMigrationManager := NewConnectionMigrationManager()

	// Create mock connections with different qualities
	conn1 := NewEnhancedMockConnectionWithTrafficMetrics("conn1", deviceID, 10, 30.0, 50.0, 20.0, 20.0)  // Low quality
	conn2 := NewEnhancedMockConnectionWithTrafficMetrics("conn2", deviceID, 50, 60.0, 100.0, 1000.0, 10.0) // Medium quality
	conn3 := NewEnhancedMockConnectionWithTrafficMetrics("conn3", deviceID, 90, 90.0, 200.0, 2000.0, 5.0)  // High quality

	// Test 1: Health monitoring
	t.Run("HealthMonitoring", func(t *testing.T) {
		healthMonitor.Start()
		defer healthMonitor.Stop()

		// Record some metrics
		healthMonitor.RecordLatency(10 * time.Millisecond)
		healthMonitor.RecordPacketLoss(0.05) // 5% packet loss

		// Check that health score is updated
		score := healthMonitor.GetHealthScore()
		if score <= 0 || score > 100 {
			t.Errorf("Expected health score between 0 and 100, got %f", score)
		}

		// Test adaptive interval
		interval := healthMonitor.GetInterval()
		if interval <= 0 {
			t.Error("Expected adaptive interval to be positive")
		}

		// Test connection quality metrics
		metrics := healthMonitor.GetConnectionQualityMetrics()
		if metrics["latencyMs"] <= 0 {
			t.Error("Expected positive latency metric")
		}
	})

	// Test 2: Packet scheduling with traffic analysis
	t.Run("PacketScheduling", func(t *testing.T) {
		// Add connections to scheduler
		packetScheduler.AddConnection(deviceID, conn1)
		packetScheduler.AddConnection(deviceID, conn2)
		packetScheduler.AddConnection(deviceID, conn3)

		// Test selection based on health
		selected := packetScheduler.SelectConnection(deviceID)
		if selected == nil {
			t.Error("Expected a connection to be selected")
		}

		// Test selection based on traffic metrics
		selectedTraffic := packetScheduler.SelectConnectionBasedOnTraffic(deviceID)
		if selectedTraffic == nil {
			t.Error("Expected a connection to be selected based on traffic")
		}

		// Test load balancing
		distribution := make(map[string]int)
		for i := 0; i < 100; i++ {
			selected := packetScheduler.SelectConnectionForLoadBalancing(deviceID)
			if selected != nil {
				distribution[selected.ConnectionID()]++
			}
		}

		// All connections should have received some packets
		if len(distribution) != 3 {
			t.Errorf("Expected all 3 connections to receive packets, got %d", len(distribution))
		}
	})

	// Test 3: Bandwidth aggregation
	t.Run("BandwidthAggregation", func(t *testing.T) {
		// Simulate data transfer on connections
		conn1.SimulateDataTransfer(1000000, 500000)  // 1MB out, 0.5MB in
		conn2.SimulateDataTransfer(2000000, 1000000) // 2MB out, 1MB in
		conn3.SimulateDataTransfer(3000000, 1500000) // 3MB out, 1.5MB in

		// Test aggregated bandwidth calculation
		aggregatedBandwidth := packetScheduler.GetAggregatedBandwidth(deviceID)
		if aggregatedBandwidth <= 0 {
			t.Error("Expected positive aggregated bandwidth")
		}

		// Test individual connection bandwidth
		conn1Bandwidth := packetScheduler.GetConnectionBandwidth(deviceID, conn1.ConnectionID())
		conn2Bandwidth := packetScheduler.GetConnectionBandwidth(deviceID, conn2.ConnectionID())
		conn3Bandwidth := packetScheduler.GetConnectionBandwidth(deviceID, conn3.ConnectionID())

		if conn1Bandwidth <= 0 || conn2Bandwidth <= 0 || conn3Bandwidth <= 0 {
			t.Error("Expected positive bandwidth for all connections")
		}

		// Test data chunk distribution
		chunkSize := int64(1024 * 1024) // 1MB chunks
		distributedChunks := packetScheduler.DistributeDataChunks(deviceID, chunkSize)
		if len(distributedChunks) == 0 {
			t.Error("Expected data chunks to be distributed")
		}
	})

	// Test 4: Connection pooling
	t.Run("ConnectionPooling", func(t *testing.T) {
		// Get pool for device
		pool := connectionPoolManager.GetPool(deviceID)

		// Add connections to pool
		added1 := pool.AddConnection(conn1)
		added2 := pool.AddConnection(conn2)
		added3 := pool.AddConnection(conn3)

		if !added1 || !added2 || !added3 {
			t.Error("Expected all connections to be added to pool")
		}

		// Check that we have connections in the pool
		if pool.connections == nil || len(pool.connections) != 3 {
			t.Errorf("Expected 3 connections in pool, got %d", len(pool.connections))
		}

		// Test different allocation strategies
		roundRobin := pool.GetConnection(RoundRobinStrategy)
		healthBased := pool.GetConnection(HealthBasedStrategy)
		random := pool.GetConnection(RandomStrategy)
		leastUsed := pool.GetConnection(LeastUsedStrategy)

		// Debug output
		t.Logf("RoundRobin: %v, HealthBased: %v, Random: %v, LeastUsed: %v", 
			roundRobin != nil, healthBased != nil, random != nil, leastUsed != nil)

		if roundRobin == nil {
			t.Error("RoundRobin strategy returned nil")
		}
		if healthBased == nil {
			t.Error("HealthBased strategy returned nil")
		}
		if random == nil {
			t.Error("Random strategy returned nil")
		}
		if leastUsed == nil {
			t.Error("LeastUsed strategy returned nil")
		}

		// Test returning connections
		pool.ReturnConnection(conn1)
		pool.ReturnConnection(conn2)
		pool.ReturnConnection(conn3)
	})

	// Test 5: Connection migration
	t.Run("ConnectionMigration", func(t *testing.T) {
		// Register a transfer
		connectionMigrationManager.RegisterTransfer(conn1, "default", "test-file", 1024*1024, 1024)

		// Update transfer progress
		connectionMigrationManager.UpdateTransferProgress(conn1, "default", "test-file", 512*1024, 500)

		// Add a pending request
		connectionMigrationManager.AddPendingRequest(conn1, "default", "test-file", 1, 500, 512*1024, 1024, []byte("hash"))

		// Get transfer state
		state, exists := connectionMigrationManager.GetTransferState(conn1, "default", "test-file")
		if !exists {
			t.Error("Expected transfer state to exist")
		}
		if state == nil {
			t.Error("Expected transfer state to be non-nil")
		}

		// Test migration decision
		// Create a simple service interface for testing
		service := &mockService{connections: []protocol.Connection{conn1, conn2, conn3}}
		_ = connectionMigrationManager.ShouldMigrateTransfer(conn1, service, "default", "test-file")
		// This might be true or false depending on the specific conditions

		// Test getting best connection for transfer
		bestConn := connectionMigrationManager.GetBestConnectionForTransfer(deviceID, service, "default", "test-file")
		if bestConn == nil {
			t.Error("Expected best connection to be found")
		}
	})

	// Test 6: Lazy health monitoring
	t.Run("LazyHealthMonitoring", func(t *testing.T) {
		// Test health monitoring functionality
		initialScore := healthMonitor.GetHealthScore()
		if initialScore <= 0 || initialScore > 100 {
			t.Errorf("Expected initial health score between 0 and 100, got %f", initialScore)
		}

		// Record some metrics
		healthMonitor.RecordLatency(10 * time.Millisecond)
		healthMonitor.RecordPacketLoss(0.05) // 5% packet loss

		// Check that health score is updated
		score := healthMonitor.GetHealthScore()
		if score <= 0 || score > 100 {
			t.Errorf("Expected health score between 0 and 100, got %f", score)
		}

		// Test adaptive interval
		interval := healthMonitor.GetInterval()
		if interval <= 0 {
			t.Error("Expected adaptive interval to be positive")
		}

		// Test connection quality metrics
		metrics := healthMonitor.GetConnectionQualityMetrics()
		if metrics["latencyMs"] <= 0 {
			t.Error("Expected positive latency metric")
		}
	})

	// Cleanup
	connectionPoolManager.CloseAllPools()
}

// mockService implements the interface needed for testing
type mockService struct {
	connections []protocol.Connection
}

// GetConnectionsForDevice returns the connections for a device
func (m *mockService) GetConnectionsForDevice(deviceID protocol.DeviceID) []protocol.Connection {
	return m.connections
}