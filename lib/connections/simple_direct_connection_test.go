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
	"testing"
	"time"

	"github.com/syncthing/syncthing/internal/db"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/stats"
)

// TestSimpleDirectConnection tests a simple direct connection between two Syncthing instances
func TestSimpleDirectConnection(t *testing.T) {
	// Set up test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two test devices
	device1ID := protocol.NewDeviceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	device2ID := protocol.NewDeviceID([]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1})

	// Create registry for connection tracking
	registry1 := registry.New()
	registry2 := registry.New()

	// Set up configuration for device 1
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
			LocalAnnEnabled:  false,
			ReconnectIntervalS: 1,
		},
	}, device1ID, events.NoopLogger)

	// Set up configuration for device 2
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
			LocalAnnEnabled:  false,
			ReconnectIntervalS: 1,
		},
	}, device2ID, events.NoopLogger)

	// Create TLS configurations - use the same approach as existing tests
	tlsCfg1 := &tls.Config{
		Certificates: []tls.Certificate{generateDirectTestCertificate(t, device1ID)},
		NextProtos:   []string{"bep/1.0"},
		ServerName:   "syncthing",
	}
	
	tlsCfg2 := &tls.Config{
		Certificates: []tls.Certificate{generateDirectTestCertificate(t, device2ID)},
		NextProtos:   []string{"bep/1.0"},
		ServerName:   "syncthing",
	}

	// Create mock models
	model1 := &directTestModel{t: t, deviceID: device1ID}
	model2 := &directTestModel{t: t, deviceID: device2ID}

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

	// Trigger immediate dialing
	service1.DialNow()
	service2.DialNow()

	// Wait for connection establishment
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			connectedDevices1 := service1.GetConnectedDevices()
			connectedDevices2 := service2.GetConnectedDevices()
			t.Logf("Timeout: Device 1 connected to: %v", connectedDevices1)
			t.Logf("Timeout: Device 2 connected to: %v", connectedDevices2)
			
			// Log connection status
			connStatus1 := service1.ConnectionStatus()
			connStatus2 := service2.ConnectionStatus()
			t.Logf("Final connection status for device 1: %v", connStatus1)
			t.Logf("Final connection status for device 2: %v", connStatus2)
			
			t.Fatal("Timeout waiting for connection establishment")
		case <-ticker.C:
			// Check if devices are connected
			connectedDevices1 := service1.GetConnectedDevices()
			connectedDevices2 := service2.GetConnectedDevices()
			
			t.Logf("Device 1 connected to: %v", connectedDevices1)
			t.Logf("Device 2 connected to: %v", connectedDevices2)
			
			// Check if both devices are connected to each other
			if containsDevice(connectedDevices1, device2ID) && containsDevice(connectedDevices2, device1ID) {
				t.Log("Direct connection successfully established")
				return
			}
		}
	}
}

// generateDirectTestCertificate generates a test certificate for a device
func generateDirectTestCertificate(t *testing.T, _ protocol.DeviceID) tls.Certificate {
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

// directTestModel implements the Model interface for testing
type directTestModel struct {
	t        *testing.T
	deviceID protocol.DeviceID
}

func (m *directTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *directTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.t.Logf("Device %s added connection to %s", m.deviceID, conn.DeviceID())
}

func (m *directTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *directTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *directTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *directTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *directTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *directTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *directTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *directTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *directTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *directTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}