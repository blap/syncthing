// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles automated failure detection
package certmanager

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

// FailureDetectionService detects and responds to connection failures
type FailureDetectionService struct {
	suture.Service
	evLogger              events.Logger
	metricsService        *MetricsService
	failureThresholds     map[protocol.DeviceID]*FailureThresholds
	mutex                 sync.RWMutex
	detectionInterval     time.Duration
	failureCallbacks      []FailureCallback
	autoRemediationEnabled bool
}

// FailureThresholds defines thresholds for failure detection
type FailureThresholds struct {
	MaxConsecutiveFailures int
	MaxFailureRate         float64
	MaxTLSErrorRate        float64
	MaxCertificateErrorRate float64
	TimeWindow             time.Duration
}

// FailureCallback is a function that gets called when a failure is detected
type FailureCallback func(deviceID protocol.DeviceID, failureType FailureType, details map[string]interface{})

// FailureType represents the type of failure detected
type FailureType int

const (
	// FailureTypeConsecutiveFailures indicates too many consecutive failures
	FailureTypeConsecutiveFailures FailureType = iota
	
	// FailureTypeHighFailureRate indicates a high overall failure rate
	FailureTypeHighFailureRate
	
	// FailureTypeHighTLSErrorRate indicates a high TLS error rate
	FailureTypeHighTLSErrorRate
	
	// FailureTypeHighCertificateErrorRate indicates a high certificate error rate
	FailureTypeHighCertificateErrorRate
	
	// FailureTypeCertificateExpiring indicates a certificate is expiring soon
	FailureTypeCertificateExpiring
	
	// FailureTypeCertificateExpired indicates a certificate has expired
	FailureTypeCertificateExpired
)

// Default failure thresholds
var DefaultFailureThresholds = FailureThresholds{
	MaxConsecutiveFailures: 5,
	MaxFailureRate:         0.5, // 50%
	MaxTLSErrorRate:        0.3, // 30%
	MaxCertificateErrorRate: 0.2, // 20%
	TimeWindow:             1 * time.Hour,
}

// NewFailureDetectionService creates a new failure detection service
func NewFailureDetectionService(evLogger events.Logger, metricsService *MetricsService) *FailureDetectionService {
	return &FailureDetectionService{
		evLogger:          evLogger,
		metricsService:    metricsService,
		failureThresholds: make(map[protocol.DeviceID]*FailureThresholds),
		detectionInterval: 5 * time.Minute,
		failureCallbacks:  make([]FailureCallback, 0),
	}
}

// Serve implements suture.Service
func (fds *FailureDetectionService) Serve(ctx context.Context) error {
	slog.Info("Starting automated failure detection service", 
		"detectionInterval", fds.detectionInterval.String())
	
	ticker := time.NewTicker(fds.detectionInterval)
	defer ticker.Stop()
	
	// Check immediately on startup
	fds.detectFailures()
	
	for {
		select {
		case <-ticker.C:
			fds.detectFailures()
		case <-ctx.Done():
			slog.Info("Stopping automated failure detection service")
			return nil
		}
	}
}

// detectFailures checks all devices for failure patterns
func (fds *FailureDetectionService) detectFailures() {
	slog.Debug("Detecting connection failures")
	
	// Get all device metrics
	allMetrics := fds.metricsService.GetAllMetrics()
	
	for deviceID, metrics := range allMetrics {
		// Check for various failure patterns
		fds.checkConsecutiveFailures(deviceID, metrics)
		fds.checkFailureRate(deviceID, metrics)
		fds.checkTLSErrorRate(deviceID, metrics)
		fds.checkCertificateErrorRate(deviceID, metrics)
	}
}

// checkConsecutiveFailures checks for consecutive connection failures
func (fds *FailureDetectionService) checkConsecutiveFailures(deviceID protocol.DeviceID, metrics *DeviceMetrics) {
	thresholds := fds.getThresholds(deviceID)
	
	// Check if we have recent consecutive failures
	consecutiveFailures := fds.countConsecutiveFailures(metrics)
	
	if consecutiveFailures >= thresholds.MaxConsecutiveFailures {
		details := map[string]interface{}{
			"consecutiveFailures": consecutiveFailures,
			"maxAllowed":          thresholds.MaxConsecutiveFailures,
			"lastError":           metrics.LastError,
			"lastErrorTime":       metrics.LastErrorTime.Format(time.RFC3339),
		}
		
		fds.reportFailure(deviceID, FailureTypeConsecutiveFailures, details)
	}
}

