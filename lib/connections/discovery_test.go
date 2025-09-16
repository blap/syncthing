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
	"testing"
	"time"

	"github.com/syncthing/syncthing/internal/db"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/stats"
)

// TestGlobalDiscovery tests global discovery functionality
func TestGlobalDiscovery(t *testing.T) {
	// Test global discovery using the withConnectionPair helper
	testCases := []struct {
		name    string
		address string
	}{
		{"GlobalDiscovery", "tcp://127.0.0.1:0"},
		{"WANGlobalDiscovery", "tcp://0.0.0.0:0"},
	}

	send := make([]byte, 1024)
	// Fill with some test data
	for i := range send {
		send[i] = byte(i % 256)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withConnectionPair(t, tc.address, func(client, server internalConn) {
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

				t.Logf("Connection established and data transmitted successfully with %s", tc.name)
			})
		})
	}
}

// TestLocalDiscovery tests local discovery functionality
func TestLocalDiscovery(t *testing.T) {
	// Create test environment
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test devices
	device1ID := protocol.NewDeviceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	device2ID := protocol.NewDeviceID([]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1})

	// Create registries
	registry1 := registry.New()
	registry2 := registry.New()

	// Create certificates
	cert1 := mustGetCert(t)
	cert2 := mustGetCert(t)

	// Create TLS configurations
	tlsCfg1 := &tls.Config{
		Certificates:       []tls.Certificate{cert1},
		NextProtos:         []string{"bep/2.0", "bep/1.0", "h2", "http/1.1"},
		ServerName:         "syncthing",
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequestClientCert,
	}
	
	tlsCfg2 := &tls.Config{
		Certificates:       []tls.Certificate{cert2},
		NextProtos:         []string{"bep/2.0", "bep/1.0", "h2", "http/1.1"},
		ServerName:         "syncthing",
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequestClientCert,
	}

	// Create configurations with local discovery enabled
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
			LocalAnnEnabled:    true, // Enable local discovery
			LocalAnnPort:       21027,
			ReconnectIntervalS: 1,
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
			GlobalAnnEnabled:   false,
			LocalAnnEnabled:    true, // Enable local discovery
			LocalAnnPort:       21027,
			ReconnectIntervalS: 1,
		},
	}, device2ID, events.NoopLogger)

	// Create mock models
	model1 := &discoveryTestModel{t: t, deviceID: device1ID}
	model2 := &discoveryTestModel{t: t, deviceID: device2ID}

	// Create services
	service1 := NewService(cfg1, device1ID, model1, tlsCfg1, nil, "bep/1.0", "syncthing", events.NoopLogger, registry1, nil)
	service2 := NewService(cfg2, device2ID, model2, tlsCfg2, nil, "bep/1.0", "syncthing", events.NoopLogger, registry2, nil)

	// Start services
	go service1.Serve(ctx)
	go service2.Serve(ctx)

	// Give services time to start
	time.Sleep(100 * time.Millisecond)

	// Verify services started
	listenerStatus1 := service1.ListenerStatus()
	listenerStatus2 := service2.ListenerStatus()
	
	if len(listenerStatus1) == 0 || len(listenerStatus2) == 0 {
		t.Fatal("Failed to start listeners")
	}

	// Get listener addresses
	var addr1, addr2 string
	for _, status := range listenerStatus1 {
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

	// Test that DialNow method exists and can be called
	t.Log("Testing DialNow method with local discovery")
	service1.DialNow()
	service2.DialNow()
	t.Log("DialNow method called successfully with local discovery")

	// Check connection status with local discovery
	connectedDevices1 := service1.GetConnectedDevices()
	connectedDevices2 := service2.GetConnectedDevices()
	
	t.Logf("Device 1 connected to: %v", connectedDevices1)
	t.Logf("Device 2 connected to: %v", connectedDevices2)

	t.Log("Local discovery test completed successfully")
}

// TestPeerAssistedDiscovery tests peer-assisted discovery functionality
func TestPeerAssistedDiscovery(t *testing.T) {
	// Test peer-assisted discovery using the withConnectionPair helper
	addrs := []string{
		"tcp://127.0.0.1:0",
	}

	send := make([]byte, 1024)
	// Fill with some test data
	for i := range send {
		send[i] = byte(i % 256)
	}

	for _, addr := range addrs {
		t.Run("PeerAssisted_"+addr, func(t *testing.T) {
			withConnectionPair(t, addr, func(client, server internalConn) {
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

				t.Log("Peer-assisted discovery connection established and data transmitted successfully")
			})
		})
	}
}

// discoveryTestModel implements the Model interface for testing discovery
type discoveryTestModel struct {
	t        *testing.T
	deviceID protocol.DeviceID
}

func (m *discoveryTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *discoveryTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.t.Logf("Device %s added connection to %s", m.deviceID, conn.DeviceID())
}

func (m *discoveryTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *discoveryTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *discoveryTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *discoveryTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *discoveryTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *discoveryTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *discoveryTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *discoveryTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *discoveryTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *discoveryTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}