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
	latencyWeight      = 0.3
	jitterWeight       = 0.2
	packetLossWeight   = 0.2
	throughputWeight   = 0.15
	bandwidthWeight    = 0.15
)

// BandwidthSample represents a bandwidth measurement
type BandwidthSample struct {
	Timestamp   time.Time
	BytesIn     int64
	BytesOut    int64
	Duration    time.Duration
	Throughput  float64 // Mbps
}

// Removed unused constants stableLatencyThreshold and unstableLatieThreshold (unusedfunc fix)

// HealthMonitor tracks connection health and calculates adaptive keep-alive intervals
type HealthMonitor struct {
	cfg               config.Wrapper
	deviceID          string
	mut               sync.RWMutex
	latencyHistory    []time.Duration
	jitterHistory     []time.Duration
	packetLossHistory []float64
	throughputHistory []float64
	bandwidthHistory  []BandwidthSample

	// Current health metrics
	currentLatency      time.Duration
	currentJitter       time.Duration
	currentPacketLoss   float64
	currentThroughput   float64 // Mbps
	currentBandwidthIn  float64 // Mbps
	currentBandwidthOut float64 // Mbps

	// Connection quality metrics
	connectionUptime    time.Duration
	connectionStability float64
	connectionEfficiency float64

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
		cfg:               cfg,
		deviceID:          deviceID,
		latencyHistory:    make([]time.Duration, 0, 20), // Keep last 20 measurements
		jitterHistory:     make([]time.Duration, 0, 20),
		packetLossHistory: make([]float64, 0, 20),
		throughputHistory: make([]float64, 0, 20),
		bandwidthHistory:  make([]BandwidthSample, 0, 20),
		healthScore:       50.0,              // Start with neutral score
		currentInterval:   120 * time.Second, // Default to max interval
		stopChan:          make(chan struct{}),
		connectionStability: 50.0, // Start with neutral stability
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

	// Keep only the last 20 measurements
	if len(hm.packetLossHistory) > 20 {
		hm.packetLossHistory = hm.packetLossHistory[1:]
	}

	hm.currentPacketLoss = packetLoss
	hm.updateConnectionStability()
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

	// Keep only the last 20 measurements
	if len(hm.jitterHistory) > 20 {
		hm.jitterHistory = hm.jitterHistory[1:]
	}
}