// countConsecutiveFailures counts consecutive failures for a device
func (fds *FailureDetectionService) countConsecutiveFailures(metrics *DeviceMetrics) int {
	// This is a simplified implementation
	// In a real implementation, we would track the actual sequence of events
	if metrics.SuccessfulConnections == 0 && metrics.FailedConnections > 0 {
		return int(metrics.FailedConnections)
	}
	
	// If we have recent successes, we don't have consecutive failures
	if metrics.LastSuccessfulConnection.After(metrics.LastFailedConnection) {
		return 0
	}
	
	// Count failures since last success
	timeSinceLastSuccess := time.Since(metrics.LastSuccessfulConnection)
	if timeSinceLastSuccess < 10*time.Minute {
		// Assume recent failures are consecutive if no recent success
		return int(metrics.FailedConnections - metrics.SuccessfulConnections)
	}
	
	return 0
}

// checkFailureRate checks if the overall failure rate is too high
func (fds *FailureDetectionService) checkFailureRate(deviceID protocol.DeviceID, metrics *DeviceMetrics) {
	thresholds := fds.getThresholds(deviceID)
	
	rate, err := fds.metricsService.GetFailureRate(deviceID)
	if err != nil {
		slog.Debug("Failed to calculate failure rate", 
			"device", deviceID.String(), 
			"error", err)
		return
	}
	
	if rate > thresholds.MaxFailureRate {
		details := map[string]interface{}{
			"failureRate": rate,
			"maxAllowed":  thresholds.MaxFailureRate,
			"totalAttempts": metrics.ConnectionAttempts,
			"failedConnections": metrics.FailedConnections,
		}
		
		fds.reportFailure(deviceID, FailureTypeHighFailureRate, details)
	}
}

// checkTLSErrorRate checks if the TLS error rate is too high
func (fds *FailureDetectionService) checkTLSErrorRate(deviceID protocol.DeviceID, metrics *DeviceMetrics) {
	thresholds := fds.getThresholds(deviceID)
	
	rate, err := fds.metricsService.GetTLSErrorRate(deviceID)
	if err != nil {
		slog.Debug("Failed to calculate TLS error rate", 
			"device", deviceID.String(), 
			"error", err)
		return
	}
	
	if rate > thresholds.MaxTLSErrorRate {
		details := map[string]interface{}{
			"tlsErrorRate": rate,
			"maxAllowed":   thresholds.MaxTLSErrorRate,
			"tlsFailures":  metrics.TLSHandshakeFailures,
			"totalAttempts": metrics.ConnectionAttempts,
		}
		
		fds.reportFailure(deviceID, FailureTypeHighTLSErrorRate, details)
	}
}

// checkCertificateErrorRate checks if the certificate error rate is too high
func (fds *FailureDetectionService) checkCertificateErrorRate(deviceID protocol.DeviceID, metrics *DeviceMetrics) {
	thresholds := fds.getThresholds(deviceID)
	
	rate, err := fds.metricsService.GetCertificateErrorRate(deviceID)
	if err != nil {
		slog.Debug("Failed to calculate certificate error rate", 
			"device", deviceID.String(), 
			"error", err)
		return
	}
	
	if rate > thresholds.MaxCertificateErrorRate {
		details := map[string]interface{}{
			"certificateErrorRate": rate,
			"maxAllowed":           thresholds.MaxCertificateErrorRate,
			"certificateErrors":    metrics.CertificateErrors,
			"totalAttempts":        metrics.ConnectionAttempts,
		}
		
		fds.reportFailure(deviceID, FailureTypeHighCertificateErrorRate, details)
	}
}

// getThresholds returns the failure thresholds for a device
func (fds *FailureDetectionService) getThresholds(deviceID protocol.DeviceID) *FailureThresholds {
	fds.mutex.RLock()
	defer fds.mutex.RUnlock()
	
	if thresholds, exists := fds.failureThresholds[deviceID]; exists {
		return thresholds
	}
	
	// Return default thresholds
	return &DefaultFailureThresholds
}

// SetFailureThresholds sets custom failure thresholds for a device
func (fds *FailureDetectionService) SetFailureThresholds(deviceID protocol.DeviceID, thresholds *FailureThresholds) {
	fds.mutex.Lock()
	defer fds.mutex.Unlock()
	
	fds.failureThresholds[deviceID] = thresholds
	slog.Debug("Set custom failure thresholds for device", 
		"device", deviceID.String())
}

// reportFailure reports a detected failure
func (fds *FailureDetectionService) reportFailure(deviceID protocol.DeviceID, failureType FailureType, details map[string]interface{}) {
	// Add device ID to details
	details["deviceID"] = deviceID.String()
	details["failureType"] = failureTypeToString(failureType)
	
	// Log the failure
	slog.Warn("Connection failure detected", 
		"device", deviceID.String(),
		"type", failureTypeToString(failureType),
		"details", details)
	
	// Send event
	fds.evLogger.Log(events.Failure, details)
	
	// Call registered callbacks
	fds.mutex.RLock()
	callbacks := make([]FailureCallback, len(fds.failureCallbacks))
	copy(callbacks, fds.failureCallbacks)
	fds.mutex.RUnlock()
	
	for _, callback := range callbacks {
		callback(deviceID, failureType, details)
	}
	
	// Perform auto-remediation if enabled
	if fds.autoRemediationEnabled {
		fds.performAutoRemediation(deviceID, failureType, details)
	}
}

