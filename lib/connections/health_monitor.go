// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// ConnectionHealth represents the health status of a connection
type ConnectionHealth struct {
	DeviceID          protocol.DeviceID
	Address           string
	LastError         error
	LastErrorTime     time.Time
	ErrorCategory     ErrorCategory
	ConsecutiveErrors int
	SuccessCount      int
	LastSuccessTime   time.Time
	IsHealthy         bool
}

// HealthMonitor monitors connection health and provides insights for retry strategies
type HealthMonitor struct {
	// Existing fields
	mut      sync.RWMutex
	health   map[protocol.DeviceID]map[string]*ConnectionHealth
	maxStats int // Maximum number of stats to keep per device
	
	// New fields for adaptive keep-alive
	cfg              config.Wrapper
	deviceID         string
	latencyHistory   []time.Duration
	packetLossHistory []float64
	healthScore      float64
	interval         time.Duration
	latencyMut       sync.RWMutex
	packetLossMut    sync.RWMutex
	healthScoreMut   sync.RWMutex
	intervalMut      sync.RWMutex
	started          bool
	startedMut       sync.RWMutex
}

// NewHealthMonitor creates a new health monitor (backward compatible version)
func NewHealthMonitor(maxStatsPerDevice int) *HealthMonitor {
	return &HealthMonitor{
		health:   make(map[protocol.DeviceID]map[string]*ConnectionHealth),
		maxStats: maxStatsPerDevice,
		latencyHistory:   make([]time.Duration, 0, 10),
		packetLossHistory: make([]float64, 0, 10),
		healthScore:      50.0, // Default neutral health score
		interval:         30 * time.Second, // Default interval
	}
}

// NewHealthMonitorWithConfig creates a new health monitor with config support
func NewHealthMonitorWithConfig(cfg config.Wrapper, deviceID string) *HealthMonitor {
	hm := &HealthMonitor{
		health:           make(map[protocol.DeviceID]map[string]*ConnectionHealth),
		maxStats:         100, // Default max stats per device
		cfg:              cfg,
		deviceID:         deviceID,
		latencyHistory:   make([]time.Duration, 0, 10),
		packetLossHistory: make([]float64, 0, 10),
		healthScore:      50.0, // Default neutral health score
		interval:         30 * time.Second, // Default interval
	}
	
	// Initialize with default values from config if available
	if cfg != nil {
		opts := cfg.Options()
		if opts.AdaptiveKeepAliveMinS > 0 && opts.AdaptiveKeepAliveMaxS > 0 {
			// Start with a middle value
			midSeconds := (opts.AdaptiveKeepAliveMinS + opts.AdaptiveKeepAliveMaxS) / 2
			hm.interval = time.Duration(midSeconds) * time.Second
		}
	}
	
	return hm
}

// RecordConnectionError records a connection error for a specific device and address
func (hm *HealthMonitor) RecordConnectionError(deviceID protocol.DeviceID, address string, err error) {
	hm.mut.Lock()
	defer hm.mut.Unlock()
	
	// Initialize device map if needed
	if hm.health[deviceID] == nil {
		hm.health[deviceID] = make(map[string]*ConnectionHealth)
	}
	
	// Get or create connection health record
	health, exists := hm.health[deviceID][address]
	if !exists {
		health = &ConnectionHealth{
			DeviceID:  deviceID,
			Address:   address,
			IsHealthy: true,
		}
		hm.health[deviceID][address] = health
	}
	
	// Update error information
	health.LastError = err
	health.LastErrorTime = time.Now()
	health.ErrorCategory = categorizeError(err)
	health.ConsecutiveErrors++
	health.IsHealthy = false
	
	// Limit the number of stats per device
	if len(hm.health[deviceID]) > hm.maxStats {
		// Remove the oldest entry
		oldestTime := time.Now()
		var oldestAddress string
		for addr, h := range hm.health[deviceID] {
			if h.LastErrorTime.Before(oldestTime) {
				oldestTime = h.LastErrorTime
				oldestAddress = addr
			}
		}
		if oldestAddress != "" {
			delete(hm.health[deviceID], oldestAddress)
		}
	}
	
	// Update health score based on error
	hm.updateHealthScoreWithError(err)
}

