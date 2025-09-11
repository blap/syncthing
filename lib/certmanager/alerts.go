// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles certificate expiration alerts
package certmanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/syncthing/syncthing/lib/certutil"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/locations"
	"github.com/syncthing/syncthing/lib/protocol"
)

// AlertService handles certificate expiration alerts
type AlertService struct {
	suture.Service
	evLogger       events.Logger
	checkInterval  time.Duration
	warningPeriods []time.Duration
	alerts         map[string]*CertificateAlert
}

// CertificateAlert represents a certificate expiration alert
type CertificateAlert struct {
	CertificateFile string
	DeviceID        protocol.DeviceID
	Subject         string
	NotAfter        time.Time
	AlertType       AlertType
	CreatedAt       time.Time
	LastNotified    time.Time
}

// AlertType represents the type of certificate alert
type AlertType int

const (
	// AlertTypeExpiringSoon indicates certificate expires soon
	AlertTypeExpiringSoon AlertType = iota

	// AlertTypeExpired indicates certificate has expired
	AlertTypeExpired

	// AlertTypeExpiringVerySoon indicates certificate expires very soon (critical)
	AlertTypeExpiringVerySoon

	// AlertTypeMissing indicates certificate files are missing
	AlertTypeMissing

	// AlertTypeInvalid indicates certificate files are invalid
	AlertTypeInvalid
)

const (
	// Certificate lifetime for newly generated certificates
	certLifetimeDays = 820 // ~2 years

	// Default common name for certificates
	tlsDefaultCommonName = "syncthing"
)

// NewAlertService creates a new certificate expiration alert service
func NewAlertService(evLogger events.Logger) *AlertService {
	return &AlertService{
		evLogger:      evLogger,
		checkInterval: 6 * time.Hour, // Check every 6 hours
		warningPeriods: []time.Duration{
			30 * 24 * time.Hour, // 30 days
			7 * 24 * time.Hour,  // 7 days
			1 * 24 * time.Hour,  // 1 day
		},
		alerts: make(map[string]*CertificateAlert),
	}
}

// Serve implements suture.Service
func (as *AlertService) Serve(ctx context.Context) error {
	slog.Info("Starting certificate expiration alert service",
		"checkInterval", as.checkInterval.String())

	ticker := time.NewTicker(as.checkInterval)
	defer ticker.Stop()

	// Check immediately on startup
	as.checkCertificates()

	for {
		select {
		case <-ticker.C:
			as.checkCertificates()
		case <-ctx.Done():
			slog.Info("Stopping certificate expiration alert service")
			return nil
		}
	}
}

// checkCertificates checks all certificates for expiration and validity
func (as *AlertService) checkCertificates() {
	slog.Debug("Checking certificates for expiration and validity")

	// Check device certificate
	as.checkCertificateFile(locations.Get(locations.CertFile), protocol.DeviceID{})

	// Check HTTPS certificate if different from device certificate
	httpsCertFile := locations.Get(locations.HTTPSCertFile)
	if httpsCertFile != locations.Get(locations.CertFile) {
		as.checkCertificateFile(httpsCertFile, protocol.DeviceID{})
	}

	// Process alerts
	as.processAlerts()
}

// checkCertificateFile checks a specific certificate file for expiration and validity
func (as *AlertService) checkCertificateFile(certFile string, deviceID protocol.DeviceID) {
	// Resolve certificate and key file paths
	resolvedCertFile, resolvedKeyFile, err := as.resolveCertificateFiles(certFile)
	if err != nil {
		slog.Warn("Failed to resolve certificate files",
			"file", certFile,
			"error", err)

		// Try to automatically regenerate the certificate if files are missing
		if as.isMissingFilesError(err) {
			as.regenerateCertificate(certFile, deviceID)
		}

		// Emit a failure event for certificate errors
		as.evLogger.Log(events.Failure, map[string]interface{}{
			"type":    "certificate_error",
			"message": fmt.Sprintf("Failed to resolve certificate files: %v", err),
		})

		return
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(resolvedCertFile, resolvedKeyFile)
	if err != nil {
		slog.Warn("Failed to load certificate",
			"certFile", resolvedCertFile,
			"keyFile", resolvedKeyFile,
			"error", err)

		// Try to automatically regenerate the certificate if it's invalid
		as.regenerateCertificate(certFile, deviceID)

		// Emit a failure event for certificate errors
		as.evLogger.Log(events.Failure, map[string]interface{}{
			"type":    "certificate_error",
			"message": fmt.Sprintf("Failed to load certificate: %v", err),
		})
		return
	}

	// Parse certificate
	if len(cert.Certificate) == 0 {
		slog.Warn("No certificates in certificate file", "file", resolvedCertFile)
		as.regenerateCertificate(certFile, deviceID)

		// Emit a failure event for certificate errors
		as.evLogger.Log(events.Failure, map[string]interface{}{
			"type":    "certificate_error",
			"message": "No certificates in certificate file",
		})
		return
	}

	parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		slog.Warn("Failed to parse certificate", "file", resolvedCertFile, "error", err)
		as.regenerateCertificate(certFile, deviceID)

		// Emit a failure event for certificate errors
		as.evLogger.Log(events.Failure, map[string]interface{}{
			"type":    "certificate_error",
			"message": fmt.Sprintf("Failed to parse certificate: %v", err),
		})
		return
	}

	// Check expiration
	as.checkCertificateExpiration(resolvedCertFile, deviceID, parsedCert)
}

