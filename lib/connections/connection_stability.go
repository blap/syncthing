// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// ConnectionStabilityManager tracks connection stability metrics and prevents churn
type ConnectionStabilityManager struct {
	cfg      config.Wrapper
	mut      sync.RWMutex
	deviceID protocol.DeviceID
	
	// Stability tracking
	connectionHistory map[string]*ConnectionHistory
	shortLivedCount   int
	lastConnectionAt  time.Time
	
	// Adaptive parameters
	adaptiveReconnectInterval time.Duration
	stabilityScore            float64
}

// ConnectionHistory tracks the history of a specific connection
type ConnectionHistory struct {
	ConnectionID    string
	EstablishedAt   time.Time
	ClosedAt        time.Time
	Duration        time.Duration
	CloseReason     string
	WasShortLived   bool
	Priority        int
	HealthScore     float64
}

// StabilityMetrics contains metrics for connection stability assessment
type StabilityMetrics struct {
	ShortLivedConnectionRate float64
	AverageConnectionDuration time.Duration
	ConnectionChurnRate      float64
	StabilityScore           float64
}

// NewConnectionStabilityManager creates a new connection stability manager
func NewConnectionStabilityManager(cfg config.Wrapper, deviceID protocol.DeviceID) *ConnectionStabilityManager {
	return &ConnectionStabilityManager{
		cfg:                       cfg,
		deviceID:                  deviceID,
		connectionHistory:         make(map[string]*ConnectionHistory),
		adaptiveReconnectInterval: time.Duration(cfg.Options().ReconnectIntervalS) * time.Second,
		stabilityScore:            50.0, // Start with neutral score
	}
}

// RecordConnectionEstablished records when a connection is established
func (csm *ConnectionStabilityManager) RecordConnectionEstablished(conn protocol.Connection) {
	csm.mut.Lock()
	defer csm.mut.Unlock()
	
	now := time.Now()
	
	// Update last connection time
	csm.lastConnectionAt = now
	
	// Create connection history entry
	history := &ConnectionHistory{
		ConnectionID:  conn.ConnectionID(),
		EstablishedAt: now,
		Priority:      conn.Priority(),
		HealthScore:   50.0, // Default score
	}
	
	// If we have a health monitor, get the health score
	if healthMonitoredConn, ok := conn.(interface{ HealthMonitor() *HealthMonitor }); ok {
		if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
			history.HealthScore = monitor.GetHealthScore()
		}
	}
	
	csm.connectionHistory[conn.ConnectionID()] = history
	
	// Update stability score based on connection establishment
	csm.updateStabilityScore()
}

// RecordConnectionClosed records when a connection is closed
func (csm *ConnectionStabilityManager) RecordConnectionClosed(conn protocol.Connection, reason string) {
	csm.mut.Lock()
	defer csm.mut.Unlock()
	
	now := time.Now()
	
	// Update connection history
	if history, exists := csm.connectionHistory[conn.ConnectionID()]; exists {
		history.ClosedAt = now
		history.Duration = now.Sub(history.EstablishedAt)
		history.CloseReason = reason
		history.WasShortLived = history.Duration < shortLivedConnectionThreshold
		
		// Update short-lived connection count
		if history.WasShortLived {
			csm.shortLivedCount++
		}
	}
	
	// Update stability score based on connection closure
	csm.updateStabilityScore()
	
	// Adjust reconnect interval based on stability
	csm.adjustReconnectInterval()
}

// GetStabilityMetrics returns current stability metrics
func (csm *ConnectionStabilityManager) GetStabilityMetrics() StabilityMetrics {
	csm.mut.RLock()
	defer csm.mut.RUnlock()
	
	totalConnections := len(csm.connectionHistory)
	if totalConnections == 0 {
		return StabilityMetrics{
			StabilityScore: csm.stabilityScore,
		}
	}
	
	var totalDuration time.Duration
	var shortLivedConnections int
	
	for _, history := range csm.connectionHistory {
		totalDuration += history.Duration
		if history.WasShortLived {
			shortLivedConnections++
		}
	}
	
	averageDuration := time.Duration(int64(totalDuration) / int64(totalConnections))
	shortLivedRate := float64(shortLivedConnections) / float64(totalConnections)
	
	// Calculate churn rate (connections per minute)
	var churnRate float64
	if !csm.lastConnectionAt.IsZero() && totalConnections > 1 {
		elapsed := time.Since(csm.lastConnectionAt)
		if elapsed > 0 {
			churnRate = float64(totalConnections) / (elapsed.Minutes())
		}
	}
	
	return StabilityMetrics{
		ShortLivedConnectionRate:  shortLivedRate,
		AverageConnectionDuration: averageDuration,
		ConnectionChurnRate:       churnRate,
		StabilityScore:            csm.stabilityScore,
	}
}

