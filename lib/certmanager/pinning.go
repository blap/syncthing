// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles certificate pinning functionality
package certmanager

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"

	"github.com/syncthing/syncthing/lib/protocol"
)

// PinningService handles certificate pinning for device authentication
type PinningService struct {
	// Map of device ID to pinned certificate hashes
	pinnedCerts map[protocol.DeviceID][]string
	mutex       sync.RWMutex
}

// NewPinningService creates a new certificate pinning service
func NewPinningService() *PinningService {
	return &PinningService{
		pinnedCerts: make(map[protocol.DeviceID][]string),
	}
}

// PinCertificate pins a certificate for a specific device
func (ps *PinningService) PinCertificate(deviceID protocol.DeviceID, cert *x509.Certificate) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// Calculate certificate hash
	hash := ps.calculateCertHash(cert)
	
	// Add to pinned certificates for this device
	if _, exists := ps.pinnedCerts[deviceID]; !exists {
		ps.pinnedCerts[deviceID] = make([]string, 0)
	}
	
	// Check if already pinned
	for _, pinnedHash := range ps.pinnedCerts[deviceID] {
		if pinnedHash == hash {
			slog.Debug("Certificate already pinned for device", 
				"device", deviceID.String(), 
				"hash", hash)
			return nil
		}
	}
	
	// Add new pin
	ps.pinnedCerts[deviceID] = append(ps.pinnedCerts[deviceID], hash)
	
	slog.Info("Certificate pinned for device", 
		"device", deviceID.String(), 
		"hash", hash,
		"subject", cert.Subject.String())
	
	return nil
}

// UnpinCertificate removes a certificate pin for a specific device
func (ps *PinningService) UnpinCertificate(deviceID protocol.DeviceID, cert *x509.Certificate) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// Calculate certificate hash
	hash := ps.calculateCertHash(cert)
	
	// Check if device has pinned certificates
	pinned, exists := ps.pinnedCerts[deviceID]
	if !exists {
		return fmt.Errorf("no pinned certificates for device %s", deviceID.String())
	}
	
	// Find and remove the pin
	found := false
	newPins := make([]string, 0, len(pinned))
	for _, pinnedHash := range pinned {
		if pinnedHash == hash {
			found = true
			slog.Info("Certificate unpinned for device", 
				"device", deviceID.String(), 
				"hash", hash)
		} else {
			newPins = append(newPins, pinnedHash)
		}
	}
	
	if !found {
		return fmt.Errorf("certificate with hash %s not pinned for device %s", hash, deviceID.String())
	}
	
	// Update pinned certificates
	if len(newPins) == 0 {
		delete(ps.pinnedCerts, deviceID)
	} else {
		ps.pinnedCerts[deviceID] = newPins
	}
	
	return nil
}

// IsCertificatePinned checks if a certificate is pinned for a specific device
func (ps *PinningService) IsCertificatePinned(deviceID protocol.DeviceID, cert *x509.Certificate) (bool, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	// Calculate certificate hash
	hash := ps.calculateCertHash(cert)
	
	// Check if device has pinned certificates
	pinned, exists := ps.pinnedCerts[deviceID]
	if !exists {
		// No pinned certificates for this device
		return false, nil
	}
	
	// Check if certificate is pinned
	for _, pinnedHash := range pinned {
		if pinnedHash == hash {
			return true, nil
		}
	}
	
	return false, nil
}

// VerifyPinnedCertificate verifies that a connection uses a pinned certificate
func (ps *PinningService) VerifyPinnedCertificate(deviceID protocol.DeviceID, connState tls.ConnectionState) error {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	// Check if device has pinned certificates
	pinned, exists := ps.pinnedCerts[deviceID]
	if !exists {
		// No pinned certificates for this device, verification passes
		slog.Debug("No pinned certificates for device, verification passes", 
			"device", deviceID.String())
		return nil
	}
	
	// Must have at least one peer certificate
	if len(connState.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates provided for pinned device %s", deviceID.String())
	}
	
	// Get the leaf certificate (the one actually used in the connection)
	leafCert := connState.PeerCertificates[0]
	leafHash := ps.calculateCertHash(leafCert)
	
	// Check if the leaf certificate is pinned
	for _, pinnedHash := range pinned {
		if pinnedHash == leafHash {
			slog.Debug("Pinned certificate verified for device", 
				"device", deviceID.String(), 
				"hash", leafHash)
			return nil
		}
	}
	
	// Certificate not pinned
	return fmt.Errorf("peer certificate for device %s is not pinned (hash: %s)", 
		deviceID.String(), leafHash)
}

// GetPinnedCertificates returns all pinned certificates for a device
func (ps *PinningService) GetPinnedCertificates(deviceID protocol.DeviceID) ([]string, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	pinned, exists := ps.pinnedCerts[deviceID]
	if !exists {
		return []string{}, nil
	}
	
	// Return a copy of the slice
	result := make([]string, len(pinned))
	copy(result, pinned)
	
	return result, nil
}

// ClearPinnedCertificates removes all pinned certificates for a device
func (ps *PinningService) ClearPinnedCertificates(deviceID protocol.DeviceID) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	if _, exists := ps.pinnedCerts[deviceID]; !exists {
		return fmt.Errorf("no pinned certificates for device %s", deviceID.String())
	}
	
	delete(ps.pinnedCerts, deviceID)
	
	slog.Info("All certificates unpinned for device", 
		"device", deviceID.String())
	
	return nil
}

// calculateCertHash calculates the SHA256 hash of a certificate
func (ps *PinningService) calculateCertHash(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:])
}

// HasPinnedCertificates checks if a device has any pinned certificates
func (ps *PinningService) HasPinnedCertificates(deviceID protocol.DeviceID) bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	_, exists := ps.pinnedCerts[deviceID]
	return exists
}