// RecordConnectionSuccess records a successful connection for a specific device and address
func (hm *HealthMonitor) RecordConnectionSuccess(deviceID protocol.DeviceID, address string) {
	hm.mut.Lock()
	defer hm.mut.Unlock()
	
	// Initialize device map if needed
	if hm.health[deviceID] == nil {
		hm.health[deviceID] = make(map[string]*ConnectionHealth)
	}
	
	// Get or create connection health record
	health, exists := hm.health[deviceID][address]
	if !exists {
		health = &ConnectionHealth{
			DeviceID: deviceID,
			Address:  address,
		}
		hm.health[deviceID][address] = health
	}
	
	// Update success information
	health.SuccessCount++
	health.LastSuccessTime = time.Now()
	health.ConsecutiveErrors = 0
	health.IsHealthy = true
	
	// Update health score based on success
	hm.updateHealthScoreWithSuccess()
}

// GetConnectionHealth returns the health status for a specific device and address
func (hm *HealthMonitor) GetConnectionHealth(deviceID protocol.DeviceID, address string) *ConnectionHealth {
	hm.mut.RLock()
	defer hm.mut.RUnlock()
	
	if deviceHealth, exists := hm.health[deviceID]; exists {
		if health, exists := deviceHealth[address]; exists {
			// Return a copy to avoid race conditions
			healthCopy := *health
			return &healthCopy
		}
	}
	
	return nil
}

// GetAllConnectionHealth returns all connection health records
func (hm *HealthMonitor) GetAllConnectionHealth() map[protocol.DeviceID]map[string]*ConnectionHealth {
	hm.mut.RLock()
	defer hm.mut.RUnlock()
	
	// Create a deep copy of the health data
	result := make(map[protocol.DeviceID]map[string]*ConnectionHealth)
	for deviceID, deviceHealth := range hm.health {
		result[deviceID] = make(map[string]*ConnectionHealth)
		for address, health := range deviceHealth {
			healthCopy := *health
			result[deviceID][address] = &healthCopy
		}
	}
	
	return result
}

// GetRetryConfigForConnection returns an adaptive retry configuration based on connection health
func (hm *HealthMonitor) GetRetryConfigForConnection(deviceID protocol.DeviceID, address string) RetryConfig {
	health := hm.GetConnectionHealth(deviceID, address)
	if health == nil {
		// No health data, use default configuration
		return DefaultRetryConfig()
	}
	
	// Use adaptive configuration based on error category
	config := AdaptiveRetryConfig(health.ErrorCategory)
	
	// Adjust based on consecutive errors
	if health.ConsecutiveErrors > 0 {
		// Increase max retries for connections with repeated errors
		config.MaxRetries = min(config.MaxRetries+health.ConsecutiveErrors, 10)
		
		// Increase base delay for connections with repeated errors
		delayMultiplier := float64(health.ConsecutiveErrors)
		if delayMultiplier > 5.0 {
			delayMultiplier = 5.0
		}
		config.BaseDelay = time.Duration(float64(config.BaseDelay) * delayMultiplier)
	}
	
	// Cap the base delay
	if config.BaseDelay > config.MaxDelay {
		config.BaseDelay = config.MaxDelay
	}
	
	return config
}

// IsConnectionHealthy checks if a connection is considered healthy
func (hm *HealthMonitor) IsConnectionHealthy(deviceID protocol.DeviceID, address string) bool {
	health := hm.GetConnectionHealth(deviceID, address)
	if health == nil {
		// No health data, assume healthy
		return true
	}
	
	return health.IsHealthy
}

// GetErrorRate calculates the error rate for a connection
func (hm *HealthMonitor) GetErrorRate(deviceID protocol.DeviceID, address string) float64 {
	health := hm.GetConnectionHealth(deviceID, address)
	if health == nil {
		// No health data, assume 0% error rate
		return 0.0
	}
	
	totalAttempts := health.SuccessCount + health.ConsecutiveErrors
	if totalAttempts == 0 {
		return 0.0
	}
	
	return float64(health.ConsecutiveErrors) / float64(totalAttempts)
}