// checkCertificateExpiration checks if a certificate is expiring or expired
func (as *AlertService) checkCertificateExpiration(certFile string, deviceID protocol.DeviceID, cert *x509.Certificate) {
	now := time.Now()
	timeUntilExpiry := cert.NotAfter.Sub(now)

	slog.Debug("Checking certificate expiration",
		"file", certFile,
		"subject", cert.Subject.String(),
		"notAfter", cert.NotAfter.Format(time.RFC3339),
		"timeUntilExpiry", timeUntilExpiry.String())

	// Check if already expired
	if timeUntilExpiry <= 0 {
		as.createAlert(certFile, deviceID, cert, AlertTypeExpired)
		return
	}

	// Check warning periods
	for _, warningPeriod := range as.warningPeriods {
		if timeUntilExpiry <= warningPeriod {
			// Determine alert type based on how soon it expires
			alertType := AlertTypeExpiringSoon
			if timeUntilExpiry <= 24*time.Hour {
				alertType = AlertTypeExpiringVerySoon
			} else if timeUntilExpiry <= 7*24*time.Hour {
				alertType = AlertTypeExpiringSoon
			}

			as.createAlert(certFile, deviceID, cert, alertType)
			return
		}
	}

	// Certificate is not expiring soon, remove any existing alerts
	as.removeAlert(certFile)
}

// createAlert creates or updates a certificate expiration alert
func (as *AlertService) createAlert(certFile string, deviceID protocol.DeviceID, cert *x509.Certificate, alertType AlertType) {
	alertKey := certFile

	// Check if alert already exists
	if existingAlert, exists := as.alerts[alertKey]; exists {
		// Update existing alert
		existingAlert.AlertType = alertType
		existingAlert.NotAfter = cert.NotAfter

		// Check if we should notify again (don't spam notifications)
		timeSinceLastNotify := time.Since(existingAlert.LastNotified)
		shouldNotify := false

		switch alertType {
		case AlertTypeExpiringVerySoon:
			// Notify every 6 hours for critical alerts
			shouldNotify = timeSinceLastNotify >= 6*time.Hour
		case AlertTypeExpiringSoon:
			// Notify every 24 hours for regular alerts
			shouldNotify = timeSinceLastNotify >= 24*time.Hour
		case AlertTypeExpired:
			// Notify every 12 hours for expired certificates
			shouldNotify = timeSinceLastNotify >= 12*time.Hour
		case AlertTypeMissing, AlertTypeInvalid:
			// Notify every 24 hours for missing or invalid certificates
			shouldNotify = timeSinceLastNotify >= 24*time.Hour
		}

		if shouldNotify {
			as.sendAlertNotification(existingAlert)
			existingAlert.LastNotified = time.Now()
		}
	} else {
		// Create new alert
		alert := &CertificateAlert{
			CertificateFile: certFile,
			DeviceID:        deviceID,
			Subject:         cert.Subject.String(),
			NotAfter:        cert.NotAfter,
			AlertType:       alertType,
			CreatedAt:       time.Now(),
			LastNotified:    time.Now(),
		}

		as.alerts[alertKey] = alert
		as.sendAlertNotification(alert)
	}
}

