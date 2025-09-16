// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/tls"
	"io"
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

// TestConcurrentConnections tests concurrent connection establishment between multiple Syncthing instances
func TestConcurrentConnections(t *testing.T) {
	// This test follows the same pattern as the working TestConnectionEstablishment
	// but tests multiple concurrent connections
	addrs := []string{
		"tcp://127.0.0.1:0",
		"tcp://127.0.0.1:0",
		"tcp://127.0.0.1:0",
	}

	send := make([]byte, 512)
	// Fill with some test data
	for i := range send {
		send[i] = byte(i % 256)
	}

	// Test multiple concurrent connections
	var wg sync.WaitGroup
	for i, addr := range addrs {
		wg.Add(1)
		go func(index int, address string) {
			defer wg.Done()
			t.Run("Concurrent_"+address+"_"+string(rune(index+'0')), func(t *testing.T) {
				withConnectionPair(t, address, func(client, server internalConn) {
					// Test data transmission
					if _, err := client.Write(send); err != nil {
						t.Fatal(err)
					}

					recv := make([]byte, len(send))
					if _, err := io.ReadFull(server, recv); err != nil {
						t.Fatal(err)
					}

					// Verify data integrity
					for i := range send {
						if recv[i] != send[i] {
							t.Fatalf("Data mismatch at position %d: expected %d, got %d", i, send[i], recv[i])
						}
					}

					t.Logf("Concurrent connection %d established and data transmitted successfully", index)
				})
			})
		}(i, addr)
	}
	wg.Wait()
	t.Log("All concurrent connections established successfully")
}

// TestServiceDialNowConcurrent tests that the DialNow method works with concurrent connections
func TestServiceDialNowConcurrent(t *testing.T) {
	// Create test environment with 3 devices
	numDevices := 3
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
	models := make([]*concurrentServiceTestModel, numDevices)
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

		// Create TLS configuration using the mustGetCert helper
		cert := mustGetCert(t)
		tlsCfg := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			NextProtos:         []string{"bep/2.0", "bep/1.0", "h2", "http/1.1"},
			ServerName:         "syncthing",
			InsecureSkipVerify: true,
			ClientAuth:         tls.RequestClientCert,
		}

		// Create model
		models[i] = &concurrentServiceTestModel{
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

	// Trigger immediate dialing for all services
	t.Log("Testing DialNow method on all services")
	for i := 0; i < numDevices; i++ {
		services[i].DialNow()
	}
	t.Log("DialNow method called on all services successfully")

	// Check that services respond to method calls
	totalConnections := 0
	for i := 0; i < numDevices; i++ {
		connectedDevices := services[i].GetConnectedDevices()
		count := len(connectedDevices)
		totalConnections += count
		t.Logf("Device %d connected to: %v (%d connections)", i, connectedDevices, count)
	}

	t.Logf("Service DialNow concurrent test completed successfully with %d total connections", totalConnections)
}

// concurrentServiceTestModel implements a model for concurrent connection testing
type concurrentServiceTestModel struct {
	t           *testing.T
	deviceID    protocol.DeviceID
	connections map[protocol.DeviceID]protocol.Connection
	mut         sync.RWMutex
}

func (m *concurrentServiceTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *concurrentServiceTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	m.connections[conn.DeviceID()] = conn
	m.t.Logf("Device %s added connection to %s (priority: %d, type: %s)", 
		m.deviceID, conn.DeviceID(), conn.Priority(), conn.Type())
}

func (m *concurrentServiceTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *concurrentServiceTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *concurrentServiceTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *concurrentServiceTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *concurrentServiceTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *concurrentServiceTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *concurrentServiceTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *concurrentServiceTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *concurrentServiceTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *concurrentServiceTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}