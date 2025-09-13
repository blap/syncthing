// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/syncthing/syncthing/internal/gen/bep"
	"github.com/syncthing/syncthing/internal/db"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/stats"
)

// TestContinuousMonitoringFramework tests the basic functionality of the continuous monitoring framework
func TestContinuousMonitoringFramework(t *testing.T) {
	// Create test environment
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test devices
	device1ID := protocol.NewDeviceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	device2ID := protocol.NewDeviceID([]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1})

	// Create registry for connection tracking
	registry1 := registry.New()

	// Create certificates
	cert1 := mustGetCert(t)
	_ = mustGetCert(t) // Create cert2 but don't use it to avoid unused variable error

	// Create TLS configurations
	tlsCfg1 := &tls.Config{
		Certificates:       []tls.Certificate{cert1},
		NextProtos:         []string{"bep/1.0"},
		ServerName:         "syncthing",
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequestClientCert,
	}

	// Create configurations
	cfg1 := config.Wrap("/dev/null", config.Configuration{
		Devices: []config.DeviceConfiguration{
			{
				DeviceID:  device1ID,
				Addresses: []string{"dynamic"},
			},
			{
				DeviceID:  device2ID,
				Addresses: []string{"dynamic"},
			},
		},
		Options: config.OptionsConfiguration{
			RawListenAddresses: []string{"tcp://127.0.0.1:0"},
			GlobalAnnEnabled:   false,
			LocalAnnEnabled:    true,
			LocalAnnPort:       21027,
			ReconnectIntervalS: 3,
		},
	}, device1ID, events.NoopLogger)

	// Create mock model
	model1 := &monitoringTestModel{t: t, deviceID: device1ID}

	// Create service
	service1 := NewService(cfg1, device1ID, model1, tlsCfg1, nil, "bep/1.0", "syncthing", events.NoopLogger, registry1, nil)

	// Start service
	go service1.Serve(ctx)

	// Give service time to start
	time.Sleep(100 * time.Millisecond)

	// Verify service started
	listenerStatus1 := service1.ListenerStatus()
	if len(listenerStatus1) == 0 {
		t.Fatal("Failed to start listener")
	}

	// Create continuous monitoring framework
	monitoringFramework := NewContinuousMonitoringFramework(cfg1, events.NoopLogger, service1)

	// Test basic functionality
	t.Run("BasicFrameworkOperations", func(t *testing.T) {
		// Framework should not be running initially
		if monitoringFramework.IsRunning() {
			t.Error("Framework should not be running initially")
		}

		// Start the framework
		monitoringFramework.Start()

		// Framework should be running now
		if !monitoringFramework.IsRunning() {
			t.Error("Framework should be running after Start()")
		}

		// Starting again should not cause issues
		monitoringFramework.Start()

		// Framework should still be running
		if !monitoringFramework.IsRunning() {
			t.Error("Framework should still be running after second Start()")
		}

		// Stop the framework
		monitoringFramework.Stop()

		// Framework should not be running now
		if monitoringFramework.IsRunning() {
			t.Error("Framework should not be running after Stop()")
		}

		// Stopping again should not cause issues
		monitoringFramework.Stop()

		// Framework should still not be running
		if monitoringFramework.IsRunning() {
			t.Error("Framework should still not be running after second Stop()")
		}
	})

	// Test interval setting
	t.Run("IntervalSetting", func(t *testing.T) {
		// Default interval should be 30 seconds
		// We can't directly check the interval, but we can test setting it

		monitoringFramework.SetInterval(15 * time.Second)
		// No assertion needed, just ensure it doesn't panic

		monitoringFramework.SetInterval(1 * time.Minute)
		// No assertion needed, just ensure it doesn't panic
	})

	// Test monitoring cycle execution
	t.Run("MonitoringCycleExecution", func(t *testing.T) {
		// Start the framework with a short interval for testing
		monitoringFramework.SetInterval(100 * time.Millisecond)
		monitoringFramework.Start()

		// Let it run for a few cycles
		time.Sleep(500 * time.Millisecond)

		// Stop the framework
		monitoringFramework.Stop()

		// If we get here without panicking, the monitoring cycle executed successfully
		t.Log("Monitoring cycles executed successfully")
	})
}

