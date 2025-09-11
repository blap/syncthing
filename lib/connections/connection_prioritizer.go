// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/config"
	"math"
)

// ConnectionPrioritizer evaluates and prioritizes connections based on multiple metrics
type ConnectionPrioritizer struct {
	cfg config.Wrapper
}

// NewConnectionPrioritizer creates a new connection prioritizer
func NewConnectionPrioritizer(cfg config.Wrapper) *ConnectionPrioritizer {
	return &ConnectionPrioritizer{
		cfg: cfg,
	}
}

// PriorityConnectionScore represents the comprehensive score for a connection
type PriorityConnectionScore struct {
	// Individual metric scores (0-100)
	LatencyScore    float64
	StabilityScore  float64
	BandwidthScore  float64
	PriorityScore   float64
	
	// Weighted composite score (0-100)
	CompositeScore  float64
	
	// Connection details
	Connection      protocol.Connection
	Priority        int
}

// EvaluateConnection evaluates a connection and returns its comprehensive score
func (cp *ConnectionPrioritizer) EvaluateConnection(conn protocol.Connection) PriorityConnectionScore {
	// Get health metrics if available
	var latencyScore float64 = 50.0
	var bandwidthScore float64 = 50.0
	
	// Try to get health monitor from connection
	if healthMonitoredConn, ok := conn.(interface{ HealthMonitor() *HealthMonitor }); ok {
		if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
			// Get detailed metrics for individual scoring
			metrics := monitor.GetConnectionQualityMetrics()
			if latencyMs, exists := metrics["latencyMs"]; exists {
				latencyScore = cp.normalizeLatencyScore(latencyMs)
			}
			if throughput, exists := metrics["throughputMbps"]; exists {
				bandwidthScore = cp.normalizeBandwidthScore(throughput)
			}
		}
	}
	
	// Get stability metrics from stability manager if available
	var stabilityScore float64 = 50.0
	
	// Priority score (lower priority value is better)
	priorityScore := cp.normalizePriorityScore(conn.Priority())
	
	// Calculate composite score with weighted factors
	// Weights can be adjusted based on importance
	latencyWeight := 0.25
	stabilityWeight := 0.25
	bandwidthWeight := 0.25
	priorityWeight := 0.25
	
	compositeScore := (latencyScore * latencyWeight) +
		(stabilityScore * stabilityWeight) +
		(bandwidthScore * bandwidthWeight) +
		(priorityScore * priorityWeight)
	
	return PriorityConnectionScore{
		LatencyScore:    latencyScore,
		StabilityScore:  stabilityScore,
		BandwidthScore:  bandwidthScore,
		PriorityScore:   priorityScore,
		CompositeScore:  compositeScore,
		Connection:      conn,
		Priority:        conn.Priority(),
	}
}

// CompareConnections compares two connections and returns true if the first is better than the second
func (cp *ConnectionPrioritizer) CompareConnections(conn1, conn2 protocol.Connection) bool {
	score1 := cp.EvaluateConnection(conn1)
	score2 := cp.EvaluateConnection(conn2)
	
	// Higher composite score is better
	return score1.CompositeScore > score2.CompositeScore
}

// SelectBestConnections selects the best N connections from a slice based on comprehensive scoring
func (cp *ConnectionPrioritizer) SelectBestConnections(connections []protocol.Connection, desiredCount int) []protocol.Connection {
	if len(connections) <= desiredCount {
		return connections
	}
	
	// Score all connections
	scores := make([]PriorityConnectionScore, len(connections))
	for i, conn := range connections {
		scores[i] = cp.EvaluateConnection(conn)
	}
	
	// Sort by composite score (highest first)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].CompositeScore < scores[j].CompositeScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	// Select top N
	result := make([]protocol.Connection, desiredCount)
	for i := 0; i < desiredCount; i++ {
		result[i] = scores[i].Connection
	}
	
	return result
}

// normalizeLatencyScore converts latency in milliseconds to a 0-100 score
// Lower latency = higher score
func (cp *ConnectionPrioritizer) normalizeLatencyScore(latencyMs float64) float64 {
	// Use exponential decay function: score = 100 * e^(-latencyMs / 50)
	score := 100.0 * math.Exp(-latencyMs/50.0)
	
	// Clamp to 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// normalizeBandwidthScore converts bandwidth in Mbps to a 0-100 score
// Higher bandwidth = higher score
func (cp *ConnectionPrioritizer) normalizeBandwidthScore(bandwidthMbps float64) float64 {
	// Assume 200 Mbps is excellent bandwidth
	score := (bandwidthMbps / 200.0) * 100.0
	
	// Clamp to 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// normalizePriorityScore converts connection priority to a 0-100 score
// Lower priority values = higher score
func (cp *ConnectionPrioritizer) normalizePriorityScore(priority int) float64 {
	// Convert priority (lower is better) to score (higher is better)
	// Assume priority range 0-1000 for normalization
	score := 100.0 - (float64(priority) / 10.0)
	
	// Clamp to 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// ShouldReplaceConnection determines if a new connection should replace an existing one
func (cp *ConnectionPrioritizer) ShouldReplaceConnection(existing, new protocol.Connection, upgradeThreshold int) bool {
	existingScore := cp.EvaluateConnection(existing)
	newScore := cp.EvaluateConnection(new)
	
	// If the new connection has a significantly better score, replace
	scoreImprovement := newScore.CompositeScore - existingScore.CompositeScore
	
	// Also consider the priority threshold from configuration
	priorityDifference := existing.Priority() - new.Priority()
	
	// Replace if either:
	// 1. The new connection has a significantly better score (more than 10 points)
	// 2. The new connection has better priority and meets the upgrade threshold
	return scoreImprovement > 10.0 || (priorityDifference >= upgradeThreshold)
}