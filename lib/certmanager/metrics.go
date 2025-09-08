// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles connection quality metrics
package certmanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/syncthing/syncthing/lib/protocol"
)

// MetricsService collects connection quality metrics
type MetricsService struct {
	suture.Service
	metrics map[protocol.DeviceID]*DeviceMetrics
	mutex   sync.RWMutex
}

// DeviceMetrics contains metrics for a specific device
type DeviceMetrics struct {
	DeviceID              protocol.DeviceID
	ConnectionAttempts    int64
	SuccessfulConnections int64
	FailedConnections     int64
	TLSHandshakeFailures  int64
	CertificateErrors     int64
	LastConnectionAttempt time.Time
	LastSuccessfulConnection time.Time
	LastFailedConnection  time.Time
	ConnectionDurations   []time.Duration
	LastError             string
	LastErrorTime         time.Time
}

// ConnectionEvent represents a connection event for metrics collection
type ConnectionEvent struct {
	DeviceID     protocol.DeviceID
	EventType    ConnectionEventType
	Error        error
	Duration     time.Duration
	CertInfo     *CertificateInfo
}

// ConnectionEventType represents the type of connection event
type ConnectionEventType int

const (
	// ConnectionAttempt indicates a connection attempt was made
	ConnectionAttempt ConnectionEventType = iota
	
	// ConnectionSuccess indicates a connection was successful
	ConnectionSuccess
	
	// ConnectionFailure indicates a connection failed
	ConnectionFailure
	
	// TLSHandshakeFailure indicates a TLS handshake failed
	TLSHandshakeFailure
	
	// CertificateError indicates a certificate-related error
	CertificateError
)

// CertificateInfo contains information about a certificate for metrics
type CertificateInfo struct {
	Subject      string
	Issuer       string
	NotBefore    time.Time
	NotAfter     time.Time
	SerialNumber string
	IsSelfSigned bool
	IsValid      bool
}

// NewMetricsService creates a new connection quality metrics service
func NewMetricsService() *MetricsService {
	return &MetricsService{
		metrics: make(map[protocol.DeviceID]*DeviceMetrics),
	}
}

// Serve implements suture.Service
func (ms *MetricsService) Serve(ctx context.Context) error {
	slog.Info("Starting connection quality metrics service")
	
	// This service mainly provides methods for other components to report metrics
	// It doesn't have its own background tasks
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping connection quality metrics service")
			return nil
		case <-time.After(1 * time.Hour):
			// Periodic cleanup of old metrics could go here
			ms.cleanupOldMetrics()
		}
	}
}

// RecordConnectionEvent records a connection event for metrics
func (ms *MetricsService) RecordConnectionEvent(event *ConnectionEvent) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	// Get or create metrics for this device
	deviceMetrics, exists := ms.metrics[event.DeviceID]
	if !exists {
		deviceMetrics = &DeviceMetrics{
			DeviceID:           event.DeviceID,
			ConnectionDurations: make([]time.Duration, 0, 100), // Keep last 100 durations
		}
		ms.metrics[event.DeviceID] = deviceMetrics
	}
	
	// Update metrics based on event type
	deviceMetrics.LastConnectionAttempt = time.Now()
	
	switch event.EventType {
	case ConnectionAttempt:
		deviceMetrics.ConnectionAttempts++
		
	case ConnectionSuccess:
		deviceMetrics.SuccessfulConnections++
		deviceMetrics.LastSuccessfulConnection = time.Now()
		// Store connection duration (up to 100 recent values)
		if len(deviceMetrics.ConnectionDurations) >= 100 {
			// Remove oldest duration
			deviceMetrics.ConnectionDurations = deviceMetrics.ConnectionDurations[1:]
		}
		deviceMetrics.ConnectionDurations = append(deviceMetrics.ConnectionDurations, event.Duration)
		
	case ConnectionFailure:
		deviceMetrics.FailedConnections++
		deviceMetrics.LastFailedConnection = time.Now()
		if event.Error != nil {
			deviceMetrics.LastError = event.Error.Error()
			deviceMetrics.LastErrorTime = time.Now()
		}
		
	case TLSHandshakeFailure:
		deviceMetrics.TLSHandshakeFailures++
		deviceMetrics.FailedConnections++
		deviceMetrics.LastFailedConnection = time.Now()
		if event.Error != nil {
			deviceMetrics.LastError = event.Error.Error()
			deviceMetrics.LastErrorTime = time.Now()
		}
		
	case CertificateError:
		deviceMetrics.CertificateErrors++
		deviceMetrics.FailedConnections++
		deviceMetrics.LastFailedConnection = time.Now()
		if event.Error != nil {
			deviceMetrics.LastError = event.Error.Error()
			deviceMetrics.LastErrorTime = time.Now()
		}
	}
	
	slog.Debug("Recorded connection event", 
		"device", event.DeviceID.String(),
		"type", event.EventType,
		"error", event.Error)
}