// updateHealthScore calculates the current health score based on metrics
func (hm *HealthMonitor) updateHealthScore() {
	// Normalize metrics to 0-1 scale (higher is better)
	latencyScore := hm.normalizeLatency()
	jitterScore := hm.normalizeJitter()
	packetLossScore := hm.normalizePacketLoss()
	throughputScore := hm.normalizeThroughput()
	bandwidthScore := hm.normalizeBandwidth()

	// Calculate weighted health score (0-100)
	hm.healthScore = ((latencyScore * latencyWeight) +
		(jitterScore * jitterWeight) +
		(packetLossScore * packetLossWeight) +
		(throughputScore * throughputWeight) +
		(bandwidthScore * bandwidthWeight)) * 100.0
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

// normalizeThroughput converts throughput to a 0-1 score (higher is better)
func (hm *HealthMonitor) normalizeThroughput() float64 {
	// For throughput, higher is better
	// Assume 100 Mbps is excellent throughput
	if hm.currentThroughput <= 0 {
		return 0.0
	}
	
	score := hm.currentThroughput / 100.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// normalizeBandwidth converts bandwidth to a 0-1 score (higher is better)
func (hm *HealthMonitor) normalizeBandwidth() float64 {
	// For bandwidth, higher is better
	// Consider both inbound and outbound bandwidth
	totalBandwidth := hm.currentBandwidthIn + hm.currentBandwidthOut
	
	// Assume 200 Mbps total bandwidth is excellent
	if totalBandwidth <= 0 {
		return 0.0
	}
	
	score := totalBandwidth / 200.0
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

// RecordThroughput records a new throughput measurement
func (hm *HealthMonitor) RecordThroughput(throughputMbps float64) {
	hm.mut.Lock()
	defer hm.mut.Unlock()

	hm.throughputHistory = append(hm.throughputHistory, throughputMbps)

	// Keep only the last 20 measurements
	if len(hm.throughputHistory) > 20 {
		hm.throughputHistory = hm.throughputHistory[1:]
	}

	hm.currentThroughput = throughputMbps
	hm.updateHealthScore()
	hm.updateInterval()
}

// RecordBandwidth records a new bandwidth measurement
func (hm *HealthMonitor) RecordBandwidth(bytesIn, bytesOut int64, duration time.Duration) {
	hm.mut.Lock()
	defer hm.mut.Unlock()

	// Calculate throughput in Mbps
	throughputIn := float64(bytesIn*8) / (duration.Seconds() * 1000000)
	throughputOut := float64(bytesOut*8) / (duration.Seconds() * 1000000)
	
	sample := BandwidthSample{
		Timestamp:  time.Now(),
		BytesIn:    bytesIn,
		BytesOut:   bytesOut,
		Duration:   duration,
		Throughput: throughputIn + throughputOut,
	}

	hm.bandwidthHistory = append(hm.bandwidthHistory, sample)

	// Keep only the last 20 measurements
	if len(hm.bandwidthHistory) > 20 {
		hm.bandwidthHistory = hm.bandwidthHistory[1:]
	}

	hm.currentBandwidthIn = throughputIn
	hm.currentBandwidthOut = throughputOut
	hm.updateHealthScore()
	hm.updateInterval()
}

// RecordLatencyJitter records both latency and jitter measurements
func (hm *HealthMonitor) RecordLatencyJitter(latency, jitter time.Duration) {
	hm.mut.Lock()
	defer hm.mut.Unlock()

	hm.latencyHistory = append(hm.latencyHistory, latency)
	hm.jitterHistory = append(hm.jitterHistory, jitter)

	// Keep only the last 20 measurements
	if len(hm.latencyHistory) > 20 {
		hm.latencyHistory = hm.latencyHistory[1:]
	}
	if len(hm.jitterHistory) > 20 {
		hm.jitterHistory = hm.jitterHistory[1:]
	}

	hm.currentLatency = latency
	hm.currentJitter = jitter
	hm.updateHealthScore()
	hm.updateInterval()
}

// updateConnectionStability calculates and updates the connection stability metric
func (hm *HealthMonitor) updateConnectionStability() {
	// Connection stability is based on packet loss consistency and uptime
	if len(hm.packetLossHistory) < 2 {
		return
	}

	// Calculate variance in packet loss
	var sum float64
	for _, loss := range hm.packetLossHistory {
		sum += loss
	}
	mean := sum / float64(len(hm.packetLossHistory))

	var varianceSum float64
	for _, loss := range hm.packetLossHistory {
		deviation := loss - mean
		varianceSum += deviation * deviation
	}
	variance := varianceSum / float64(len(hm.packetLossHistory))

	// Lower variance means more stable connection
	// Convert variance to stability score (0-1, where 1 is most stable)
	stability := 1.0 - math.Min(variance/100.0, 1.0)
	
	// Blend with uptime factor
	uptimeFactor := hm.calculateUptimeFactor()
	hm.connectionStability = (stability*0.7 + uptimeFactor*0.3) * 100.0
}

// calculateUptimeFactor converts connection uptime to a 0-1 stability factor
func (hm *HealthMonitor) calculateUptimeFactor() float64 {
	// This would typically be based on actual connection uptime
	// For now, we'll use a simple time-based factor
	return 0.5 // Placeholder
}

// GetConnectionQualityMetrics returns detailed connection quality metrics
func (hm *HealthMonitor) GetConnectionQualityMetrics() map[string]float64 {
	hm.mut.RLock()
	defer hm.mut.RUnlock()

	return map[string]float64{
		"latencyMs":          float64(hm.currentLatency) / float64(time.Millisecond),
		"jitterMs":           float64(hm.currentJitter) / float64(time.Millisecond),
		"packetLossPercent":  hm.currentPacketLoss,
		"throughputMbps":     hm.currentThroughput,
		"bandwidthInMbps":    hm.currentBandwidthIn,
		"bandwidthOutMbps":   hm.currentBandwidthOut,
		"healthScore":        hm.healthScore,
		"connectionStability": hm.connectionStability,
	}
}

// GetDetailedHealthReport returns a comprehensive health report
func (hm *HealthMonitor) GetDetailedHealthReport() map[string]interface{} {
	hm.mut.RLock()
	defer hm.mut.RUnlock()

	// Calculate averages
	var avgLatency time.Duration
	var avgJitter time.Duration
	var avgPacketLoss float64
	var avgThroughput float64

	if len(hm.latencyHistory) > 0 {
		var latencySum time.Duration
		for _, latency := range hm.latencyHistory {
			latencySum += latency
		}
		avgLatency = latencySum / time.Duration(len(hm.latencyHistory))
	}

	if len(hm.jitterHistory) > 0 {
		var jitterSum time.Duration
		for _, jitter := range hm.jitterHistory {
			jitterSum += jitter
		}
		avgJitter = jitterSum / time.Duration(len(hm.jitterHistory))
	}

	if len(hm.packetLossHistory) > 0 {
		var packetLossSum float64
		for _, loss := range hm.packetLossHistory {
			packetLossSum += loss
		}
		avgPacketLoss = packetLossSum / float64(len(hm.packetLossHistory))
	}

	if len(hm.throughputHistory) > 0 {
		var throughputSum float64
		for _, throughput := range hm.throughputHistory {
			throughputSum += throughput
		}
		avgThroughput = throughputSum / float64(len(hm.throughputHistory))
	}

	return map[string]interface{}{
		"current": map[string]interface{}{
			"latencyMs":        float64(hm.currentLatency) / float64(time.Millisecond),
			"jitterMs":         float64(hm.currentJitter) / float64(time.Millisecond),
			"packetLossPercent": hm.currentPacketLoss,
			"throughputMbps":    hm.currentThroughput,
			"bandwidthInMbps":   hm.currentBandwidthIn,
			"bandwidthOutMbps":  hm.currentBandwidthOut,
			"healthScore":       hm.healthScore,
		},
		"average": map[string]interface{}{
			"latencyMs":        float64(avgLatency) / float64(time.Millisecond),
			"jitterMs":         float64(avgJitter) / float64(time.Millisecond),
			"packetLossPercent": avgPacketLoss,
			"throughputMbps":    avgThroughput,
		},
		"historyLength": map[string]int{
			"latency":    len(hm.latencyHistory),
			"jitter":     len(hm.jitterHistory),
			"packetLoss": len(hm.packetLossHistory),
			"throughput": len(hm.throughputHistory),
		},
		"adaptiveIntervalSeconds": int(hm.currentInterval.Seconds()),
		"isHealthy":              hm.healthScore > 70.0,
	}
}

// Add constant for monitoring state
const (
	monitoringStateActive = "active"
)
