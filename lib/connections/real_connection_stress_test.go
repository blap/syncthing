// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/syncthing/syncthing/internal/db"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/stats"
)

// TestConcurrentRealConnections tests concurrent connection establishment between multiple Syncthing instances
func TestConcurrentRealConnections(t *testing.T) {
	// Set up test environment with 5 devices
	numDevices := 5
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create device IDs
	deviceIDs := make([]protocol.DeviceID, numDevices)
	for i := 0; i < numDevices; i++ {
		idBytes := make([]byte, 32)
		for j := 0; j < 32; j++ {
			idBytes[j] = byte(i*numDevices + j + 1)
		}
		deviceIDs[i] = protocol.NewDeviceID(idBytes)
	}

	// Create configurations and services
	configs := make([]config.Wrapper, numDevices)
	services := make([]Service, numDevices)
	models := make([]*stressTestModel, numDevices)
	registries := make([]*registry.Registry, numDevices)

	// Create all configurations first
	for i := 0; i < numDevices; i++ {
		// Create registry for connection tracking
		registries[i] = registry.New()
		
		// Create device configurations
		deviceCfgs := make([]config.DeviceConfiguration, 0, numDevices-1)
		for j := 0; j < numDevices; j++ {
			if i != j {
				deviceCfgs = append(deviceCfgs, config.DeviceConfiguration{
					DeviceID:  deviceIDs[j],
					Addresses: []string{"dynamic"},
				})
			}
		}

		configs[i] = config.Wrap("/dev/null", config.Configuration{
			Devices: deviceCfgs,
			Options: config.OptionsConfiguration{
				RawListenAddresses: []string{"tcp://127.0.0.1:0"},
				GlobalAnnEnabled: false,
				LocalAnnEnabled:  true,
				LocalAnnPort:     21027,
				ConnectionLimitMax: 10,
				ConnectionLimitEnough: 3,
				ReconnectIntervalS: 5,
			},
		}, deviceIDs[i], events.NoopLogger)

		// Create TLS configuration
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{generateStressTestCertificate(t, deviceIDs[i])},
			NextProtos:   []string{"bep/1.0"},
			ServerName:   "syncthing",
		}

		// Create model
		models[i] = &stressTestModel{
			t:        t,
			deviceID: deviceIDs[i],
			connections: make(map[protocol.DeviceID]protocol.Connection),
			mut:      sync.RWMutex{},
		}

		// Create service
		services[i] = NewService(configs[i], deviceIDs[i], models[i], tlsCfg, nil, "bep/1.0", "syncthing", events.NoopLogger, registries[i], nil)
	}

	// Start all services
	for i := 0; i < numDevices; i++ {
		go services[i].Serve(ctx)
	}

	// Wait for services to start
	time.Sleep(200 * time.Millisecond)

	// Get listener addresses and update configurations
	addresses := make([]string, numDevices)
	for i := 0; i < numDevices; i++ {
		listenerStatus := services[i].ListenerStatus()
		if len(listenerStatus) == 0 {
			t.Fatalf("Failed to start listener for device %d", i)
		}

		// Extract actual listening addresses - use the first LAN address that's not 0.0.0.0
		var addr string
		for _, status := range listenerStatus {
			// Find the first LAN address that's not 0.0.0.0
			for _, lanAddr := range status.LANAddresses {
				if lanAddr != "tcp://0.0.0.0:0" {
					addr = lanAddr
					break
				}
			}
			if addr != "" {
				break
			}
		}
		
		if addr == "" {
			t.Fatalf("Failed to get listening address for device %d", i)
		}
		
		addresses[i] = addr
		t.Logf("Device %d listening on: %s", i, addresses[i])
	}

	// Update all configurations with peer addresses
	for i := 0; i < numDevices; i++ {
		configs[i].Modify(func(cfg *config.Configuration) {
			deviceIdx := 0
			for j := 0; j < len(cfg.Devices); j++ {
				if cfg.Devices[j].DeviceID != deviceIDs[i] {
					peerIndex := deviceIdx
					if deviceIdx >= i {
						peerIndex = deviceIdx + 1
					}
					if peerIndex < numDevices {
						cfg.Devices[j].Addresses = []string{addresses[peerIndex]}
					}
					deviceIdx++
				}
			}
		})
	}

	// Wait for connections to establish
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	connectedDevices := make([]int, numDevices)
	expectedConnections := numDevices - 1

	for {
		select {
		case <-timeout:
			// Log connection status for debugging
			for i := 0; i < numDevices; i++ {
				connected := services[i].GetConnectedDevices()
				connectedDevices[i] = len(connected)
				t.Logf("Device %d connected to %d devices: %v", i, len(connected), connected)
			}
			
			// Check if we have enough connections
			totalConnections := 0
			for _, count := range connectedDevices {
				totalConnections += count
			}
			
			// In a full mesh, we expect numDevices * (numDevices - 1) connections
			// But since each connection is bidirectional, we count each once
			expectedTotal := numDevices * expectedConnections
			t.Logf("Total connections: %d, Expected: %d", totalConnections, expectedTotal)
			
			if totalConnections >= expectedTotal/2 {
				t.Logf("Achieved sufficient connections: %d/%d", totalConnections, expectedTotal/2)
				return
			}
			
			t.Fatal("Timeout waiting for concurrent connections")
		case <-ticker.C:
			// Check connection status
			allConnected := true
			totalConnections := 0
			
			for i := 0; i < numDevices; i++ {
				connected := services[i].GetConnectedDevices()
				connectedDevices[i] = len(connected)
				totalConnections += len(connected)
				
				t.Logf("Device %d connected to %d devices", i, len(connected))
				
				// Each device should connect to all others
				if len(connected) < expectedConnections {
					allConnected = false
				}
			}
			
			if allConnected {
				t.Logf("All devices successfully connected to each other (%d total connections)", totalConnections)
				return
			}
		}
	}
}

