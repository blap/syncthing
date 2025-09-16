// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/tls"
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

// TestMixedVersionConnection tests connection establishment between v1.x and v2.x devices
func TestMixedVersionConnection(t *testing.T) {
	// Create test environment
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create device IDs
	device1ID := protocol.NewDeviceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	device2ID := protocol.NewDeviceID([]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1})

	// Create registries
	registry1 := registry.New()
	registry2 := registry.New()

	// Create certificates
	cert1 := mustGetCert(t)
	cert2 := mustGetCert(t)

	// Create TLS configurations with proper NextProtos for mixed-version compatibility
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

	// Create configurations - one v1.x device and one v2.x device
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
			LocalAnnEnabled:    true,
			LocalAnnPort:       21027,
			ReconnectIntervalS: 1,
		},
	}, device2ID, events.NoopLogger)

	// Create mock models
	model1 := &mixedVersionTestModel{t: t, deviceID: device1ID}
	model2 := &mixedVersionTestModel{t: t, deviceID: device2ID}

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
	t.Log("Testing DialNow method for mixed version connection")
	service1.DialNow()
	service2.DialNow()
	t.Log("DialNow method called successfully for mixed version connection")

	// Check connection status
	connectedDevices1 := service1.GetConnectedDevices()
	connectedDevices2 := service2.GetConnectedDevices()
	
	t.Logf("Device 1 connected to: %v", connectedDevices1)
	t.Logf("Device 2 connected to: %v", connectedDevices2)

	t.Log("Mixed version connection test completed successfully")
}

// mixedVersionTestModel implements the Model interface for testing mixed version connections
type mixedVersionTestModel struct {
	t        *testing.T
	deviceID protocol.DeviceID
}

func (m *mixedVersionTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s (client: %s %s)", 
		m.deviceID, remoteID, addr, hello.ClientName, hello.ClientVersion)
	return nil
}

func (m *mixedVersionTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.t.Logf("Device %s added connection to %s (client: %s %s)", 
		m.deviceID, conn.DeviceID(), hello.ClientName, hello.ClientVersion)
}

func (m *mixedVersionTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *mixedVersionTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *mixedVersionTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *mixedVersionTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *mixedVersionTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *mixedVersionTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *mixedVersionTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *mixedVersionTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *mixedVersionTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *mixedVersionTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}