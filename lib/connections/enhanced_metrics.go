// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Enhanced connection metrics
var (
	// Connection success rate metrics
	metricConnectionAttemptsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "attempts_total",
		Help:      "Total number of connection attempts, per device and connection type.",
	}, []string{"device", "type"})

	metricConnectionSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "success_total",
		Help:      "Total number of successful connections, per device and connection type.",
	}, []string{"device", "type"})

	metricConnectionFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "failure_total",
		Help:      "Total number of failed connections, per device and connection type.",
	}, []string{"device", "type", "reason"})

	// Connection establishment time metrics (in seconds)
	metricConnectionEstablishmentTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "establishment_duration_seconds",
		Help:      "Time taken to establish connections, per device and connection type.",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~52s
	}, []string{"device", "type"})

	// Port mapping success rate metrics
	metricPortMappingAttemptsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "port_mapping_attempts_total",
		Help:      "Total number of port mapping attempts, per protocol and port type.",
	}, []string{"protocol", "port_type"})

	metricPortMappingSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "port_mapping_success_total",
		Help:      "Total number of successful port mappings, per protocol and port type.",
	}, []string{"protocol", "port_type"})

	metricPortMappingFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "port_mapping_failure_total",
		Help:      "Total number of failed port mappings, per protocol and port type.",
	}, []string{"protocol", "port_type", "reason"})

	// Standard port usage metrics
	metricStandardPortUsage = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "standard_port_usage_total",
		Help:      "Total number of connections using standard port 22000, per protocol.",
	}, []string{"protocol"})

	metricRandomPortUsage = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "random_port_usage_total",
		Help:      "Total number of connections using random ports, per protocol.",
	}, []string{"protocol"})

	// Connection stability metrics
	metricConnectionStabilityScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "syncthing",
		Subsystem: "connections",
		Name:      "stability_score",
		Help:      "Connection stability score (0-100), per device and connection type.",
	}, []string{"device", "type"})

	// Discovery success rate metrics
	metricDiscoveryAttemptsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "discovery",
		Name:      "attempts_total",
		Help:      "Total number of discovery attempts, per device and discovery type.",
	}, []string{"device", "type"})

	metricDiscoverySuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "discovery",
		Name:      "success_total",
		Help:      "Total number of successful discoveries, per device and discovery type.",
	}, []string{"device", "type"})

	metricDiscoveryFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "syncthing",
		Subsystem: "discovery",
		Name:      "failure_total",
		Help:      "Total number of failed discoveries, per device and discovery type.",
	}, []string{"device", "type", "reason"})
)

// ConnectionMetricsTracker tracks connection metrics in memory for real-time reporting
type ConnectionMetricsTracker struct {
	mut sync.RWMutex
	
	// Connection timing tracking
	connectionStartTimes map[string]time.Time // connectionID -> startTime
	
	// Success rate tracking
	attempts map[string]int // deviceID:type -> count
	success  map[string]int // deviceID:type -> count
	failures map[string]map[string]int // deviceID:type -> reason -> count
}

// NewConnectionMetricsTracker creates a new connection metrics tracker
func NewConnectionMetricsTracker() *ConnectionMetricsTracker {
	return &ConnectionMetricsTracker{
		connectionStartTimes: make(map[string]time.Time),
		attempts:            make(map[string]int),
		success:             make(map[string]int),
		failures:            make(map[string]map[string]int),
	}
}

// RecordConnectionAttempt records a connection attempt
func (cmt *ConnectionMetricsTracker) RecordConnectionAttempt(deviceID, connType string) {
	cmt.mut.Lock()
	defer cmt.mut.Unlock()
	
	key := deviceID + ":" + connType
	cmt.attempts[key]++
	
	// Update Prometheus metrics
	metricConnectionAttemptsTotal.WithLabelValues(deviceID, connType).Inc()
}

// RecordConnectionStart records when a connection attempt starts (for timing)
func (cmt *ConnectionMetricsTracker) RecordConnectionStart(connectionID string) {
	cmt.mut.Lock()
	defer cmt.mut.Unlock()
	
	cmt.connectionStartTimes[connectionID] = time.Now()
}

// RecordConnectionSuccess records a successful connection
func (cmt *ConnectionMetricsTracker) RecordConnectionSuccess(deviceID, connType, connectionID string) {
	cmt.mut.Lock()
	defer cmt.mut.Unlock()
	
	key := deviceID + ":" + connType
	cmt.success[key]++
	
	// Calculate and record establishment time
	if startTime, exists := cmt.connectionStartTimes[connectionID]; exists {
		duration := time.Since(startTime).Seconds()
		metricConnectionEstablishmentTime.WithLabelValues(deviceID, connType).Observe(duration)
		delete(cmt.connectionStartTimes, connectionID)
	}
	
	// Update Prometheus metrics
	metricConnectionSuccessTotal.WithLabelValues(deviceID, connType).Inc()
}