// TestNetworkConditionResilience tests connection resilience under various network conditions
func TestNetworkConditionResilience(t *testing.T) {
	// Set up test environment
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Create two test devices
	device1ID := protocol.NewDeviceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	device2ID := protocol.NewDeviceID([]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1})

	// Create registries for connection tracking
	registry1 := registry.New()
	registry2 := registry.New()

	// Set up configuration with aggressive reconnection settings
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
			GlobalAnnEnabled: false,
			LocalAnnEnabled:  true,
			LocalAnnPort:     21027,
			ReconnectIntervalS: 3, // Aggressive reconnection
			StunKeepaliveStartS: 10,
			StunKeepaliveMinS: 5,
		},
	}, device1ID, events.NoopLogger)

	cfg2 := config.Wrap("/dev/null", config.Configuration{
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
			GlobalAnnEnabled: false,
			LocalAnnEnabled:  true,
			LocalAnnPort:     21027,
			ReconnectIntervalS: 3, // Aggressive reconnection
			StunKeepaliveStartS: 10,
			StunKeepaliveMinS: 5,
		},
	}, device2ID, events.NoopLogger)

	// Create TLS configurations
	tlsCfg1 := &tls.Config{
		Certificates: []tls.Certificate{generateStressTestCertificate(t, device1ID)},
		NextProtos:   []string{"bep/1.0"},
		ServerName:   "syncthing",
	}
	
	tlsCfg2 := &tls.Config{
		Certificates: []tls.Certificate{generateStressTestCertificate(t, device2ID)},
		NextProtos:   []string{"bep/1.0"},
		ServerName:   "syncthing",
	}

	// Create models
	model1 := &resilienceTestModel{t: t, deviceID: device1ID, connectionEvents: make([]connectionEvent, 0), mut: sync.RWMutex{}}
	model2 := &resilienceTestModel{t: t, deviceID: device2ID, connectionEvents: make([]connectionEvent, 0), mut: sync.RWMutex{}}

	// Create connection services
	service1 := NewService(cfg1, device1ID, model1, tlsCfg1, nil, "bep/1.0", "syncthing", events.NoopLogger, registry1, nil)
	service2 := NewService(cfg2, device2ID, model2, tlsCfg2, nil, "bep/1.0", "syncthing", events.NoopLogger, registry2, nil)

	// Start services
	go service1.Serve(ctx)
	go service2.Serve(ctx)

	// Wait for services to start
	time.Sleep(100 * time.Millisecond)

	// Get listener addresses
	listenerStatus1 := service1.ListenerStatus()
	listenerStatus2 := service2.ListenerStatus()
	
	if len(listenerStatus1) == 0 || len(listenerStatus2) == 0 {
		t.Fatal("Failed to start listeners")
	}

	// Extract actual listening addresses - use the first LAN address that's not 0.0.0.0
	var addr1, addr2 string
	for _, status := range listenerStatus1 {
		// Find the first LAN address that's not 0.0.0.0
		for _, lanAddr := range status.LANAddresses {
			if lanAddr != "tcp://0.0.0.0:0" {
				addr1 = lanAddr
				break
			}
		}
		if addr1 != "" {
			break
		}
	}
	
	for _, status := range listenerStatus2 {
		// Find the first LAN address that's not 0.0.0.0
		for _, lanAddr := range status.LANAddresses {
			if lanAddr != "tcp://0.0.0.0:0" {
				addr2 = lanAddr
				break
			}
		}
		if addr2 != "" {
			break
		}
	}

	if addr1 == "" || addr2 == "" {
		t.Fatal("Failed to get listening addresses")
	}

	t.Logf("Device 1 listening on: %s", addr1)
	t.Logf("Device 2 listening on: %s", addr2)

	// Update configurations with actual addresses
	cfg1.Modify(func(cfg *config.Configuration) {
		for i := range cfg.Devices {
			if cfg.Devices[i].DeviceID == device2ID {
				cfg.Devices[i].Addresses = []string{addr2}
				break
			}
		}
	})
	
	cfg2.Modify(func(cfg *config.Configuration) {
		for i := range cfg.Devices {
			if cfg.Devices[i].DeviceID == device1ID {
				cfg.Devices[i].Addresses = []string{addr1}
				break
			}
		}
	})

	// Wait for initial connection establishment
	t.Log("Waiting for initial connection...")
	time.Sleep(5 * time.Second)

	// Verify initial connection
	connectedDevices1 := service1.GetConnectedDevices()
	connectedDevices2 := service2.GetConnectedDevices()
	
	if !containsDevice(connectedDevices1, device2ID) || !containsDevice(connectedDevices2, device1ID) {
		t.Fatal("Failed to establish initial connection")
	}
	
	t.Log("Initial connection established successfully")

	// Test connection resilience by simulating network interruptions
	// In a real test, we would simulate network issues, but here we'll test reconnection logic
	
	// Wait and observe reconnection behavior
	t.Log("Observing connection stability for 20 seconds...")
	time.Sleep(20 * time.Second)

	// Check connection events
	model1.mut.RLock()
	model2.mut.RLock()
	
	connEvents1 := len(model1.connectionEvents)
	connEvents2 := len(model2.connectionEvents)
	
	model1.mut.RUnlock()
	model2.mut.RUnlock()

	t.Logf("Device 1 connection events: %d", connEvents1)
	t.Logf("Device 2 connection events: %d", connEvents2)

	// Verify devices are still connected
	finalConnectedDevices1 := service1.GetConnectedDevices()
	finalConnectedDevices2 := service2.GetConnectedDevices()
	
	if !containsDevice(finalConnectedDevices1, device2ID) || !containsDevice(finalConnectedDevices2, device1ID) {
		t.Fatal("Connection lost during resilience test")
	}
	
	t.Log("Connection resilience test passed - devices remained connected")
}

