// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"sort"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// ConvergenceManager implements algorithms to coordinate multiple connections
// and achieve stable connection states for multipath scenarios
type ConvergenceManager struct {
	cfg      config.Wrapper
	deviceID protocol.DeviceID
	mut      sync.RWMutex
	
	// Convergence state
	connectionScores map[string]ConnectionScore
	lastEvaluation   time.Time
	convergenceState ConvergenceState
	
	// Configuration
	evaluationInterval time.Duration
	convergenceWindow  time.Duration
}

// ConnectionScore represents the quality score of a connection
type ConnectionScore struct {
	ConnectionID string
	Priority     int
	HealthScore  float64
	Stability    float64
	Latency      time.Duration
	PacketLoss   float64
	Throughput   float64 // Mbps
	Established  time.Time
	LastActive   time.Time
}

// ConvergenceState represents the current state of connection convergence
type ConvergenceState int

const (
	ConvergenceStateUnknown ConvergenceState = iota
	ConvergenceStateConverging
	ConvergenceStateStable
	ConvergenceStateDiverging
)

// ConvergenceResult contains the results of a convergence evaluation
type ConvergenceResult struct {
	State              ConvergenceState
	RecommendedAction  ConvergenceAction
	PrimaryConnection  string
	ActiveConnections  []string
	QualityMetrics     map[string]ConnectionScore
	EvaluationTime     time.Time
}

// ConvergenceAction represents a recommended action to achieve convergence
type ConvergenceAction int

const (
	ConvergenceActionNone ConvergenceAction = iota
	ConvergenceActionWait
	ConvergenceActionPromote
	ConvergenceActionDemote
	ConvergenceActionClose
	ConvergenceActionOpen
)

// NewConvergenceManager creates a new convergence manager
func NewConvergenceManager(cfg config.Wrapper, deviceID protocol.DeviceID) *ConvergenceManager {
	opts := cfg.Options()
	
	return &ConvergenceManager{
		cfg:               cfg,
		deviceID:          deviceID,
		connectionScores:  make(map[string]ConnectionScore),
		evaluationInterval: time.Second * 10,
		convergenceWindow:  time.Duration(opts.ConnectionReplacementAgeThreshold) * time.Second,
	}
}

// UpdateConnectionScore updates the score for a connection
func (cm *ConvergenceManager) UpdateConnectionScore(conn protocol.Connection) {
	cm.mut.Lock()
	defer cm.mut.Unlock()
	
	score := ConnectionScore{
		ConnectionID: conn.ConnectionID(),
		Priority:     conn.Priority(),
		Established:  conn.EstablishedAt(),
		LastActive:   time.Now(),
	}
	
	// Get health score if available
	if healthMonitoredConn, ok := conn.(interface{ HealthMonitor() *HealthMonitor }); ok {
		if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
			score.HealthScore = monitor.GetHealthScore()
			metrics := monitor.GetConnectionQualityMetrics()
			score.Latency = time.Duration(metrics["latencyMs"]) * time.Millisecond
			score.PacketLoss = metrics["packetLossPercent"]
		}
	}
	
	// Calculate stability based on connection duration
	uptime := time.Since(conn.EstablishedAt())
	score.Stability = normalizeStability(uptime)
	
	cm.connectionScores[conn.ConnectionID()] = score
}

// EvaluateConvergence evaluates the current connection state and determines
// if convergence has been achieved
func (cm *ConvergenceManager) EvaluateConvergence(connections []protocol.Connection) ConvergenceResult {
	cm.mut.Lock()
	defer cm.mut.Unlock()
	
	now := time.Now()
	cm.lastEvaluation = now
	
	// Update scores for all current connections
	for _, conn := range connections {
		cm.UpdateConnectionScore(conn)
	}
	
	// Clean up scores for closed connections
	cm.cleanupClosedConnections(connections)
	
	// If we don't have any connections, we're in unknown state
	if len(cm.connectionScores) == 0 {
		return ConvergenceResult{
			State:          ConvergenceStateUnknown,
			RecommendedAction: ConvergenceActionNone,
			EvaluationTime:    now,
		}
	}
	
	// Sort connections by quality
	sortedConnections := cm.getSortedConnections()
	
	// Determine convergence state
	state := cm.determineConvergenceState(sortedConnections)
	
	// Determine recommended action
	action := cm.determineRecommendedAction(sortedConnections, state)
	
	// Select primary connection
	var primaryConn string
	if len(sortedConnections) > 0 {
		primaryConn = sortedConnections[0].ConnectionID
	}
	
	// Select active connections
	activeConns := make([]string, len(sortedConnections))
	for i, conn := range sortedConnections {
		activeConns[i] = conn.ConnectionID
	}
	
	return ConvergenceResult{
		State:              state,
		RecommendedAction:  action,
		PrimaryConnection:  primaryConn,
		ActiveConnections:  activeConns,
		QualityMetrics:     cm.connectionScores,
		EvaluationTime:     now,
	}
}

