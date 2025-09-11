// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles certificate revocation functionality
package certmanager

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"
)

// RevocationService handles certificate revocation checking
type RevocationService struct {
	suture.Service
	revokedCerts map[string]bool // certificate serial numbers that are revoked
	mutex        sync.RWMutex
}

// NewRevocationService creates a new revocation service
func NewRevocationService() *RevocationService {
	return &RevocationService{
		revokedCerts: make(map[string]bool),
	}
}

// Serve implements suture.Service
func (rs *RevocationService) Serve(ctx context.Context) error {
	slog.Info("Starting certificate revocation service")

	// In a full implementation, this would periodically check CRLs or OCSP
	// For now, we just provide the framework

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping certificate revocation service")
			return nil
		case <-time.After(1 * time.Hour):
			// Periodic check would go here
			rs.mutex.RLock()
			count := len(rs.revokedCerts)
			rs.mutex.RUnlock()
			slog.Debug("Certificate revocation service running", "revokedCerts", count)
		}
	}
}

// RevokeCertificate adds a certificate to the revoked list
func (rs *RevocationService) RevokeCertificate(cert *x509.Certificate) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	serial := cert.SerialNumber.String()
	rs.revokedCerts[serial] = true

	slog.Info("Certificate revoked",
		"serial", serial,
		"subject", cert.Subject.String())

	return nil
}

// IsCertificateRevoked checks if a certificate has been revoked
func (rs *RevocationService) IsCertificateRevoked(cert *x509.Certificate) (bool, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	serial := cert.SerialNumber.String()
	if revoked, exists := rs.revokedCerts[serial]; exists {
		return revoked, nil
	}

	// In a full implementation, we would check CRLs or OCSP here
	// For now, we assume it's not revoked if not in our list

	return false, nil
}

// RemoveRevokedCertificate removes a certificate from the revoked list
func (rs *RevocationService) RemoveRevokedCertificate(cert *x509.Certificate) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	serial := cert.SerialNumber.String()
	if _, exists := rs.revokedCerts[serial]; !exists {
		return fmt.Errorf("certificate with serial %s is not in revocation list", serial)
	}

	delete(rs.revokedCerts, serial)

	slog.Info("Certificate removed from revocation list",
		"serial", serial,
		"subject", cert.Subject.String())

	return nil
}

// GetRevokedCertificates returns a list of revoked certificate serial numbers
func (rs *RevocationService) GetRevokedCertificates() []string {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	serials := make([]string, 0, len(rs.revokedCerts))
	for serial := range rs.revokedCerts {
		serials = append(serials, serial)
	}

	return serials
}
