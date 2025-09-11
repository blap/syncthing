// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles automatic certificate renewal and management
package certmanager

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/syncthing/syncthing/internal/slogutil"
	"github.com/syncthing/syncthing/lib/certutil"
)

const (
	// How often to check for certificate expiration
	checkInterval = 6 * time.Hour

	// Renew certificates when they have less than this time left
	renewalThreshold = 30 * 24 * time.Hour // 30 days

	// Certificate lifetime for newly generated certificates
	// certLifetimeDays is now defined in alerts.go to avoid duplication
)

type Service struct {
	suture.Service
	certFile   string
	keyFile    string
	commonName string
	cert       *tls.Certificate
	onRenew    func(tls.Certificate)
}

// New creates a new certificate manager service
func New(certFile, keyFile, commonName string, onRenew func(tls.Certificate)) *Service {
	return &Service{
		certFile:   certFile,
		keyFile:    keyFile,
		commonName: commonName,
		onRenew:    onRenew,
	}
}

// SetCertificate updates the current certificate reference
func (s *Service) SetCertificate(cert tls.Certificate) {
	s.cert = &cert
}

// Serve implements suture.Service
func (s *Service) Serve(ctx context.Context) error {
	slog.Info("Starting certificate manager service",
		"certFile", s.certFile,
		"checkInterval", checkInterval.String())

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Check immediately on startup
	s.checkCertificate()

	for {
		select {
		case <-ticker.C:
			s.checkCertificate()
		case <-ctx.Done():
			slog.Info("Stopping certificate manager service")
			return nil
		}
	}
}

// checkCertificate checks if the certificate needs renewal or regeneration
func (s *Service) checkCertificate() {
	// First check if certificate files exist and are valid
	certExists := s.checkCertificateFiles()

	if !certExists {
		slog.Info("Certificate manager: Certificate files missing, regenerating")
		if err := s.regenerateCertificate(); err != nil {
			slog.Error("Certificate manager: Failed to regenerate certificate", slogutil.Error(err))
			return
		}
		return
	}

	if s.cert == nil {
		slog.Debug("Certificate manager: No certificate to check")
		return
	}

	// Parse the certificate if needed
	leaf := s.cert.Leaf
	if leaf == nil && len(s.cert.Certificate) > 0 {
		var err error
		leaf, err = certutil.ParseCertificate(s.cert.Certificate[0])
		if err != nil {
			slog.Warn("Certificate manager: Failed to parse certificate", slogutil.Error(err))
			// If we can't parse the certificate, try to regenerate it
			if err := s.regenerateCertificate(); err != nil {
				slog.Error("Certificate manager: Failed to regenerate certificate after parse failure", slogutil.Error(err))
			}
			return
		}
	}

	if leaf == nil {
		slog.Warn("Certificate manager: No valid certificate to check")
		// If we don't have a valid certificate, try to regenerate it
		if err := s.regenerateCertificate(); err != nil {
			slog.Error("Certificate manager: Failed to regenerate certificate", slogutil.Error(err))
		}
		return
	}

	slog.Debug("Certificate manager: Checking certificate",
		"notBefore", leaf.NotBefore.Format(time.RFC3339),
		"notAfter", leaf.NotAfter.Format(time.RFC3339),
		"subject", leaf.Subject.String())

	// Check if certificate needs renewal
	timeLeft := time.Until(leaf.NotAfter)
	if timeLeft < renewalThreshold {
		slog.Info("Certificate manager: Certificate needs renewal",
			"expires", leaf.NotAfter.Format(time.RFC3339),
			"timeLeft", timeLeft.String())

		// Generate new certificate
		newCert, err := certutil.NewCertificate(s.certFile, s.keyFile, s.commonName, certLifetimeDays, true)
		if err != nil {
			slog.Error("Certificate manager: Failed to generate new certificate", slogutil.Error(err))
			return
		}

		slog.Info("Certificate manager: Successfully generated new certificate",
			"notAfter", newCert.Leaf.NotAfter.Format(time.RFC3339))

		// Update the certificate reference
		s.cert = &newCert

		// Notify subscribers
		if s.onRenew != nil {
			s.onRenew(newCert)
		}
	} else {
		slog.Debug("Certificate manager: Certificate is still valid",
			"expires", leaf.NotAfter.Format(time.RFC3339),
			"timeLeft", timeLeft.String())
	}
}

// checkCertificateFiles verifies that certificate files exist and are readable
func (s *Service) checkCertificateFiles() bool {
	// Use the same robust resolution logic as in alerts.go
	as := &AlertService{}
	resolvedCertFile, resolvedKeyFile, err := as.resolveCertificateFiles(s.certFile)
	if err != nil {
		slog.Debug("Certificate manager: Failed to resolve certificate files",
			"certFile", s.certFile,
			"error", err)
		return false
	}

	// Try to load the certificate to verify it's valid
	if _, err := tls.LoadX509KeyPair(resolvedCertFile, resolvedKeyFile); err != nil {
		slog.Warn("Certificate manager: Certificate/key pair is invalid",
			"certFile", resolvedCertFile,
			"keyFile", resolvedKeyFile,
			"error", err)
		return false
	}

	// Update our service with the resolved paths
	s.certFile = resolvedCertFile
	s.keyFile = resolvedKeyFile

	return true
}

// regenerateCertificate creates a new certificate/key pair
func (s *Service) regenerateCertificate() error {
	slog.Info("Certificate manager: Regenerating certificate",
		"certFile", s.certFile,
		"keyFile", s.keyFile)

	newCert, err := certutil.NewCertificate(s.certFile, s.keyFile, s.commonName, certLifetimeDays, true)
	if err != nil {
		return fmt.Errorf("failed to generate new certificate: %w", err)
	}

	slog.Info("Certificate manager: Successfully regenerated certificate",
		"notAfter", newCert.Leaf.NotAfter.Format(time.RFC3339))

	// Update the certificate reference
	s.cert = &newCert

	// Notify subscribers
	if s.onRenew != nil {
		s.onRenew(newCert)
	}

	return nil
}
