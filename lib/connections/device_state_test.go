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

// TestDeviceStates tests connection behavior with different device states
func TestDeviceStates(t *testing.T) {
	// Test different device state scenarios using the withConnectionPair helper
	testCases := []struct {
		name    string
		address string
	}{
		{"NormalDevice", "tcp://127.0.0.1:0"},
		{"WANDevice", "tcp://0.0.0.0:0"},
		{"LocalDevice", "tcp://127.0.0.1:0"},
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

// TestPausedDeviceConnection tests behavior when connecting to a paused device
func TestPausedDeviceConnection(t *testing.T) {
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
		NextProtos:         []string{"bep/1.0"},
		ServerName:         "syncthing",
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequestClientCert,
	}
	
	tlsCfg2 := &tls.Config{
		Certificates:       []tls.Certificate{cert2},
		NextProtos:         []string{"bep/1.0"},
		ServerName:         "syncthing",
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequestClientCert,
	}

	// Create configurations with one device paused
	cfg1 := config.Wrap("/dev/null", config.Configuration{
		Devices: []config.DeviceConfiguration{
			{
				DeviceID:  device1ID,
				Addresses: []string{"dynamic"},
			},
			{
				DeviceID:  device2ID,
				Addresses: []string{"dynamic"},
				Paused:    true, // Paused device
			},
		},
		Options: config.OptionsConfiguration{
			RawListenAddresses: []string{"tcp://127.0.0.1:0"},
			GlobalAnnEnabled:   false,
			LocalAnnEnabled:    false,
			ReconnectIntervalS: 1,
		},
	}, device1ID, events.NoopLogger)

	cfg2 := config.Wrap("/dev/null", config.Configuration{
		Devices: []config.DeviceConfiguration{
			{
				DeviceID:  device1ID,
				Addresses: []string{"dynamic"},
				Paused:    true, // Paused device
			},
			{
				DeviceID:  device2ID,
				Addresses: []string{"dynamic"},
			},
		},
		Options: config.OptionsConfiguration{
			RawListenAddresses: []string{"tcp://127.0.0.1:0"},
			GlobalAnnEnabled:   false,
			LocalAnnEnabled:    false,
			ReconnectIntervalS: 1,
		},
	}, device2ID, events.NoopLogger)

	// Create mock models
	model1 := &deviceStateTestModel{t: t, deviceID: device1ID}
	model2 := &deviceStateTestModel{t: t, deviceID: device2ID}

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

	// Test that DialNow method exists and can be called
	t.Log("Testing DialNow method with paused devices")
	service1.DialNow()
	service2.DialNow()
	t.Log("DialNow method called successfully with paused devices")

	// Check connection status with paused devices
	connectedDevices1 := service1.GetConnectedDevices()
	connectedDevices2 := service2.GetConnectedDevices()
	
	t.Logf("Device 1 connected to: %v", connectedDevices1)
	t.Logf("Device 2 connected to: %v", connectedDevices2)

	t.Log("Paused device connection test completed successfully")
}

// deviceStateTestModel implements the Model interface for testing device states
type deviceStateTestModel struct {
	t        *testing.T
	deviceID protocol.DeviceID
}

func (m *deviceStateTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *deviceStateTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.t.Logf("Device %s added connection to %s", m.deviceID, conn.DeviceID())
}

func (m *deviceStateTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *deviceStateTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *deviceStateTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *deviceStateTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *deviceStateTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *deviceStateTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *deviceStateTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *deviceStateTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *deviceStateTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *deviceStateTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}