// ShouldAcceptConnection determines if a new connection should be accepted
// based on stability metrics to prevent churn
func (csm *ConnectionStabilityManager) ShouldAcceptConnection(newConnPriority int) bool {
	csm.mut.RLock()
	defer csm.mut.RUnlock()
	
	// If we have good stability, be more selective
	if csm.stabilityScore > 70.0 {
		// Allow connections with better or equal priority
		return true
	}
	
	// If we have poor stability, be more conservative
	if csm.stabilityScore < 30.0 {
		// Only allow significantly better connections
		// (unless we have no connections)
		return len(csm.connectionHistory) == 0
	}
	
	// For moderate stability, use standard logic
	return true
}

// GetAdaptiveReconnectInterval returns the adaptive reconnect interval
func (csm *ConnectionStabilityManager) GetAdaptiveReconnectInterval() time.Duration {
	csm.mut.RLock()
	defer csm.mut.RUnlock()
	return csm.adaptiveReconnectInterval
}

// updateStabilityScore calculates and updates the stability score
func (csm *ConnectionStabilityManager) updateStabilityScore() {
	// Consider multiple factors:
	// 1. Short-lived connection rate (lower is better)
	// 2. Average connection duration (higher is better)
	// 3. Connection churn rate (lower is better)
	
	metrics := csm.GetStabilityMetrics()
	
	// Normalize metrics to 0-1 scale
	shortLivedScore := 1.0 - metrics.ShortLivedConnectionRate // Invert (lower short-lived rate is better)
	durationScore := normalizeDuration(metrics.AverageConnectionDuration)
	churnScore := 1.0 - (metrics.ConnectionChurnRate / 10.0) // Assume 10 connections/minute is poor
	
	// Clamp churn score to 0-1 range
	if churnScore < 0 {
		churnScore = 0
	}
	if churnScore > 1 {
		churnScore = 1
	}
	
	// Weighted average (adjust weights as needed)
	csm.stabilityScore = (shortLivedScore*0.4 + durationScore*0.3 + churnScore*0.3) * 100.0
}

// adjustReconnectInterval adapts the reconnect interval based on stability
func (csm *ConnectionStabilityManager) adjustReconnectInterval() {
	opts := csm.cfg.Options()
	baseInterval := time.Duration(opts.ReconnectIntervalS) * time.Second
	
	// Map stability score to interval adjustment
	// Higher stability score = longer intervals (less aggressive)
	// Lower stability score = shorter intervals (more aggressive)
	
	if csm.stabilityScore > 80 {
		// Very stable, increase interval
		csm.adaptiveReconnectInterval = baseInterval * 2
	} else if csm.stabilityScore > 60 {
		// Stable, slightly increase interval
		csm.adaptiveReconnectInterval = time.Duration(float64(baseInterval) * 1.5)
	} else if csm.stabilityScore < 20 {
		// Unstable, decrease interval
		csm.adaptiveReconnectInterval = baseInterval / 2
	} else if csm.stabilityScore < 40 {
		// Somewhat unstable, slightly decrease interval
		csm.adaptiveReconnectInterval = time.Duration(float64(baseInterval) * 0.75)
	} else {
		// Moderate stability, use base interval
		csm.adaptiveReconnectInterval = baseInterval
	}
	
	// Ensure interval stays within reasonable bounds
	minInterval := 5 * time.Second
	maxInterval := 5 * time.Minute
	
	if csm.adaptiveReconnectInterval < minInterval {
		csm.adaptiveReconnectInterval = minInterval
	}
	if csm.adaptiveReconnectInterval > maxInterval {
		csm.adaptiveReconnectInterval = maxInterval
	}
}

// normalizeDuration converts connection duration to a 0-1 stability score
func normalizeDuration(duration time.Duration) float64 {
	// Convert to minutes for easier scaling
	minutes := duration.Minutes()
	
	// Use logarithmic scaling - longer durations are better but with diminishing returns
	// After 30 minutes, we consider the connection very stable
	if minutes <= 0 {
		return 0.0
	}
	if minutes >= 30 {
		return 1.0
	}
	
	// Logarithmic scaling: log(minutes+1)/log(31) to map 0-30 minutes to 0-1
	return (minutes + 1) / 31.0
}

// IsConnectionStable checks if a connection is considered stable based on duration
func (csm *ConnectionStabilityManager) IsConnectionStable(conn protocol.Connection) bool {
	csm.mut.RLock()
	defer csm.mut.RUnlock()
	
	if history, exists := csm.connectionHistory[conn.ConnectionID()]; exists {
		// A connection is considered stable if it has been alive for at least
		// the short-lived threshold and has a good health score
		if !history.WasShortLived && history.HealthScore > 50.0 {
			return true
		}
	}
	
	return false
}