// CleanupOldStats removes old connection health statistics
func (hm *HealthMonitor) CleanupOldStats(maxAge time.Duration) {
	hm.mut.Lock()
	defer hm.mut.Unlock()
	
	cutoffTime := time.Now().Add(-maxAge)
	
	for deviceID, deviceHealth := range hm.health {
		for address, health := range deviceHealth {
			// Remove stats that haven't been updated recently
			if health.LastErrorTime.Before(cutoffTime) && health.LastSuccessTime.Before(cutoffTime) {
				delete(deviceHealth, address)
			}
		}
		
		// Clean up empty device maps
		if len(deviceHealth) == 0 {
			delete(hm.health, deviceID)
		}
	}
}

// StartCleanupRoutine starts a background routine to periodically clean up old stats
func (hm *HealthMonitor) StartCleanupRoutine(ctx context.Context, cleanupInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				hm.CleanupOldStats(24 * time.Hour) // Keep stats for 24 hours
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Implementation of protocol.HealthMonitorInterface

// GetInterval returns the current adaptive keep-alive interval
func (hm *HealthMonitor) GetInterval() time.Duration {
	hm.intervalMut.RLock()
	defer hm.intervalMut.RUnlock()
	return hm.interval
}

// RecordLatency records a new latency measurement
func (hm *HealthMonitor) RecordLatency(latency time.Duration) {
	hm.latencyMut.Lock()
	defer hm.latencyMut.Unlock()
	
	// Add to history, keeping only the last 10 measurements
	hm.latencyHistory = append(hm.latencyHistory, latency)
	if len(hm.latencyHistory) > 10 {
		hm.latencyHistory = hm.latencyHistory[1:]
	}
	
	// Update health score and interval based on latency
	// We need to acquire packetLossMut as well to avoid deadlocks in updateHealthMetrics
	hm.packetLossMut.RLock()
	defer hm.packetLossMut.RUnlock()
	hm.updateHealthMetrics()
}

// RecordPacketLoss records a new packet loss measurement
func (hm *HealthMonitor) RecordPacketLoss(packetLoss float64) {
	hm.packetLossMut.Lock()
	defer hm.packetLossMut.Unlock()
	
	// Add to history, keeping only the last 10 measurements
	hm.packetLossHistory = append(hm.packetLossHistory, packetLoss)
	if len(hm.packetLossHistory) > 10 {
		hm.packetLossHistory = hm.packetLossHistory[1:]
	}
	
	// Update health score and interval based on packet loss
	// We need to acquire latencyMut as well to avoid deadlocks in updateHealthMetrics
	hm.latencyMut.RLock()
	defer hm.latencyMut.RUnlock()
	hm.updateHealthMetrics()
}

// GetHealthScore returns the current health score (0-100)
func (hm *HealthMonitor) GetHealthScore() float64 {
	hm.healthScoreMut.RLock()
	defer hm.healthScoreMut.RUnlock()
	return hm.healthScore
}

// SetHealthScore sets the health score (for testing purposes)
func (hm *HealthMonitor) SetHealthScore(score float64) {
	hm.healthScoreMut.Lock()
	defer hm.healthScoreMut.Unlock()
	
	// Ensure score is within bounds
	if score < 0.0 {
		score = 0.0
	}
	if score > 100.0 {
		score = 100.0
	}
	
	hm.healthScore = score
	
	// Update interval based on new health score
	interval := hm.calculateInterval(score)
	
	hm.intervalMut.Lock()
	hm.interval = interval
	hm.intervalMut.Unlock()
}

// Start begins monitoring the connection health
func (hm *HealthMonitor) Start() {
	hm.startedMut.Lock()
	defer hm.startedMut.Unlock()
	hm.started = true
}

// Stop stops monitoring the connection health
func (hm *HealthMonitor) Stop() {
	hm.startedMut.Lock()
	defer hm.startedMut.Unlock()
	hm.started = false
}

// Helper methods

// updateHealthMetrics updates the health score and interval based on latency and packet loss history
// This function assumes that the caller has already acquired the necessary locks
func (hm *HealthMonitor) updateHealthMetrics() {
	// This function should only be called when the caller has already acquired
	// latencyMut and packetLossMut locks to avoid deadlocks
	
	if len(hm.latencyHistory) == 0 && len(hm.packetLossHistory) == 0 {
		return
	}
	
	// Calculate average latency
	var avgLatency time.Duration
	if len(hm.latencyHistory) > 0 {
		var totalLatency time.Duration
		for _, latency := range hm.latencyHistory {
			totalLatency += latency
		}
		avgLatency = totalLatency / time.Duration(len(hm.latencyHistory))
	}
	
	// Calculate average packet loss
	var avgPacketLoss float64
	if len(hm.packetLossHistory) > 0 {
		var totalPacketLoss float64
		for _, packetLoss := range hm.packetLossHistory {
			totalPacketLoss += packetLoss
		}
		avgPacketLoss = totalPacketLoss / float64(len(hm.packetLossHistory))
	}
	
	// Calculate jitter (standard deviation of latency)
	var jitter time.Duration
	if len(hm.latencyHistory) > 1 {
		var variance float64
		for _, latency := range hm.latencyHistory {
			diff := float64(latency) - float64(avgLatency)
			variance += diff * diff
		}
		variance /= float64(len(hm.latencyHistory))
		jitter = time.Duration(variance)
	}
	
	// Update health score based on metrics
	healthScore := hm.calculateHealthScore(avgLatency, avgPacketLoss, jitter)
	
	hm.healthScoreMut.Lock()
	hm.healthScore = healthScore
	hm.healthScoreMut.Unlock()
	
	// Update interval based on health score
	interval := hm.calculateInterval(healthScore)
	
	hm.intervalMut.Lock()
	hm.interval = interval
	hm.intervalMut.Unlock()
}

// calculateHealthScore calculates a health score (0-100) based on network metrics
func (hm *HealthMonitor) calculateHealthScore(latency time.Duration, packetLoss float64, jitter time.Duration) float64 {
	// Normalize metrics to 0-1 scale (0 = good, 1 = bad)
	latencyScore := normalizeLatency(latency)
	packetLossScore := normalizePacketLoss(packetLoss)
	jitterScore := normalizeJitter(jitter)
	
	// Weighted average (latency 50%, packet loss 30%, jitter 20%)
	healthScore := 100.0 - (latencyScore*50.0 + packetLossScore*30.0 + jitterScore*20.0)
	
	// Ensure score is within bounds
	if healthScore < 0.0 {
		healthScore = 0.0
	}
	if healthScore > 100.0 {
		healthScore = 100.0
	}
	
	return healthScore
}

// calculateInterval calculates the adaptive keep-alive interval based on health score
func (hm *HealthMonitor) calculateInterval(healthScore float64) time.Duration {
	// Get min and max intervals from config
	minInterval := 10 * time.Second
	maxInterval := 60 * time.Second
	
	if hm.cfg != nil {
		opts := hm.cfg.Options()
		if opts.AdaptiveKeepAliveMinS > 0 {
			minInterval = time.Duration(opts.AdaptiveKeepAliveMinS) * time.Second
		}
		if opts.AdaptiveKeepAliveMaxS > 0 {
			maxInterval = time.Duration(opts.AdaptiveKeepAliveMaxS) * time.Second
		}
	}
	
	// Quadratic mapping: better health = longer intervals (less frequent pings)
	// healthScore of 100 -> maxInterval, healthScore of 0 -> minInterval
	normalizedScore := healthScore / 100.0
	interval := minInterval + time.Duration(float64(maxInterval-minInterval)*(normalizedScore*normalizedScore))
	
	// Ensure interval is within bounds
	if interval < minInterval {
		interval = minInterval
	}
	if interval > maxInterval {
		interval = maxInterval
	}
	
	return interval
}

// updateHealthScoreWithError updates the health score when an error occurs
func (hm *HealthMonitor) updateHealthScoreWithError(err error) {
	hm.healthScoreMut.Lock()
	defer hm.healthScoreMut.Unlock()
	
	// Decrease health score based on error category
	category := categorizeError(err)
	scoreDecrease := 0.0
	
	switch category {
	case ErrorCategoryConnectionReset:
		scoreDecrease = 5.0
	case ErrorCategoryTimeout:
		scoreDecrease = 10.0
	case ErrorCategoryNetworkUnreachable:
		scoreDecrease = 15.0
	case ErrorCategoryHostUnreachable:
		scoreDecrease = 15.0
	default:
		scoreDecrease = 2.0
	}
	
	hm.healthScore -= scoreDecrease
	if hm.healthScore < 0.0 {
		hm.healthScore = 0.0
	}
	
	// Update interval based on new health score
	interval := hm.calculateInterval(hm.healthScore)
	
	hm.intervalMut.Lock()
	hm.interval = interval
	hm.intervalMut.Unlock()
}

// updateHealthScoreWithSuccess updates the health score when a connection succeeds
func (hm *HealthMonitor) updateHealthScoreWithSuccess() {
	hm.healthScoreMut.Lock()
	defer hm.healthScoreMut.Unlock()
	
	// Increase health score with success
	hm.healthScore += 2.0
	if hm.healthScore > 100.0 {
		hm.healthScore = 100.0
	}
	
	// Update interval based on new health score
	interval := hm.calculateInterval(hm.healthScore)
	
	hm.intervalMut.Lock()
	hm.interval = interval
	hm.intervalMut.Unlock()
}

// normalizeLatency normalizes latency to a 0-1 scale (0 = good, 1 = bad)
func normalizeLatency(latency time.Duration) float64 {
	// Consider 10ms as excellent, 500ms as poor
	ms := float64(latency.Milliseconds())
	if ms <= 10.0 {
		return 0.0
	}
	if ms >= 500.0 {
		return 1.0
	}
	return ms / 500.0
}

// normalizePacketLoss normalizes packet loss to a 0-1 scale (0 = good, 1 = bad)
func normalizePacketLoss(packetLoss float64) float64 {
	// Consider 0% as excellent, 20% as poor
	if packetLoss <= 0.0 {
		return 0.0
	}
	if packetLoss >= 20.0 {
		return 1.0
	}
	return packetLoss / 20.0
}

// normalizeJitter normalizes jitter to a 0-1 scale (0 = good, 1 = bad)
func normalizeJitter(jitter time.Duration) float64 {
	// Consider 0ms as excellent, 100ms as poor
	ms := float64(jitter.Milliseconds())
	if ms <= 0.0 {
		return 0.0
	}
	if ms >= 100.0 {
		return 1.0
	}
	return ms / 100.0
}

// GetConnectionQualityMetrics returns the current connection quality metrics
// This method is used by the convergence manager to get latency and packet loss information
func (hm *HealthMonitor) GetConnectionQualityMetrics() map[string]float64 {
	hm.latencyMut.RLock()
	hm.packetLossMut.RLock()
	defer hm.latencyMut.RUnlock()
	defer hm.packetLossMut.RUnlock()
	
	metrics := make(map[string]float64)
	
	// Calculate average latency in milliseconds
	if len(hm.latencyHistory) > 0 {
		var totalLatency time.Duration
		for _, latency := range hm.latencyHistory {
			totalLatency += latency
		}
		avgLatency := totalLatency / time.Duration(len(hm.latencyHistory))
		metrics["latencyMs"] = float64(avgLatency.Milliseconds())
	} else {
		// Default value if no latency data
		metrics["latencyMs"] = 50.0
	}
	
	// Calculate average packet loss percentage
	if len(hm.packetLossHistory) > 0 {
		var totalPacketLoss float64
		for _, packetLoss := range hm.packetLossHistory {
			totalPacketLoss += packetLoss
		}
		avgPacketLoss := totalPacketLoss / float64(len(hm.packetLossHistory))
		metrics["packetLossPercent"] = avgPacketLoss
	} else {
		// Default value if no packet loss data
		metrics["packetLossPercent"] = 0.0
	}
	
	return metrics
}