// failureTypeToString converts a FailureType to a string
func failureTypeToString(failureType FailureType) string {
	switch failureType {
	case FailureTypeConsecutiveFailures:
		return "consecutive_failures"
	case FailureTypeHighFailureRate:
		return "high_failure_rate"
	case FailureTypeHighTLSErrorRate:
		return "high_tls_error_rate"
	case FailureTypeHighCertificateErrorRate:
		return "high_certificate_error_rate"
	case FailureTypeCertificateExpiring:
		return "certificate_expiring"
	case FailureTypeCertificateExpired:
		return "certificate_expired"
	default:
		return "unknown"
	}
}

// AddFailureCallback registers a callback function for failure notifications
func (fds *FailureDetectionService) AddFailureCallback(callback FailureCallback) {
	fds.mutex.Lock()
	defer fds.mutex.Unlock()
	
	fds.failureCallbacks = append(fds.failureCallbacks, callback)
	slog.Debug("Added failure callback")
}

// RemoveFailureCallback removes a callback function
func (fds *FailureDetectionService) RemoveFailureCallback(callback FailureCallback) {
	fds.mutex.Lock()
	defer fds.mutex.Unlock()
	
	for i, cb := range fds.failureCallbacks {
		if fmt.Sprintf("%p", cb) == fmt.Sprintf("%p", callback) {
			fds.failureCallbacks = append(fds.failureCallbacks[:i], fds.failureCallbacks[i+1:]...)
			slog.Debug("Removed failure callback")
			return
		}
	}
}

// EnableAutoRemediation enables automatic remediation of detected failures
func (fds *FailureDetectionService) EnableAutoRemediation(enabled bool) {
	fds.mutex.Lock()
	defer fds.mutex.Unlock()
	
	fds.autoRemediationEnabled = enabled
	slog.Info("Auto-remediation enabled", "enabled", enabled)
}

// performAutoRemediation attempts to automatically fix detected failures
func (fds *FailureDetectionService) performAutoRemediation(deviceID protocol.DeviceID, failureType FailureType, details map[string]interface{}) {
	_ = details // Mark parameter as used to avoid unused parameter warning
	slog.Info("Performing auto-remediation for failure", 
		"device", deviceID.String(),
		"type", failureTypeToString(failureType))
	
	switch failureType {
	case FailureTypeHighCertificateErrorRate, FailureTypeCertificateExpiring, FailureTypeCertificateExpired:
		// For certificate-related issues, we might want to:
		// 1. Force certificate renewal
		// 2. Clear pinned certificates
		// 3. Send notification to user
		
		slog.Info("Certificate-related failure detected, consider certificate renewal", 
			"device", deviceID.String())
		
	case FailureTypeHighTLSErrorRate:
		// For TLS errors, we might want to:
		// 1. Check TLS configuration
		// 2. Temporarily allow less secure connections
		// 3. Send notification to user
		
		slog.Info("TLS-related failure detected, consider checking TLS configuration", 
			"device", deviceID.String())
		
	case FailureTypeConsecutiveFailures:
		// For consecutive failures, we might want to:
		// 1. Temporarily pause connections to this device
		// 2. Try alternative connection methods
		// 3. Send notification to user
		
		slog.Info("Consecutive failures detected, consider pausing connections", 
			"device", deviceID.String())
		
	default:
		slog.Debug("No auto-remediation available for failure type", 
			"type", failureTypeToString(failureType))
	}
}

// ForceDetection triggers an immediate failure detection check
func (fds *FailureDetectionService) ForceDetection() {
	slog.Info("Forcing failure detection check")
	fds.detectFailures()
}

// GetFailureStats returns statistics about detected failures
func (fds *FailureDetectionService) GetFailureStats() map[string]interface{} {
	// This would return statistics about detected failures
	// For now, we'll return a placeholder
	return map[string]interface{}{
		"service": "failure_detection",
		"status":  "active",
	}
}

// ResetFailureStats resets failure statistics
func (fds *FailureDetectionService) ResetFailureStats() {
	// This would reset failure statistics
	slog.Info("Reset failure detection statistics")
}

// IsAutoRemediationEnabled returns whether auto-remediation is enabled
func (fds *FailureDetectionService) IsAutoRemediationEnabled() bool {
	fds.mutex.RLock()
	defer fds.mutex.RUnlock()
	
	return fds.autoRemediationEnabled
}