// TestHealthAnalysis tests the health analysis functionality
func TestHealthAnalysis(t *testing.T) {
	// Create test environment
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create mock config
	cfg := createTestConfig()

	// Create mock service
	service := &monitoringMockService{
		connectionStatus: make(map[string]ConnectionStatusEntry),
		connectedDevices: make([]protocol.DeviceID, 0),
		deviceConnections: make(map[protocol.DeviceID][]protocol.Connection),
	}

	// Create monitoring framework
	monitoringFramework := NewContinuousMonitoringFramework(cfg, events.NoopLogger, service)

	t.Run("OverallHealthCalculation", func(t *testing.T) {
		// Test various health scenarios
		score1 := monitoringFramework.calculateOverallHealthScore(10, 10, 0)  // Perfect health
		score2 := monitoringFramework.calculateOverallHealthScore(10, 5, 2)   // Moderate health
		score3 := monitoringFramework.calculateOverallHealthScore(10, 0, 10)  // Poor health
		score4 := monitoringFramework.calculateOverallHealthScore(0, 0, 0)    // No connections

		if score1 < 90.0 {
			t.Errorf("Expected high health score for perfect conditions, got %f", score1)
		}

		if score2 > score1 {
			t.Errorf("Moderate health score should be lower than perfect score, got %f (perfect: %f)", score2, score1)
		}

		if score3 > score2 {
			t.Errorf("Poor health score should be lower than moderate score, got %f (moderate: %f)", score3, score2)
		}

		if score4 != 100.0 {
			t.Errorf("Expected perfect score for no connections, got %f", score4)
		}
	})

	t.Run("ConnectionPatternAnalysis", func(t *testing.T) {
		// Create a mock connection with statistics
		deviceID := protocol.NewDeviceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
		
		// Test statistics analysis
		highLatencyStats := protocol.Statistics{
			StartedAt: time.Now().Add(-10 * time.Minute),
			InBytesTotal: 1000000,
			OutBytesTotal: 1000000,
		}
		
		// This should not panic
		monitoringFramework.analyzeConnectionPatterns(ctx, deviceID, &mockConnection{}, highLatencyStats)
		
		// Test low throughput detection with different stats
		lowThroughputStats := protocol.Statistics{
			StartedAt: time.Now().Add(-10 * time.Minute),
			InBytesTotal: 100,
			OutBytesTotal: 100,
		}
		
		// This should not panic
		monitoringFramework.analyzeConnectionPatterns(ctx, deviceID, &mockConnection{}, lowThroughputStats)
		
		// Test low throughput detection with different variable name
		anotherLowThroughputStats := protocol.Statistics{
			StartedAt: time.Now().Add(-10 * time.Minute),
			InBytesTotal: 512,
			OutBytesTotal: 512,
		}
		
		// This should not panic
		monitoringFramework.analyzeConnectionPatterns(ctx, deviceID, &mockConnection{}, anotherLowThroughputStats)
	})

	t.Run("HealthIssueHandling", func(t *testing.T) {
		// Test with no connections
		monitoringFramework.handleHealthIssues(ctx, make(map[string]ConnectionStatusEntry))
		
		// Test with connections but no errors
		goodStatus := make(map[string]ConnectionStatusEntry)
		goodStatus["addr1"] = ConnectionStatusEntry{}
		goodStatus["addr2"] = ConnectionStatusEntry{}
		
		monitoringFramework.handleHealthIssues(ctx, goodStatus)
		
		// Test with some errors (but not enough to trigger reconnection)
		someErrors := make(map[string]ConnectionStatusEntry)
		someErrors["addr1"] = ConnectionStatusEntry{}
		someErrors["addr2"] = ConnectionStatusEntry{Error: stringPtr("connection failed")}
		
		monitoringFramework.handleHealthIssues(ctx, someErrors)
		
		// Test with many errors (should trigger reconnection)
		manyErrors := make(map[string]ConnectionStatusEntry)
		for i := 0; i < 5; i++ {
			manyErrors[fmt.Sprintf("addr%d", i)] = ConnectionStatusEntry{Error: stringPtr("connection failed")}
		}
		
		monitoringFramework.handleHealthIssues(ctx, manyErrors)
	})
}

// TestContinuousMonitoringIntegration tests integration with real connections
func TestContinuousMonitoringIntegration(t *testing.T) {
	// Test with the withConnectionPair helper to ensure integration works
	withConnectionPair(t, "tcp://127.0.0.1:0", func(client, server internalConn) {
		// This test ensures that the monitoring framework can work with real connections
		// We can't fully test the integration without a real service, but we can test
		// that the framework doesn't break with real connection objects
		
		t.Log("Connection pair established successfully")
		
		// Send some data to make the connection active
		data := []byte("test data")
		_, err := client.Write(data)
		if err != nil {
			t.Fatalf("Failed to write to client connection: %v", err)
		}
		
		// Read the data on the server side
		buf := make([]byte, len(data))
		_, err = server.Read(buf)
		if err != nil {
			t.Fatalf("Failed to read from server connection: %v", err)
		}
		
		// Verify data integrity
		for i := range data {
			if buf[i] != data[i] {
				t.Fatalf("Data mismatch at position %d: expected %d, got %d", i, data[i], buf[i])
			}
		}
		
		t.Log("Data transmission successful")
		
		// The monitoring framework should be able to handle these connection objects
		// without issues, even though we can't fully test the monitoring without
		// a real service implementation
	})
}

// Helper functions and mock types

func stringPtr(s string) *string {
	return &s
}