// GetDeviceMetrics returns metrics for a specific device
func (ms *MetricsService) GetDeviceMetrics(deviceID protocol.DeviceID) (*DeviceMetrics, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	metrics, exists := ms.metrics[deviceID]
	if !exists {
		return nil, fmt.Errorf("no metrics found for device %s", deviceID.String())
	}
	
	// Return a copy of the metrics
	metricsCopy := *metrics
	return &metricsCopy, nil
}

// GetAllMetrics returns metrics for all devices
func (ms *MetricsService) GetAllMetrics() map[protocol.DeviceID]*DeviceMetrics {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	// Return a copy of all metrics
	result := make(map[protocol.DeviceID]*DeviceMetrics, len(ms.metrics))
	for deviceID, metrics := range ms.metrics {
		metricsCopy := *metrics
		// Copy the durations slice
		durationsCopy := make([]time.Duration, len(metrics.ConnectionDurations))
		copy(durationsCopy, metrics.ConnectionDurations)
		metricsCopy.ConnectionDurations = durationsCopy
		
		result[deviceID] = &metricsCopy
	}
	
	return result
}

// GetConnectionSuccessRate calculates the connection success rate for a device
func (ms *MetricsService) GetConnectionSuccessRate(deviceID protocol.DeviceID) (float64, error) {
	metrics, err := ms.GetDeviceMetrics(deviceID)
	if err != nil {
		return 0, err
	}
	
	totalAttempts := metrics.ConnectionAttempts
	if totalAttempts == 0 {
		return 1.0, nil // No attempts, assume 100% success
	}
	
	successRate := float64(metrics.SuccessfulConnections) / float64(totalAttempts)
	return successRate, nil
}

// GetAverageConnectionDuration calculates the average connection duration for a device
func (ms *MetricsService) GetAverageConnectionDuration(deviceID protocol.DeviceID) (time.Duration, error) {
	metrics, err := ms.GetDeviceMetrics(deviceID)
	if err != nil {
		return 0, err
	}
	
	if len(metrics.ConnectionDurations) == 0 {
		return 0, nil
	}
	
	var total time.Duration
	for _, duration := range metrics.ConnectionDurations {
		total += duration
	}
	
	return total / time.Duration(len(metrics.ConnectionDurations)), nil
}

// GetFailureRate calculates the failure rate for a device
func (ms *MetricsService) GetFailureRate(deviceID protocol.DeviceID) (float64, error) {
	metrics, err := ms.GetDeviceMetrics(deviceID)
	if err != nil {
		return 0, err
	}
	
	totalAttempts := metrics.ConnectionAttempts
	if totalAttempts == 0 {
		return 0, nil
	}
	
	failureRate := float64(metrics.FailedConnections) / float64(totalAttempts)
	return failureRate, nil
}

// GetTLSErrorRate calculates the TLS error rate for a device
func (ms *MetricsService) GetTLSErrorRate(deviceID protocol.DeviceID) (float64, error) {
	metrics, err := ms.GetDeviceMetrics(deviceID)
	if err != nil {
		return 0, err
	}
	
	totalAttempts := metrics.ConnectionAttempts
	if totalAttempts == 0 {
		return 0, nil
	}
	
	tlsErrorRate := float64(metrics.TLSHandshakeFailures) / float64(totalAttempts)
	return tlsErrorRate, nil
}

// GetCertificateErrorRate calculates the certificate error rate for a device
func (ms *MetricsService) GetCertificateErrorRate(deviceID protocol.DeviceID) (float64, error) {
	metrics, err := ms.GetDeviceMetrics(deviceID)
	if err != nil {
		return 0, err
	}
	
	totalAttempts := metrics.ConnectionAttempts
	if totalAttempts == 0 {
		return 0, nil
	}
	
	certErrorRate := float64(metrics.CertificateErrors) / float64(totalAttempts)
	return certErrorRate, nil
}

