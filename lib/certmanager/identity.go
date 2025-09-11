// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles enhanced device identity verification
package certmanager

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// IdentityVerificationService handles enhanced device identity verification
type IdentityVerificationService struct {
	pinningService *PinningService
}

// NewIdentityVerificationService creates a new identity verification service
func NewIdentityVerificationService(pinningService *PinningService) *IdentityVerificationService {
	return &IdentityVerificationService{
		pinningService: pinningService,
	}
}

// VerifyDeviceIdentity performs comprehensive device identity verification
func (ivs *IdentityVerificationService) VerifyDeviceIdentity(deviceID protocol.DeviceID, connState tls.ConnectionState) error {
	// Must have at least one peer certificate
	if len(connState.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates provided for device %s", deviceID.String())
	}

	// Get the leaf certificate (the one actually used in the connection)
	leafCert := connState.PeerCertificates[0]

	// Verify the device ID matches the certificate
	expectedDeviceID := protocol.NewDeviceID(leafCert.Raw)
	if !deviceID.Equals(expectedDeviceID) {
		return fmt.Errorf("device ID mismatch - expected: %s, certificate: %s",
			deviceID.String(), expectedDeviceID.String())
	}

	// Check certificate validity period
	now := time.Now()
	if now.Before(leafCert.NotBefore) {
		return fmt.Errorf("peer certificate is not yet valid - notBefore: %s",
			leafCert.NotBefore.Format(time.RFC3339))
	}

	if now.After(leafCert.NotAfter) {
		return fmt.Errorf("peer certificate has expired - notAfter: %s",
			leafCert.NotAfter.Format(time.RFC3339))
	}

	// Verify certificate signature
	_, err := leafCert.Verify(x509.VerifyOptions{
		Roots:       nil, // Self-signed certificate
		CurrentTime: now,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return fmt.Errorf("peer certificate signature verification failed: %w", err)
	}

	// Check certificate pinning if available
	if ivs.pinningService != nil {
		isPinned, err := ivs.pinningService.IsCertificatePinned(deviceID, leafCert)
		if err != nil {
			slog.Warn("Failed to check certificate pinning",
				"device", deviceID.String(),
				"error", err)
		} else if !isPinned && ivs.pinningService.HasPinnedCertificates(deviceID) {
			// Device has pinned certificates but this one is not pinned
			return fmt.Errorf("peer certificate is not pinned for device %s", deviceID.String())
		}
	}

	// Log successful verification
	slog.Debug("Device identity verification successful",
		"device", deviceID.String(),
		"subject", leafCert.Subject.String(),
		"issuer", leafCert.Issuer.String(),
		"notBefore", leafCert.NotBefore.Format(time.RFC3339),
		"notAfter", leafCert.NotAfter.Format(time.RFC3339))

	return nil
}

// GetDeviceCertificateInfo extracts certificate information for a device
func (ivs *IdentityVerificationService) GetDeviceCertificateInfo(deviceID protocol.DeviceID, connState tls.ConnectionState) (*DeviceCertificateInfo, error) {
	// Must have at least one peer certificate
	if len(connState.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates provided for device %s", deviceID.String())
	}

	// Get the leaf certificate
	leafCert := connState.PeerCertificates[0]

	// Calculate if certificate is pinned
	isPinned := false
	if ivs.pinningService != nil {
		var err error
		isPinned, err = ivs.pinningService.IsCertificatePinned(deviceID, leafCert)
		if err != nil {
			slog.Warn("Failed to check certificate pinning",
				"device", deviceID.String(),
				"error", err)
		}
	}

	info := &DeviceCertificateInfo{
		DeviceID:        deviceID,
		Subject:         leafCert.Subject.String(),
		Issuer:          leafCert.Issuer.String(),
		NotBefore:       leafCert.NotBefore,
		NotAfter:        leafCert.NotAfter,
		SerialNumber:    leafCert.SerialNumber.String(),
		IsPinned:        isPinned,
		IsValid:         time.Now().After(leafCert.NotBefore) && time.Now().Before(leafCert.NotAfter),
		TimeUntilExpiry: time.Until(leafCert.NotAfter),
	}

	return info, nil
}

// DeviceCertificateInfo contains information about a device's certificate
type DeviceCertificateInfo struct {
	DeviceID        protocol.DeviceID
	Subject         string
	Issuer          string
	NotBefore       time.Time
	NotAfter        time.Time
	SerialNumber    string
	IsPinned        bool
	IsValid         bool
	TimeUntilExpiry time.Duration
}

// ValidateCertificateChain validates the entire certificate chain
func (ivs *IdentityVerificationService) ValidateCertificateChain(deviceID protocol.DeviceID, connState tls.ConnectionState) error {
	// Validate each certificate in the chain
	for i, cert := range connState.PeerCertificates {
		now := time.Now()

		// Check validity period
		if now.Before(cert.NotBefore) {
			return fmt.Errorf("certificate %d in chain is not yet valid - notBefore: %s",
				i, cert.NotBefore.Format(time.RFC3339))
		}

		if now.After(cert.NotAfter) {
			return fmt.Errorf("certificate %d in chain has expired - notAfter: %s",
				i, cert.NotAfter.Format(time.RFC3339))
		}

		// Verify certificate signature (for non-leaf certificates, verify against parent)
		if i == 0 {
			// Leaf certificate - self-signed in Syncthing
			_, err := cert.Verify(x509.VerifyOptions{
				Roots:       nil, // Self-signed certificate
				CurrentTime: now,
				KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			})
			if err != nil {
				return fmt.Errorf("leaf certificate signature verification failed: %w", err)
			}
		} else {
			// Intermediate certificate - verify against parent
			_, err := cert.Verify(x509.VerifyOptions{
				Roots: x509.NewCertPool(),
				Intermediates: func() *x509.CertPool {
					pool := x509.NewCertPool()
					for j := 0; j < i-1; j++ {
						pool.AddCert(connState.PeerCertificates[j])
					}
					return pool
				}(),
				CurrentTime: now,
				KeyUsages:   cert.ExtKeyUsage,
			})
			if err != nil {
				return fmt.Errorf("intermediate certificate %d signature verification failed: %w", i, err)
			}
		}

		slog.Debug("Certificate in chain validated",
			"device", deviceID.String(),
			"index", i,
			"subject", cert.Subject.String())
	}

	return nil
}

// IsDeviceCertificateExpiringSoon checks if a device's certificate expires soon
func (ivs *IdentityVerificationService) IsDeviceCertificateExpiringSoon(deviceID protocol.DeviceID, connState tls.ConnectionState, threshold time.Duration) (bool, error) {
	// Must have at least one peer certificate
	if len(connState.PeerCertificates) == 0 {
		return false, fmt.Errorf("no peer certificates provided for device %s", deviceID.String())
	}

	// Get the leaf certificate
	leafCert := connState.PeerCertificates[0]

	// Check if certificate expires soon
	timeUntilExpiry := time.Until(leafCert.NotAfter)

	slog.Debug("Checking certificate expiry",
		"device", deviceID.String(),
		"notAfter", leafCert.NotAfter.Format(time.RFC3339),
		"timeUntilExpiry", timeUntilExpiry.String(),
		"threshold", threshold.String())

	return timeUntilExpiry < threshold, nil
}
