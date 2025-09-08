// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"math"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// Ensure HealthMonitor implements the protocol.HealthMonitorInterface
var _ protocol.HealthMonitorInterface = (*HealthMonitor)(nil)

const (
	// Default health check interval
	healthCheckInterval = 10 * time.Second
	
	// Weight factors for health score calculation
	latencyWeight  = 0.5
	jitterWeight   = 0.3
	packetLossWeight = 0.2
)

// Removed unused constants stableLatencyThreshold and unstableLatieThreshold (unusedfunc fix)

// HealthMonitor tracks connection health and calculates adaptive keep-alive intervals
type HealthMonitor struct {
	cfg            config.Wrapper
	deviceID       string
	mut            sync.RWMutex
	latencyHistory []time.Duration
	jitterHistory  []time.Duration
	packetLossHistory []float64
	
	// Current health metrics
	currentLatency  time.Duration
	currentJitter   time.Duration
	currentPacketLoss float64
	
	// Current health score (0-100, where 0 is poor health and 100 is excellent health)
	healthScore float64
	
	// Current adaptive interval
	currentInterval time.Duration
	
	// Channels for control
	stopChan chan struct{}
}

// NewHealthMonitor creates a new health monitor for a connection
func NewHealthMonitor(cfg config.Wrapper, deviceID string) *HealthMonitor {
	hm := &HealthMonitor{
		cfg:             cfg,
		deviceID:        deviceID,
		latencyHistory:  make([]time.Duration, 0, 10), // Keep last 10 measurements
		jitterHistory:   make([]time.Duration, 0, 10),
		packetLossHistory: make([]float64, 0, 10),
		healthScore:     50.0, // Start with neutral score
		currentInterval: 120 * time.Second, // Default to max interval
		stopChan:        make(chan struct{}),
	}
	
	// Initialize with default interval based on config
	opts := cfg.Options()
	if opts.AdaptiveKeepAliveMaxS > 0 {
		hm.currentInterval = time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
	}
	
	return hm
}

// Start begins monitoring the connection health
func (hm *HealthMonitor) Start() {
	go hm.monitorLoop()
}

// Stop stops monitoring the connection health
func (hm *HealthMonitor) Stop() {
	close(hm.stopChan)
}

// RecordLatency records a new latency measurement
func (hm *HealthMonitor) RecordLatency(latency time.Duration) {
	hm.mut.Lock()
	defer hm.mut.Unlock()
	
	hm.latencyHistory = append(hm.latencyHistory, latency)
	
	// Keep only the last 10 measurements
	if len(hm.latencyHistory) > 10 {
		hm.latencyHistory = hm.latencyHistory[1:]
	}
	
	hm.currentLatency = latency
	hm.updateJitter()
	hm.updateHealthScore()
	hm.updateInterval()
}

// RecordPacketLoss records a new packet loss measurement
func (hm *HealthMonitor) RecordPacketLoss(packetLoss float64) {
	hm.mut.Lock()
	defer hm.mut.Unlock()
	
	hm.packetLossHistory = append(hm.packetLossHistory, packetLoss)
	
	// Keep only the last 10 measurements
	if len(hm.packetLossHistory) > 10 {
		hm.packetLossHistory = hm.packetLossHistory[1:]
	}
	
	hm.currentPacketLoss = packetLoss
	hm.updateHealthScore()
	hm.updateInterval()
}

// GetInterval returns the current adaptive keep-alive interval
func (hm *HealthMonitor) GetInterval() time.Duration {
	hm.mut.RLock()
	defer hm.mut.RUnlock()
	return hm.currentInterval
}

// GetHealthScore returns the current health score
func (hm *HealthMonitor) GetHealthScore() float64 {
	hm.mut.RLock()
	defer hm.mut.RUnlock()
	return hm.healthScore
}

// SetHealthScore sets the health score directly (for testing purposes)
func (hm *HealthMonitor) SetHealthScore(score float64) {
	hm.mut.Lock()
	defer hm.mut.Unlock()
	hm.healthScore = score
	hm.updateInterval()
}

// updateJitter calculates the current jitter based on latency history
func (hm *HealthMonitor) updateJitter() {
	if len(hm.latencyHistory) < 2 {
		hm.currentJitter = 0
		return
	}
	
	// Calculate jitter as the average deviation from the mean
	var sum time.Duration
	for _, latency := range hm.latencyHistory {
		sum += latency
	}
	mean := time.Duration(int64(sum) / int64(len(hm.latencyHistory)))
	
	var deviationSum time.Duration
	for _, latency := range hm.latencyHistory {
		deviation := latency - mean
		if deviation < 0 {
			deviation = -deviation
		}
		deviationSum += deviation
	}
	
	hm.currentJitter = time.Duration(int64(deviationSum) / int64(len(hm.latencyHistory)))
	
	// Add to history
	hm.jitterHistory = append(hm.jitterHistory, hm.currentJitter)
	
	// Keep only the last 10 measurements
	if len(hm.jitterHistory) > 10 {
		hm.jitterHistory = hm.jitterHistory[1:]
	}
}

