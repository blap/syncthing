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

// TestImprovedDirectLANConnection tests direct LAN connection establishment between two Syncthing instances
func TestImprovedDirectLANConnection(t *testing.T) {
	// Set up test environment
	t.Log("Setting up test environment with 30 second timeout")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	t.Log("Test environment set up")

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
			LocalAnnEnabled:  false, // Disable local discovery for direct connection testing
			ReconnectIntervalS: 1, // Faster reconnection for testing
			ConnectionPriorityTCPLAN: 10,
			ConnectionPriorityTCPWAN: 30,
			ConnectionPriorityUpgradeThreshold: 0,
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
			LocalAnnEnabled:  false, // Disable local discovery for direct connection testing
			ReconnectIntervalS: 1, // Faster reconnection for testing
			ConnectionPriorityTCPLAN: 10,
			ConnectionPriorityTCPWAN: 30,
			ConnectionPriorityUpgradeThreshold: 0,
		},
	}, device2ID, events.NoopLogger)

	// Create TLS configurations
	tlsCfg1 := &tls.Config{
		Certificates: []tls.Certificate{generateImprovedLANTestCertificate(t, device1ID)},
		NextProtos:   []string{"bep/1.0"},
		ServerName:   "syncthing",
	}
	
	tlsCfg2 := &tls.Config{
		Certificates: []tls.Certificate{generateImprovedLANTestCertificate(t, device2ID)},
		NextProtos:   []string{"bep/1.0"},
		ServerName:   "syncthing",
	}

	// Create mock models
	model1 := &improvedTestModel{t: t, deviceID: device1ID}
	model2 := &improvedTestModel{t: t, deviceID: device2ID}

	// Create connection services
	t.Log("Creating service1")
	service1 := NewService(cfg1, device1ID, model1, tlsCfg1, nil, "bep/1.0", "syncthing", events.NoopLogger, registry1, nil)
	t.Log("Service1 created")

	t.Log("Creating service2")
	service2 := NewService(cfg2, device2ID, model2, tlsCfg2, nil, "bep/1.0", "syncthing", events.NoopLogger, registry2, nil)
	t.Log("Service2 created")

	// Start services
	t.Log("Starting service1")
	go service1.Serve(ctx)
	t.Log("Service1 started goroutine")

	t.Log("Starting service2")
	go service2.Serve(ctx)
	t.Log("Service2 started goroutine")

	// Wait for services to start
	t.Log("Waiting 200ms for services to start")
	time.Sleep(200 * time.Millisecond)
	t.Log("Finished waiting for services to start")

	// Get listener addresses
	listenerStatus1 := service1.ListenerStatus()
	listenerStatus2 := service2.ListenerStatus()
	
	if len(listenerStatus1) == 0 || len(listenerStatus2) == 0 {
		t.Fatal("Failed to start listeners")
	}

	// Log all listener status for debugging
	t.Logf("Device 1 listener status: %v", listenerStatus1)
	t.Logf("Device 2 listener status: %v", listenerStatus2)

	// Extract actual listening addresses - use the first LAN address that's not 0.0.0.0
	var addr1, addr2 string
	for _, status := range listenerStatus1 {
		t.Logf("Device 1 status - LAN: %v, WAN: %v", status.LANAddresses, status.WANAddresses)
		// Find the first LAN address that's not 0.0.0.0
		for _, lanAddr := range status.LANAddresses {
			if lanAddr != "tcp://0.0.0.0:0" && lanAddr != "tcp://127.0.0.1:0" && lanAddr != "" {
				addr1 = lanAddr
				t.Logf("Selected addr1: %s", addr1)
				break
			}
		}
		if addr1 != "" {
			break
		}
	}
	
	for _, status := range listenerStatus2 {
		t.Logf("Device 2 status - LAN: %v, WAN: %v", status.LANAddresses, status.WANAddresses)
		// Find the first LAN address that's not 0.0.0.0
		for _, lanAddr := range status.LANAddresses {
			if lanAddr != "tcp://0.0.0.0:0" && lanAddr != "tcp://127.0.0.1:0" && lanAddr != "" {
				addr2 = lanAddr
				t.Logf("Selected addr2: %s", addr2)
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

	// Update configurations with actual addresses - make sure to parse and reformat correctly
	addr1 = formatAddress(addr1)
	addr2 = formatAddress(addr2)
	
	t.Logf("Formatted addr1: %s", addr1)
	t.Logf("Formatted addr2: %s", addr2)

	// Update configurations with actual addresses
	cfg1.Modify(func(cfg *config.Configuration) {
		for i := range cfg.Devices {
			if cfg.Devices[i].DeviceID == device2ID {
				cfg.Devices[i].Addresses = []string{addr2}
				t.Logf("Set device2 address in device1 config: %s", addr2)
				break
			}
		}
	})
	
	cfg2.Modify(func(cfg *config.Configuration) {
		for i := range cfg.Devices {
			if cfg.Devices[i].DeviceID == device1ID {
				cfg.Devices[i].Addresses = []string{addr1}
				t.Logf("Set device1 address in device2 config: %s", addr1)
				break
			}
		}
	})

	// Add a small delay to ensure configurations are updated
	t.Log("About to sleep for 200ms to ensure configurations are updated")
	time.Sleep(200 * time.Millisecond)
	t.Log("Finished sleeping")

	// Wait for connection establishment
	t.Log("Setting up timeout and ticker for connection establishment")
	timeout := time.After(20 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	t.Log("Timeout and ticker set up")

	// Log initial connection status
	t.Log("Getting initial connection status")
	connStatus1 := service1.ConnectionStatus()
	connStatus2 := service2.ConnectionStatus()
	t.Logf("Initial connection status for device 1: %v", connStatus1)
	t.Logf("Initial connection status for device 2: %v", connStatus2)

	// Trigger immediate dialing
	t.Log("About to call DialNow on service1")
	service1.DialNow()
	t.Log("Called DialNow on service1")

	t.Log("About to call DialNow on service2")
	service2.DialNow()
	t.Log("Called DialNow on service2")

	t.Log("Starting connection establishment loop")
	for {
		select {
		case <-timeout:
			// Debug information
			t.Log("Timeout reached in connection establishment loop")
			connectedDevices1 := service1.GetConnectedDevices()
			connectedDevices2 := service2.GetConnectedDevices()
			t.Logf("Timeout: Device 1 connected to: %v", connectedDevices1)
			t.Logf("Timeout: Device 2 connected to: %v", connectedDevices2)
			
			// Log connection status
			connStatus1 := service1.ConnectionStatus()
			connStatus2 := service2.ConnectionStatus()
			t.Logf("Final connection status for device 1: %v", connStatus1)
			t.Logf("Final connection status for device 2: %v", connStatus2)
			
			// Log device statistics
			stats1, _ := model1.DeviceStatistics()
			stats2, _ := model2.DeviceStatistics()
			t.Logf("Device 1 statistics: %v", stats1)
			t.Logf("Device 2 statistics: %v", stats2)
			
			t.Fatal("Timeout waiting for connection establishment")
		case <-ticker.C:
			// Check if devices are connected
			connectedDevices1 := service1.GetConnectedDevices()
			connectedDevices2 := service2.GetConnectedDevices()
			
			t.Logf("Device 1 connected to: %v", connectedDevices1)
			t.Logf("Device 2 connected to: %v", connectedDevices2)
			
			// Check if both devices are connected to each other
			if containsDevice(connectedDevices1, device2ID) && containsDevice(connectedDevices2, device1ID) {
				t.Log("Direct LAN connection successfully established")
				return
			}
			
			// Log connection status periodically
			if time.Now().Unix()%5 == 0 { // Log every 5 seconds
				connStatus1 := service1.ConnectionStatus()
				connStatus2 := service2.ConnectionStatus()
				t.Logf("Current connection status for device 1: %v", connStatus1)
				t.Logf("Current connection status for device 2: %v", connStatus2)
			}
		}
	}
}

// formatAddress ensures the address is in the correct format
func formatAddress(addr string) string {
	// If it's already in the correct format, return as is
	if len(addr) > 6 && addr[:6] == "tcp://" {
		return addr
	}
	
	// Otherwise, add the tcp:// prefix
	return "tcp://" + addr
}

// generateImprovedLANTestCertificate generates a test certificate for a device
func generateImprovedLANTestCertificate(t *testing.T, _ protocol.DeviceID) tls.Certificate {
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

// improvedTestModel implements the Model interface for testing
type improvedTestModel struct {
	t        *testing.T
	deviceID protocol.DeviceID
}

func (m *improvedTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device %s received hello from %s at %s", m.deviceID, remoteID, addr)
	return nil
}

func (m *improvedTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.t.Logf("Device %s added connection to %s (type: %s, priority: %d)", m.deviceID, conn.DeviceID(), conn.Type(), conn.Priority())
}

func (m *improvedTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *improvedTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *improvedTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device %s received index from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *improvedTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device %s received index update from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *improvedTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device %s received request from %s", m.deviceID, conn.DeviceID())
	return nil, nil
}

func (m *improvedTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device %s received cluster config from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *improvedTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device %s connection to %s closed: %v", m.deviceID, conn.DeviceID(), err)
}

func (m *improvedTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device %s received download progress from %s", m.deviceID, conn.DeviceID())
	return nil
}

func (m *improvedTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *improvedTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}