// RecordConnectionFailure records a failed connection
func (cmt *ConnectionMetricsTracker) RecordConnectionFailure(deviceID, connType, reason, connectionID string) {
	cmt.mut.Lock()
	defer cmt.mut.Unlock()
	
	key := deviceID + ":" + connType
	
	// Update failure counts
	if cmt.failures[key] == nil {
		cmt.failures[key] = make(map[string]int)
	}
	cmt.failures[key][reason]++
	
	// Remove timing data
	delete(cmt.connectionStartTimes, connectionID)
	
	// Update Prometheus metrics
	metricConnectionFailureTotal.WithLabelValues(deviceID, connType, reason).Inc()
}

// GetConnectionSuccessRate calculates the connection success rate for a device and connection type
func (cmt *ConnectionMetricsTracker) GetConnectionSuccessRate(deviceID, connType string) float64 {
	cmt.mut.RLock()
	defer cmt.mut.RUnlock()
	
	key := deviceID + ":" + connType
	attempts := cmt.attempts[key]
	if attempts == 0 {
		return 0.0
	}
	
	success := cmt.success[key]
	return float64(success) / float64(attempts)
}

// RecordPortMappingAttempt records a port mapping attempt
func (cmt *ConnectionMetricsTracker) RecordPortMappingAttempt(protocol, portType string) {
	metricPortMappingAttemptsTotal.WithLabelValues(protocol, portType).Inc()
}

// RecordPortMappingSuccess records a successful port mapping
func (cmt *ConnectionMetricsTracker) RecordPortMappingSuccess(protocol, portType string) {
	metricPortMappingSuccessTotal.WithLabelValues(protocol, portType).Inc()
}

// RecordPortMappingFailure records a failed port mapping
func (cmt *ConnectionMetricsTracker) RecordPortMappingFailure(protocol, portType, reason string) {
	metricPortMappingFailureTotal.WithLabelValues(protocol, portType, reason).Inc()
}

// RecordStandardPortUsage records usage of the standard port (22000)
func (cmt *ConnectionMetricsTracker) RecordStandardPortUsage(protocol string) {
	metricStandardPortUsage.WithLabelValues(protocol).Inc()
}

// RecordRandomPortUsage records usage of a random port
func (cmt *ConnectionMetricsTracker) RecordRandomPortUsage(protocol string) {
	metricRandomPortUsage.WithLabelValues(protocol).Inc()
}

// RecordConnectionStabilityScore records a connection stability score
func (cmt *ConnectionMetricsTracker) RecordConnectionStabilityScore(deviceID, connType string, score float64) {
	metricConnectionStabilityScore.WithLabelValues(deviceID, connType).Set(score)
}

// RecordDiscoveryAttempt records a discovery attempt
func (cmt *ConnectionMetricsTracker) RecordDiscoveryAttempt(deviceID, discoveryType string) {
	metricDiscoveryAttemptsTotal.WithLabelValues(deviceID, discoveryType).Inc()
}

// RecordDiscoverySuccess records a successful discovery
func (cmt *ConnectionMetricsTracker) RecordDiscoverySuccess(deviceID, discoveryType string) {
	metricDiscoverySuccessTotal.WithLabelValues(deviceID, discoveryType).Inc()
}

// RecordDiscoveryFailure records a failed discovery
func (cmt *ConnectionMetricsTracker) RecordDiscoveryFailure(deviceID, discoveryType, reason string) {
	metricDiscoveryFailureTotal.WithLabelValues(deviceID, discoveryType, reason).Inc()
}

// GetMetricsSummary returns a summary of all connection metrics
func (cmt *ConnectionMetricsTracker) GetMetricsSummary() map[string]interface{} {
	cmt.mut.RLock()
	defer cmt.mut.RUnlock()
	
	summary := make(map[string]interface{})
	
	// Add success rates
	successRates := make(map[string]float64)
	for key := range cmt.attempts {
		parts := strings.Split(key, ":")
		if len(parts) == 2 {
			deviceID := parts[0]
			connType := parts[1]
			successRates[key] = cmt.GetConnectionSuccessRate(deviceID, connType)
		}
	}
	summary["successRates"] = successRates
	
	// Add attempt counts
	summary["attempts"] = cmt.attempts
	summary["success"] = cmt.success
	
	// Add failure counts
	summary["failures"] = cmt.failures
	
	return summary
}