// cleanupClosedConnections removes scores for connections that are no longer active
func (cm *ConvergenceManager) cleanupClosedConnections(activeConnections []protocol.Connection) {
	activeIDs := make(map[string]bool)
	for _, conn := range activeConnections {
		activeIDs[conn.ConnectionID()] = true
	}
	
	for id := range cm.connectionScores {
		if !activeIDs[id] {
			delete(cm.connectionScores, id)
		}
	}
}

// getSortedConnections returns connections sorted by quality score
func (cm *ConvergenceManager) getSortedConnections() []ConnectionScore {
	scores := make([]ConnectionScore, 0, len(cm.connectionScores))
	for _, score := range cm.connectionScores {
		scores = append(scores, score)
	}
	
	// Sort by composite quality score (higher is better)
	sort.Slice(scores, func(i, j int) bool {
		return cm.calculateCompositeScore(scores[i]) > cm.calculateCompositeScore(scores[j])
	})
	
	return scores
}

// calculateCompositeScore calculates a composite quality score for a connection
func (cm *ConvergenceManager) calculateCompositeScore(score ConnectionScore) float64 {
	// Weight factors for different metrics
	healthWeight := 0.4
	stabilityWeight := 0.3
	latencyWeight := 0.2
	packetLossWeight := 0.1
	
	// Normalize latency (lower is better, so invert)
	latencyScore := 1.0
	if score.Latency > 0 {
		// Assume 100ms is a reasonable baseline
		latencyScore = 100.0 / float64(score.Latency.Milliseconds())
		if latencyScore > 1.0 {
			latencyScore = 1.0
		}
	}
	
	// Normalize packet loss (lower is better, so invert)
	packetLossScore := 1.0 - (score.PacketLoss / 100.0)
	if packetLossScore < 0 {
		packetLossScore = 0
	}
	
	// Calculate composite score
	composite := (score.HealthScore/100.0)*healthWeight +
		score.Stability*stabilityWeight +
		latencyScore*latencyWeight +
		packetLossScore*packetLossWeight
	
	return composite
}

// determineConvergenceState determines the current convergence state
func (cm *ConvergenceManager) determineConvergenceState(sortedScores []ConnectionScore) ConvergenceState {
	if len(sortedScores) == 0 {
		return ConvergenceStateUnknown
	}
	
	// If we have only one connection, we're stable
	if len(sortedScores) == 1 {
		return ConvergenceStateStable
	}
	
	// Check if the top connections have similar scores (indicating divergence)
	if len(sortedScores) >= 2 {
		topScore := cm.calculateCompositeScore(sortedScores[0])
		secondScore := cm.calculateCompositeScore(sortedScores[1])
		
		// If scores are very close, we might be diverging
		scoreDiff := topScore - secondScore
		if scoreDiff < 0.1 {
			return ConvergenceStateDiverging
		}
		
		// If we've had a recent evaluation, check for stability
		if !cm.lastEvaluation.IsZero() && time.Since(cm.lastEvaluation) < cm.convergenceWindow {
			return ConvergenceStateConverging
		}
	}
	
	return ConvergenceStateStable
}

// determineRecommendedAction determines the recommended action based on convergence state
func (cm *ConvergenceManager) determineRecommendedAction(sortedScores []ConnectionScore, state ConvergenceState) ConvergenceAction {
	switch state {
	case ConvergenceStateUnknown:
		return ConvergenceActionNone
		
	case ConvergenceStateConverging:
		// During convergence, wait for stability
		return ConvergenceActionWait
		
	case ConvergenceStateDiverging:
		// When diverging, we might need to close poor connections or wait
		if cm.cfg.Options().ConnectionLimitMax > 0 && len(sortedScores) > cm.cfg.Options().ConnectionLimitMax {
			// Too many connections, close the worst one
			return ConvergenceActionClose
		}
		// Otherwise, wait for convergence
		return ConvergenceActionWait
		
	case ConvergenceStateStable:
		// When stable, check if we should promote a better connection
		if len(sortedScores) >= 2 {
			topScore := cm.calculateCompositeScore(sortedScores[0])
			secondScore := cm.calculateCompositeScore(sortedScores[1])
			
			// If there's a significantly better connection, promote it
			if secondScore > topScore+0.2 {
				return ConvergenceActionPromote
			}
		}
		return ConvergenceActionNone
		
	default:
		return ConvergenceActionNone
	}
}

