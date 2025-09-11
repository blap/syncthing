// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package certmanager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCertificateFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test certificate and key files with different naming conventions
	certFile := filepath.Join(tempDir, "test.crt")
	keyFile := filepath.Join(tempDir, "test.key")

	// Write dummy certificate and key content
	certContent := []byte("-----BEGIN CERTIFICATE-----\nMIICljCCAX4CCQDlE8g3lJ2E7TANBgkqhkiG9w0BAQsFADCBjTELMAkGA1UEBhMC\n-----END CERTIFICATE-----")
	keyContent := []byte("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD...\n-----END PRIVATE KEY-----")

	if err := os.WriteFile(certFile, certContent, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(keyFile, keyContent, 0o600); err != nil {
		t.Fatal(err)
	}

	// Create alert service for testing
	as := &AlertService{}

	// Test resolving certificate files with standard naming
	resolvedCert, resolvedKey, err := as.resolveCertificateFiles(certFile)
	if err != nil {
		t.Errorf("Failed to resolve certificate files: %v", err)
	}

	if resolvedCert != certFile {
		t.Errorf("Expected certificate file %s, got %s", certFile, resolvedCert)
	}

	if resolvedKey != keyFile {
		t.Errorf("Expected key file %s, got %s", keyFile, resolvedKey)
	}

	// Test with PEM extension
	certFilePEM := filepath.Join(tempDir, "test.pem")
	if err := os.WriteFile(certFilePEM, certContent, 0o644); err != nil {
		t.Fatal(err)
	}

	resolvedCert, resolvedKey, err = as.resolveCertificateFiles(certFilePEM)
	if err != nil {
		t.Errorf("Failed to resolve certificate files with PEM extension: %v", err)
	}

	if resolvedCert != certFilePEM {
		t.Errorf("Expected certificate file %s, got %s", certFilePEM, resolvedCert)
	}

	if resolvedKey != keyFile {
		t.Errorf("Expected key file %s, got %s", keyFile, resolvedKey)
	}

	// Test with key file having different naming convention
	os.Remove(keyFile)
	altKeyFile := filepath.Join(tempDir, "test-key.pem")
	if err := os.WriteFile(altKeyFile, keyContent, 0o600); err != nil {
		t.Fatal(err)
	}

	resolvedCert, resolvedKey, err = as.resolveCertificateFiles(certFile)
	if err != nil {
		t.Errorf("Failed to resolve certificate files with alternative key naming: %v", err)
	}

	if resolvedCert != certFile {
		t.Errorf("Expected certificate file %s, got %s", certFile, resolvedCert)
	}

	if resolvedKey != altKeyFile {
		t.Errorf("Expected key file %s, got %s", altKeyFile, resolvedKey)
	}
}