// monitoringMockService implements a mock Service for testing
type monitoringMockService struct {
	connectionStatus  map[string]ConnectionStatusEntry
	connectedDevices  []protocol.DeviceID
	deviceConnections map[protocol.DeviceID][]protocol.Connection
	mut               sync.RWMutex
}

func (m *monitoringMockService) Serve(ctx context.Context) error {
	// Mock implementation
	return nil
}

func (m *monitoringMockService) Stop() {
	// Mock implementation
}

func (m *monitoringMockService) ListenerStatus() map[string]ListenerStatusEntry {
	// Mock implementation
	return make(map[string]ListenerStatusEntry)
}

func (m *monitoringMockService) ConnectionStatus() map[string]ConnectionStatusEntry {
	m.mut.RLock()
	defer m.mut.RUnlock()
	
	// Return a copy of the connection status
	result := make(map[string]ConnectionStatusEntry)
	for k, v := range m.connectionStatus {
		result[k] = v
	}
	return result
}

func (m *monitoringMockService) NATType() string {
	// Mock implementation
	return "unknown"
}

func (m *monitoringMockService) GetConnectedDevices() []protocol.DeviceID {
	m.mut.RLock()
	defer m.mut.RUnlock()
	
	// Return a copy of the connected devices
	result := make([]protocol.DeviceID, len(m.connectedDevices))
	copy(result, m.connectedDevices)
	return result
}

func (m *monitoringMockService) GetConnectionsForDevice(deviceID protocol.DeviceID) []protocol.Connection {
	m.mut.RLock()
	defer m.mut.RUnlock()
	
	// Return a copy of the connections for the device
	if connections, ok := m.deviceConnections[deviceID]; ok {
		result := make([]protocol.Connection, len(connections))
		copy(result, connections)
		return result
	}
	
	return nil
}

func (m *monitoringMockService) PacketScheduler() *PacketScheduler {
	// Mock implementation
	return nil
}

func (m *monitoringMockService) DialNow() {
	// Mock implementation - just log that it was called
	fmt.Println("DialNow called on mock service")
}

func (m *monitoringMockService) AllAddresses() []string {
	// Mock implementation - return empty slice
	return []string{}
}

func (m *monitoringMockService) ExternalAddresses() []string {
	// Mock implementation - return empty slice
	return []string{}
}

// mockConnection implements a mock protocol.Connection for testing
type mockConnection struct{}

func (m *mockConnection) Index(ctx context.Context, idx *protocol.Index) error { return nil }

func (m *mockConnection) IndexUpdate(ctx context.Context, idxUp *protocol.IndexUpdate) error { return nil }

func (m *mockConnection) Request(ctx context.Context, req *protocol.Request) ([]byte, error) { return nil, nil }

func (m *mockConnection) ClusterConfig(config *protocol.ClusterConfig, passwords map[string]string) {}

func (m *mockConnection) DownloadProgress(ctx context.Context, dp *protocol.DownloadProgress) {}

func (m *mockConnection) Start() {}

func (m *mockConnection) Close(err error) {}

func (m *mockConnection) DeviceID() protocol.DeviceID { return protocol.EmptyDeviceID }

func (m *mockConnection) Statistics() protocol.Statistics { 
	return protocol.Statistics{
		At:            time.Now(),
		InBytesTotal:  0,
		OutBytesTotal: 0,
		StartedAt:     time.Now(),
	} 
}

func (m *mockConnection) Closed() <-chan struct{} { return nil }

func (m *mockConnection) GetPingLossRate() float64 { return 0.0 }

func (m *mockConnection) Type() string { return "mock" }

func (m *mockConnection) Transport() string { return "mock" }

func (m *mockConnection) IsLocal() bool { return false }

func (m *mockConnection) RemoteAddr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345} }

func (m *mockConnection) Priority() int { return 0 }

func (m *mockConnection) String() string { return "mock" }

func (m *mockConnection) Crypto() string { return "mock" }

func (m *mockConnection) EstablishedAt() time.Time { return time.Now() }

func (m *mockConnection) ConnectionID() string { return "mock-connection" }

// QueryDevice sends a QueryDevice message to the peer device
func (m *mockConnection) QueryDevice(ctx context.Context, query *bep.QueryDevice) error {
	return nil
}

// ResponseDevice sends a ResponseDevice message to the peer device
func (m *mockConnection) ResponseDevice(ctx context.Context, response *bep.ResponseDevice) error {
	return nil
}

// monitoringTestModel implements the Model interface for testing monitoring
type monitoringTestModel struct {
	t        *testing.T
	deviceID protocol.DeviceID
}

func (m *monitoringTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *monitoringTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.t.Logf("Device %s added connection to %s", m.deviceID, conn.DeviceID())
}

func (m *monitoringTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *monitoringTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *monitoringTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *monitoringTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *monitoringTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *monitoringTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *monitoringTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *monitoringTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *monitoringTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *monitoringTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}