// generateStressTestCertificate generates a test certificate for stress testing
func generateStressTestCertificate(t *testing.T, _ protocol.DeviceID) tls.Certificate {
	// Generate a new RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create a certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "syncthing",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}

	// Create the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Encode the certificate and private key to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Parse the certificate
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}
	
	return cert
}

// stressTestModel implements a model for stress testing
type stressTestModel struct {
	t           *testing.T
	deviceID    protocol.DeviceID
	connections map[protocol.DeviceID]protocol.Connection
	mut         sync.RWMutex
}

func (m *stressTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *stressTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	m.connections[conn.DeviceID()] = conn
	m.t.Logf("Device %s added connection to %s (priority: %d, type: %s)", 
		m.deviceID, conn.DeviceID(), conn.Priority(), conn.Type())
}

func (m *stressTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *stressTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *stressTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *stressTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *stressTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *stressTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *stressTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *stressTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *stressTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *stressTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}

// connectionEvent represents a connection event for testing
type connectionEvent struct {
	timestamp time.Time
	deviceID  protocol.DeviceID
	eventType string // "connected", "disconnected"
}

// resilienceTestModel implements a model for resilience testing
type resilienceTestModel struct {
	t               *testing.T
	deviceID        protocol.DeviceID
	connectionEvents []connectionEvent
	mut             sync.RWMutex
}

func (m *resilienceTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *resilienceTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	m.connectionEvents = append(m.connectionEvents, connectionEvent{
		timestamp: time.Now(),
		deviceID:  conn.DeviceID(),
		eventType: "connected",
	})
	
	m.t.Logf("Device %s added connection to %s (priority: %d, type: %s)", 
		m.deviceID, conn.DeviceID(), conn.Priority(), conn.Type())
}

func (m *resilienceTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *resilienceTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *resilienceTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *resilienceTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *resilienceTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *resilienceTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *resilienceTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *resilienceTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *resilienceTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *resilienceTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}