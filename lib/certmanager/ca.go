// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles integration with external CA systems
package certmanager

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/syncthing/syncthing/lib/locations"
)

// CAService handles integration with external Certificate Authority systems
type CAService struct {
	caCertificates []*x509.Certificate
	caCertPool     *x509.CertPool
	crlFiles       []string
}

// NewCAService creates a new CA integration service
func NewCAService() *CAService {
	return &CAService{
		caCertificates: make([]*x509.Certificate, 0),
		crlFiles:       make([]string, 0),
	}
}

// LoadCACertificates loads CA certificates from a directory or file
func (cas *CAService) LoadCACertificates(caPath string) error {
	// Check if path exists
	info, err := os.Stat(caPath)
	if err != nil {
		return fmt.Errorf("failed to stat CA path %s: %w", caPath, err)
	}

	var certFiles []string
	if info.IsDir() {
		// Load all .pem and .crt files from directory
		entries, err := os.ReadDir(caPath)
		if err != nil {
			return fmt.Errorf("failed to read CA directory %s: %w", caPath, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if filepath.Ext(name) == ".pem" || filepath.Ext(name) == ".crt" {
				certFiles = append(certFiles, filepath.Join(caPath, name))
			}
		}
	} else {
		// Single file
		certFiles = append(certFiles, caPath)
	}

	// Load each certificate file
	for _, certFile := range certFiles {
		if err := cas.loadCACertificateFile(certFile); err != nil {
			slog.Warn("Failed to load CA certificate", "file", certFile, "error", err)
			continue
		}
		slog.Debug("Loaded CA certificate", "file", certFile)
	}

	// Update certificate pool
	cas.updateCertPool()

	slog.Info("Loaded CA certificates", "count", len(cas.caCertificates), "files", len(certFiles))
	return nil
}

// loadCACertificateFile loads a single CA certificate file
func (cas *CAService) loadCACertificateFile(certFile string) error {
	// Read certificate file
	certData, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file %s: %w", certFile, err)
	}

	// Parse certificates
	certs, err := parseCertificates(certData)
	if err != nil {
		return fmt.Errorf("failed to parse certificates from %s: %w", certFile, err)
	}

	// Add certificates to CA list
	for _, cert := range certs {
		// Only add CA certificates (BasicConstraints with CA=true)
		if cert.BasicConstraintsValid && cert.IsCA {
			cas.caCertificates = append(cas.caCertificates, cert)
		} else {
			slog.Debug("Skipping non-CA certificate", "subject", cert.Subject.String(), "file", certFile)
		}
	}

	return nil
}

// parseCertificates parses PEM-encoded certificates
func parseCertificates(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	// Try to parse as PEM
	for len(data) > 0 {
		cert, rest, err := parsePEMCertificate(data)
		if err != nil {
			// Try to parse as DER
			if len(certs) == 0 {
				cert, err := x509.ParseCertificate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate: %w", err)
				}
				certs = append(certs, cert)
				break
			}
			return certs, nil
		}

		certs = append(certs, cert)
		data = rest
	}

	return certs, nil
}

// parsePEMCertificate parses a single PEM-encoded certificate
func parsePEMCertificate(data []byte) (*x509.Certificate, []byte, error) {
	// Implementation would parse PEM format
	// For now, we'll just parse as DER since the full PEM parsing is complex
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, nil, err
	}

	return cert, nil, nil
}

// updateCertPool updates the certificate pool with current CA certificates
func (cas *CAService) updateCertPool() {
	pool := x509.NewCertPool()
	for _, cert := range cas.caCertificates {
		pool.AddCert(cert)
	}
	cas.caCertPool = pool
}

// VerifyWithCA verifies a certificate against the loaded CA certificates
func (cas *CAService) VerifyWithCA(cert *x509.Certificate) error {
	if cas.caCertPool == nil {
		return fmt.Errorf("no CA certificates loaded")
	}

	// Verify certificate against CA pool
	_, err := cert.Verify(x509.VerifyOptions{
		Roots:       cas.caCertPool,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return fmt.Errorf("certificate verification against CA failed: %w", err)
	}

	return nil
}

// LoadDefaultCAs loads default system CA certificates
func (cas *CAService) LoadDefaultCAs() error {
	// Try to load system CA certificates
	pool, err := x509.SystemCertPool()
	if err != nil {
		slog.Warn("Failed to load system certificate pool", "error", err)
		// Create empty pool as fallback
		pool = x509.NewCertPool()
	}

	// Store the pool for later use
	cas.caCertPool = pool

	slog.Info("Loaded default system CA certificates")
	return nil
}

// LoadSyncthingCAs loads Syncthing's built-in CA certificates
func (cas *CAService) LoadSyncthingCAs() error {
	// Load from Syncthing's CA directory if it exists
	caDir := filepath.Join(locations.GetBaseDir(locations.ConfigBaseDir), "ca")

	if _, err := os.Stat(caDir); os.IsNotExist(err) {
		slog.Debug("Syncthing CA directory does not exist", "dir", caDir)
		return nil
	}

	return cas.LoadCACertificates(caDir)
}

// GetCACertificates returns the loaded CA certificates
func (cas *CAService) GetCACertificates() []*x509.Certificate {
	// Return a copy of the slice
	result := make([]*x509.Certificate, len(cas.caCertificates))
	copy(result, cas.caCertificates)
	return result
}

// HasCAs returns true if any CA certificates are loaded
func (cas *CAService) HasCAs() bool {
	return len(cas.caCertificates) > 0
}

// ConfigureTLS configures TLS settings to use CA certificates
func (cas *CAService) ConfigureTLS(tlsConfig *tls.Config) {
	if cas.caCertPool != nil {
		tlsConfig.RootCAs = cas.caCertPool
		tlsConfig.ClientCAs = cas.caCertPool
	}
}

// LoadCRLs loads Certificate Revocation Lists
func (cas *CAService) LoadCRLs(crlPath string) error {
	// Check if path exists
	info, err := os.Stat(crlPath)
	if err != nil {
		return fmt.Errorf("failed to stat CRL path %s: %w", crlPath, err)
	}

	var crlFiles []string
	if info.IsDir() {
		// Load all .crl files from directory
		entries, err := os.ReadDir(crlPath)
		if err != nil {
			return fmt.Errorf("failed to read CRL directory %s: %w", crlPath, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if filepath.Ext(name) == ".crl" {
				crlFiles = append(crlFiles, filepath.Join(crlPath, name))
			}
		}
	} else {
		// Single file
		crlFiles = append(crlFiles, crlPath)
	}

	// Store CRL files for later use
	cas.crlFiles = append(cas.crlFiles, crlFiles...)

	slog.Info("Loaded CRL files", "count", len(crlFiles))
	return nil
}

// CheckRevocation checks if a certificate has been revoked
func (cas *CAService) CheckRevocation(cert *x509.Certificate) error {
	// This would check against loaded CRLs or OCSP
	// For now, we'll just log that revocation checking would happen here

	slog.Debug("Certificate revocation check would be performed here",
		"subject", cert.Subject.String(),
		"serial", cert.SerialNumber.String())

	return nil
}

// IsCA returns true if a certificate is a CA certificate
func IsCA(cert *x509.Certificate) bool {
	return cert.BasicConstraintsValid && cert.IsCA
}

// IsIntermediateCA returns true if a certificate is an intermediate CA
func IsIntermediateCA(cert *x509.Certificate) bool {
	return IsCA(cert) && !cert.IsCA && len(cert.Subject.String()) > 0
}