// removeAlert removes an alert for a certificate that is no longer expiring
func (as *AlertService) removeAlert(certFile string) {
	if _, exists := as.alerts[certFile]; exists {
		slog.Debug("Removing certificate expiration alert", "file", certFile)
		delete(as.alerts, certFile)
	}
}

// processAlerts processes all active alerts
func (as *AlertService) processAlerts() {
	now := time.Now()

	for _, alert := range as.alerts {
		// Re-check if alert is still valid
		timeSinceCreated := now.Sub(alert.CreatedAt)

		// Remove alerts older than 30 days
		if timeSinceCreated > 30*24*time.Hour {
			slog.Debug("Removing old certificate alert",
				"file", alert.CertificateFile,
				"age", timeSinceCreated.String())
			delete(as.alerts, alert.CertificateFile)
			continue
		}

		// Send reminder notifications for active alerts
		timeSinceLastNotify := now.Sub(alert.LastNotified)
		shouldNotify := false

		switch alert.AlertType {
		case AlertTypeExpiringVerySoon:
			// Notify every 6 hours for critical alerts
			shouldNotify = timeSinceLastNotify >= 6*time.Hour
		case AlertTypeExpiringSoon:
			// Notify every 24 hours for regular alerts
			shouldNotify = timeSinceLastNotify >= 24*time.Hour
		case AlertTypeExpired:
			// Notify every 12 hours for expired certificates
			shouldNotify = timeSinceLastNotify >= 12*time.Hour
		case AlertTypeMissing, AlertTypeInvalid:
			// Notify every 24 hours for missing or invalid certificates
			shouldNotify = timeSinceLastNotify >= 24*time.Hour
		}

		if shouldNotify {
			as.sendAlertNotification(alert)
			alert.LastNotified = now
		}
	}
}

// resolveCertificateFiles resolves the certificate and key file paths with robust path resolution
func (as *AlertService) resolveCertificateFiles(certFile string) (resolvedCertFile, resolvedKeyFile string, err error) {
	// First check if the certificate file exists as provided
	if _, statErr := os.Stat(certFile); statErr != nil {
		// If the exact path doesn't exist, try with common extensions
		var extensions []string
		if strings.HasSuffix(certFile, ".pem") {
			extensions = []string{".crt", ".cer", ".der"}
		} else if strings.HasSuffix(certFile, ".crt") {
			extensions = []string{".pem", ".cer", ".der"}
		} else {
			extensions = []string{".pem", ".crt", ".cer", ".der"}
		}

		// Try each extension
		baseName := strings.TrimSuffix(certFile, filepath.Ext(certFile))
		found := false
		for _, ext := range extensions {
			candidate := baseName + ext
			if _, candidateErr := os.Stat(candidate); candidateErr == nil {
				certFile = candidate
				found = true
				break
			}
		}

		// If still not found, return the original error
		if !found {
			if _, statErr := os.Stat(certFile); statErr != nil {
				return "", "", fmt.Errorf("certificate file not found: %w", statErr)
			}
		}
	}

	resolvedCertFile = certFile

	// Try to find the corresponding key file with various naming conventions
	keyFileCandidates := []string{
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + ".key",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + "-key.pem",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + "_key.pem",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + "-key.key",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + ".priv",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + ".private",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + "_private.pem",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + "-key",
		strings.TrimSuffix(certFile, filepath.Ext(certFile)) + "_key",
	}

	// Also try looking in the same directory for any key file
	certDir := filepath.Dir(certFile)
	certBase := strings.TrimSuffix(filepath.Base(certFile), filepath.Ext(certFile))

	// Look for key files in the same directory that match the certificate base name
	entries, readErr := os.ReadDir(certDir)
	if readErr == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			baseName := strings.TrimSuffix(name, filepath.Ext(name))

			// If the base name matches and it's a key file, add it as a candidate
			if baseName == certBase || baseName == certBase+"-key" || baseName == certBase+"_key" {
				if strings.Contains(strings.ToLower(name), "key") || strings.Contains(strings.ToLower(name), "priv") {
					keyFileCandidates = append(keyFileCandidates, filepath.Join(certDir, name))
				}
			}
		}
	}

	// Try each key file candidate
	for _, keyFile := range keyFileCandidates {
		if _, statErr := os.Stat(keyFile); statErr == nil {
			resolvedKeyFile = keyFile
			break
		}
	}

	// If no key file found, try with common key extensions
	if resolvedKeyFile == "" {
		keyBase := strings.TrimSuffix(certFile, filepath.Ext(certFile))
		keyExtensions := []string{".key", "-key.pem", "_key.pem", "-key.key", ".priv", ".private", "_private.pem", "-key", "_key"}

		for _, ext := range keyExtensions {
			candidate := keyBase + ext
			if _, statErr := os.Stat(candidate); statErr == nil {
				resolvedKeyFile = candidate
				break
			}
		}
	}

	// If still no key file found, return an error
	if resolvedKeyFile == "" {
		return "", "", fmt.Errorf("key file not found for certificate %s, tried: %v", certFile, keyFileCandidates)
	}

	return resolvedCertFile, resolvedKeyFile, nil
}