// ShouldReplaceConnection determines if a new connection should replace an existing one
// based on convergence criteria
func (cm *ConvergenceManager) ShouldReplaceConnection(existingConn, newConn protocol.Connection) bool {
	cm.mut.RLock()
	defer cm.mut.RUnlock()
	
	// Get scores for both connections
	existingScore, existingExists := cm.connectionScores[existingConn.ConnectionID()]
	newScore := ConnectionScore{
		ConnectionID: newConn.ConnectionID(),
		Priority:     newConn.Priority(),
		Established:  newConn.EstablishedAt(),
		LastActive:   time.Now(),
	}
	
	// Get health metrics for new connection
	if healthMonitoredConn, ok := newConn.(interface{ HealthMonitor() *HealthMonitor }); ok {
		if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
			newScore.HealthScore = monitor.GetHealthScore()
			metrics := monitor.GetConnectionQualityMetrics()
			newScore.Latency = time.Duration(metrics["latencyMs"]) * time.Millisecond
			newScore.PacketLoss = metrics["packetLossPercent"]
		}
	}
	
	// Calculate stability for new connection
	newScore.Stability = normalizeStability(time.Since(newConn.EstablishedAt()))
	
	// If we don't have a score for the existing connection, accept the new one
	if !existingExists {
		return true
	}
	
	// Compare composite scores
	existingComposite := cm.calculateCompositeScore(existingScore)
	newComposite := cm.calculateCompositeScore(newScore)
	
	// Require a significant improvement to replace
	return newComposite > existingComposite+0.1
}

// GetConvergenceMetrics returns detailed metrics about the convergence state
func (cm *ConvergenceManager) GetConvergenceMetrics() map[string]interface{} {
	cm.mut.RLock()
	defer cm.mut.RUnlock()
	
	metrics := make(map[string]interface{})
	metrics["convergenceState"] = int(cm.convergenceState)
	metrics["connectionCount"] = len(cm.connectionScores)
	metrics["lastEvaluation"] = cm.lastEvaluation
	
	// Add individual connection metrics
	connectionMetrics := make(map[string]interface{})
	for id, score := range cm.connectionScores {
		connectionMetrics[id] = map[string]interface{}{
			"priority":    score.Priority,
			"healthScore": score.HealthScore,
			"stability":   score.Stability,
			"latencyMs":   score.Latency.Milliseconds(),
			"packetLoss":  score.PacketLoss,
			"established": score.Established,
			"lastActive":  score.LastActive,
		}
	}
	metrics["connections"] = connectionMetrics
	
	return metrics
}

// Reset resets the convergence manager state
func (cm *ConvergenceManager) Reset() {
	cm.mut.Lock()
	defer cm.mut.Unlock()
	
	cm.connectionScores = make(map[string]ConnectionScore)
	cm.lastEvaluation = time.Time{}
	cm.convergenceState = ConvergenceStateUnknown
}

// normalizeStability converts connection uptime to a 0-1 stability score
func normalizeStability(uptime time.Duration) float64 {
	// Convert to minutes
	minutes := uptime.Minutes()
	
	// Use logarithmic scaling - longer uptime is better but with diminishing returns
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

// GetPrimaryConnection selects the best connection from a list based on convergence criteria
func (cm *ConvergenceManager) GetPrimaryConnection(connections []protocol.Connection) protocol.Connection {
	if len(connections) == 0 {
		return nil
	}
	
	if len(connections) == 1 {
		return connections[0]
	}
	
	cm.mut.Lock()
	defer cm.mut.Unlock()
	
	// Update scores for all connections
	for _, conn := range connections {
		cm.UpdateConnectionScore(conn)
	}
	
	// Sort by composite score
	scores := make([]ConnectionScore, len(connections))
	for i, conn := range connections {
		if score, exists := cm.connectionScores[conn.ConnectionID()]; exists {
			scores[i] = score
		} else {
			// Create a basic score if we don't have one
			scores[i] = ConnectionScore{
				ConnectionID: conn.ConnectionID(),
				Priority:     conn.Priority(),
				Established:  conn.EstablishedAt(),
				LastActive:   time.Now(),
				HealthScore:  50.0, // Default score
				Stability:    normalizeStability(time.Since(conn.EstablishedAt())),
			}
		}
	}
	
	// Sort by composite score
	sort.Slice(scores, func(i, j int) bool {
		return cm.calculateCompositeScore(scores[i]) > cm.calculateCompositeScore(scores[j])
	})
	
	// Return the connection with the best score
	for _, conn := range connections {
		if conn.ConnectionID() == scores[0].ConnectionID {
			return conn
		}
	}
	
	// Fallback to first connection
	return connections[0]
}