// ResetMetrics resets metrics for a specific device
func (ms *MetricsService) ResetMetrics(deviceID protocol.DeviceID) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	if _, exists := ms.metrics[deviceID]; !exists {
		return fmt.Errorf("no metrics found for device %s", deviceID.String())
	}
	
	delete(ms.metrics, deviceID)
	slog.Info("Reset connection metrics for device", "device", deviceID.String())
	return nil
}

// ResetAllMetrics resets metrics for all devices
func (ms *MetricsService) ResetAllMetrics() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	ms.metrics = make(map[protocol.DeviceID]*DeviceMetrics)
	slog.Info("Reset all connection metrics")
}

// cleanupOldMetrics removes metrics for devices that haven't been seen in a while
func (ms *MetricsService) cleanupOldMetrics() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	now := time.Now()
	threshold := 30 * 24 * time.Hour // 30 days
	
	for deviceID, metrics := range ms.metrics {
		// Check if we've seen this device recently
		lastActivity := metrics.LastConnectionAttempt
		if metrics.LastSuccessfulConnection.After(lastActivity) {
			lastActivity = metrics.LastSuccessfulConnection
		}
		if metrics.LastFailedConnection.After(lastActivity) {
			lastActivity = metrics.LastFailedConnection
		}
		
		if now.Sub(lastActivity) > threshold {
			slog.Debug("Removing old connection metrics", 
				"device", deviceID.String(),
				"lastActivity", lastActivity.Format(time.RFC3339))
			delete(ms.metrics, deviceID)
		}
	}
}

// CreateConnectionEvent creates a new connection event
func CreateConnectionEvent(deviceID protocol.DeviceID, eventType ConnectionEventType, err error, duration time.Duration, cert *tls.Certificate) *ConnectionEvent {
	event := &ConnectionEvent{
		DeviceID:  deviceID,
		EventType: eventType,
		Error:     err,
		Duration:  duration,
	}
	
	// Extract certificate information if available
	if cert != nil && len(cert.Certificate) > 0 {
		if parsedCert, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {
			event.CertInfo = &CertificateInfo{
				Subject:      parsedCert.Subject.String(),
				Issuer:       parsedCert.Issuer.String(),
				NotBefore:    parsedCert.NotBefore,
				NotAfter:     parsedCert.NotAfter,
				SerialNumber: parsedCert.SerialNumber.String(),
				IsSelfSigned: parsedCert.Subject.String() == parsedCert.Issuer.String(),
				IsValid:      time.Now().After(parsedCert.NotBefore) && time.Now().Before(parsedCert.NotAfter),
			}
		}
	}
	
	return event
}

// GetMetricsSummary returns a summary of all metrics
func (ms *MetricsService) GetMetricsSummary() map[string]interface{} {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	summary := make(map[string]interface{})
	
	totalDevices := len(ms.metrics)
	summary["totalDevices"] = totalDevices
	
	if totalDevices == 0 {
		return summary
	}
	
	var totalAttempts, totalSuccess, totalFailures, totalTLSFailures, totalCertErrors int64
	
	for _, metrics := range ms.metrics {
		totalAttempts += metrics.ConnectionAttempts
		totalSuccess += metrics.SuccessfulConnections
		totalFailures += metrics.FailedConnections
		totalTLSFailures += metrics.TLSHandshakeFailures
		totalCertErrors += metrics.CertificateErrors
	}
	
	summary["totalConnectionAttempts"] = totalAttempts
	summary["totalSuccessfulConnections"] = totalSuccess
	summary["totalFailedConnections"] = totalFailures
	summary["totalTLSHandshakeFailures"] = totalTLSFailures
	summary["totalCertificateErrors"] = totalCertErrors
	
	if totalAttempts > 0 {
		summary["overallSuccessRate"] = float64(totalSuccess) / float64(totalAttempts)
		summary["overallFailureRate"] = float64(totalFailures) / float64(totalAttempts)
		summary["overallTLSErrorRate"] = float64(totalTLSFailures) / float64(totalAttempts)
		summary["overallCertificateErrorRate"] = float64(totalCertErrors) / float64(totalAttempts)
	}
	
	return summary
}