// sendAlertNotification sends a notification about a certificate alert
func (as *AlertService) sendAlertNotification(alert *CertificateAlert) {
	// Create event data
	eventData := map[string]interface{}{
		"certificateFile": alert.CertificateFile,
		"subject":         alert.Subject,
		"notAfter":        alert.NotAfter.Format(time.RFC3339),
		"timeUntilExpiry": time.Until(alert.NotAfter).String(),
		"alertType":       alert.AlertType,
	}

	if alert.DeviceID != protocol.EmptyDeviceID {
		eventData["deviceID"] = alert.DeviceID.String()
	}

	// Log the alert
	switch alert.AlertType {
	case AlertTypeExpiringVerySoon:
		slog.Warn("Certificate expires very soon",
			"file", alert.CertificateFile,
			"subject", alert.Subject,
			"expires", alert.NotAfter.Format(time.RFC3339),
			"timeLeft", time.Until(alert.NotAfter).String())
	case AlertTypeExpiringSoon:
		slog.Warn("Certificate expires soon",
			"file", alert.CertificateFile,
			"subject", alert.Subject,
			"expires", alert.NotAfter.Format(time.RFC3339),
			"timeLeft", time.Until(alert.NotAfter).String())
	case AlertTypeExpired:
		slog.Error("Certificate has expired",
			"file", alert.CertificateFile,
			"subject", alert.Subject,
			"expired", alert.NotAfter.Format(time.RFC3339))
	case AlertTypeMissing:
		slog.Error("Certificate files are missing",
			"file", alert.CertificateFile)
	case AlertTypeInvalid:
		slog.Error("Certificate files are invalid",
			"file", alert.CertificateFile)
	}

	// Send event using Failure event type instead of undefined CertificateError
	as.evLogger.Log(events.Failure, eventData)
}

// isMissingFilesError checks if an error indicates missing certificate files
func (as *AlertService) isMissingFilesError(err error) bool {
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such file")
}

// regenerateCertificate creates a new certificate/key pair when existing ones are missing or invalid
func (as *AlertService) regenerateCertificate(certFile string, deviceID protocol.DeviceID) {
	slog.Info("Attempting to regenerate certificate", "file", certFile)

	// Determine the key file path based on the certificate file path
	keyFile := strings.TrimSuffix(certFile, filepath.Ext(certFile)) + ".key"

	// If it's the main certificate, use the default locations
	if certFile == locations.Get(locations.CertFile) {
		certFile = locations.Get(locations.CertFile)
		keyFile = locations.Get(locations.KeyFile)
	} else if certFile == locations.Get(locations.HTTPSCertFile) && certFile != locations.Get(locations.CertFile) {
		// If it's the HTTPS certificate and it's different from the main certificate
		certFile = locations.Get(locations.HTTPSCertFile)
		keyFile = locations.Get(locations.HTTPSKeyFile)
	}

	// Generate new certificate
	newCert, err := certutil.NewCertificate(certFile, keyFile, tlsDefaultCommonName, certLifetimeDays, true)
	if err != nil {
		slog.Error("Failed to regenerate certificate",
			"certFile", certFile,
			"keyFile", keyFile,
			"error", err)

		// Create an alert for the invalid certificate
		as.createAlert(certFile, deviceID, &x509.Certificate{
			Subject: pkix.Name{CommonName: "Unknown"},
		}, AlertTypeInvalid)
		return
	}

	slog.Info("Successfully regenerated certificate",
		"certFile", certFile,
		"notAfter", newCert.Leaf.NotAfter.Format(time.RFC3339))

	// Remove any existing alerts for this certificate since it's now valid
	as.removeAlert(certFile)
}