// updateHealthScore calculates the current health score based on metrics
func (hm *HealthMonitor) updateHealthScore() {
	// Normalize metrics to 0-1 scale (higher is better)
	latencyScore := hm.normalizeLatency()
	jitterScore := hm.normalizeJitter()
	packetLossScore := hm.normalizePacketLoss()
	
	// Calculate weighted health score (0-100)
	hm.healthScore = ((latencyScore * latencyWeight) +
		(jitterScore * jitterWeight) +
		(packetLossScore * packetLossWeight)) * 100.0
}

// normalizeLatency converts latency to a 0-1 score (higher is better)
func (hm *HealthMonitor) normalizeLatency() float64 {
	// For latency, lower is better
	latencyMs := float64(hm.currentLatency) / float64(time.Millisecond)
	
	// Use exponential decay function: score = e^(-latencyMs / 30)
	score := math.Exp(-latencyMs / 30.0)
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// normalizeJitter converts jitter to a 0-1 score (higher is better)
func (hm *HealthMonitor) normalizeJitter() float64 {
	// For jitter, lower is better
	jitterMs := float64(hm.currentJitter) / float64(time.Millisecond)
	
	// Use curve: score = e^(-jitterMs / 15)
	score := math.Exp(-jitterMs / 15.0)
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// normalizePacketLoss converts packet loss to a 0-1 score (higher is better)
func (hm *HealthMonitor) normalizePacketLoss() float64 {
	// For packet loss, lower is better
	// Use exponential decay: score = e^(-packetLoss / 1.0)
	score := math.Exp(-hm.currentPacketLoss / 1.0)
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// updateInterval adjusts the keep-alive interval based on health score
func (hm *HealthMonitor) updateInterval() {
	opts := hm.cfg.Options()
	minInterval := time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second
	maxInterval := time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
	
	// Ensure we have valid min/max values
	if minInterval <= 0 {
		minInterval = 20 * time.Second
	}
	if maxInterval <= 0 {
		maxInterval = 120 * time.Second
	}
	
	// Map health score (0-100) to interval (minInterval to maxInterval)
	// Higher health score = longer interval (less aggressive)
	// Lower health score = shorter interval (more aggressive)
	
	// Use quadratic mapping for balanced response
	// interval = min + (max-min) * (healthScore/100)^2
	healthRatio := hm.healthScore / 100.0
	intervalRange := float64(maxInterval - minInterval)
	
	// Use square to make the response more aggressive at low health scores
	// but not too aggressive at high health scores
	hm.currentInterval = minInterval + time.Duration(intervalRange*healthRatio*healthRatio)
	
	// Ensure interval is within bounds
	if hm.currentInterval < minInterval {
		hm.currentInterval = minInterval
	}
	if hm.currentInterval > maxInterval {
		hm.currentInterval = maxInterval
	}
}

// monitorLoop periodically updates health metrics
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			hm.performHealthCheck()
		case <-hm.stopChan:
			return
		}
	}
}

// performHealthCheck performs periodic health checks
func (hm *HealthMonitor) performHealthCheck() {
	// This would typically involve:
	// 1. Sending a ping and measuring RTT
	// 2. Checking for packet loss
	// 3. Updating metrics
	
	// For now, we'll just ensure the health score is updated
	// based on the latest recorded metrics
	hm.mut.Lock()
	hm.updateHealthScore()
	hm.updateInterval()
	hm.mut.Unlock()
}

// IsHealthy returns whether the connection is considered healthy
func (hm *HealthMonitor) IsHealthy() bool {
	hm.mut.RLock()
	defer hm.mut.RUnlock()
	// Consider connection healthy if health score is above 70
	return hm.healthScore > 70.0
}

// Add missing methods for enhanced functionality

// GetMonitoringState returns the current monitoring state
func (hm *HealthMonitor) GetMonitoringState() string {
	hm.mut.RLock()
	defer hm.mut.RUnlock()
	// For now, always return active state
	// In a more complex implementation, this could track different states
	return monitoringStateActive
}

// RecordActivity records connection activity to keep monitoring active
func (hm *HealthMonitor) RecordActivity() {
	// This would typically reset any idle timers or counters
	// For now, it's a placeholder to satisfy the interface
}

// Add constant for monitoring state
const (
	monitoringStateActive = "active"
)
