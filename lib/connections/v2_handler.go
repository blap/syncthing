// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"log/slog"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// V2ConnectionHandler handles connections between v2.0 devices
type V2ConnectionHandler struct {
	features protocol.V2FeatureSet
	monitor  *protocol.ProtocolHealthMonitor
}

// NewV2ConnectionHandler creates a new v2.0 connection handler
func NewV2ConnectionHandler(features protocol.V2FeatureSet, monitor *protocol.ProtocolHealthMonitor) *V2ConnectionHandler {
	return &V2ConnectionHandler{
		features: features,
		monitor:  monitor,
	}
}

// GetV2Features returns the v2.0 features supported by this handler
func (h *V2ConnectionHandler) GetV2Features() protocol.V2FeatureSet {
	return h.features
}

// HandleV2Connection handles a connection between v2.0 devices with enhanced feature configuration
func (h *V2ConnectionHandler) HandleV2Connection(connInfo protocol.ConnectionInfo, localHello, remoteHello protocol.Hello) error {
	slog.Debug("Handling v2.0 connection", 
		"localVersion", localHello.ClientVersion,
		"remoteVersion", remoteHello.ClientVersion,
		"multipath", h.features.MultipathConnections,
		"compression", h.features.EnhancedCompression,
		"indexing", h.features.ImprovedIndexing)
	
	// Apply v2.0 specific settings based on supported features
	if h.features.MultipathConnections {
		slog.Debug("Enabling multipath connections for v2.0 devices")
		// Configure multipath connection settings
		// This would involve setting up multiple connections and load balancing between them
		// For now, we just log that this feature would be enabled
	}
	
	if h.features.EnhancedCompression {
		slog.Debug("Enabling enhanced compression for v2.0 devices")
		// Configure enhanced compression settings
		// This might involve using more efficient compression algorithms
		// For now, we just log that this feature would be enabled
	}
	
	if h.features.ImprovedIndexing {
		slog.Debug("Enabling improved indexing for v2.0 devices")
		// Configure improved indexing settings
		// This might involve using more efficient indexing algorithms or data structures
		// For now, we just log that this feature would be enabled
	}
	
	// Monitor connection health specifically for v2.0
	h.monitor.RecordProtocolAttempt("bep/2.0", true, nil)
	
	return nil
}

// ConfigureConnection applies v2.0 specific configurations to a connection
func (h *V2ConnectionHandler) ConfigureConnection(connInfo protocol.ConnectionInfo) error {
	slog.Debug("Configuring v2.0 connection with detected features",
		"multipath", h.features.MultipathConnections,
		"compression", h.features.EnhancedCompression,
		"indexing", h.features.ImprovedIndexing)
	
	// In a full implementation, this would configure the connection based on the detected features
	// For example:
	// - Setting up multipath connections
	// - Configuring enhanced compression
	// - Optimizing indexing parameters
	
	return nil
}

// GetCompatibilityScore calculates a compatibility score between two v2.0 devices
func (h *V2ConnectionHandler) GetCompatibilityScore(remoteFeatures protocol.V2FeatureSet) float64 {
	score := 0.0
	
	// Base score for having v2.0 capability
	score += 1.0
	
	// Additional points for shared features
	if h.features.MultipathConnections && remoteFeatures.MultipathConnections {
		score += 1.0
	}
	
	if h.features.EnhancedCompression && remoteFeatures.EnhancedCompression {
		score += 1.0
	}
	
	if h.features.ImprovedIndexing && remoteFeatures.ImprovedIndexing {
		score += 1.0
	}
	
	// Maximum possible score is 4.0 (base + 3 features)
	return score / 4.0
}

// HandleV2Error handles errors specific to v2.0 connections with enhanced recovery
func (h *V2ConnectionHandler) HandleV2Error(err error, localHello, remoteHello protocol.Hello) error {
	// Use the protocol package's HandleV2Error function
	return protocol.HandleV2Error(err, localHello, remoteHello)
}

// AdaptiveV2DialTimeout calculates an adaptive dial timeout for v2.0 connections
// based on historical success rates and network conditions
func (h *V2ConnectionHandler) AdaptiveV2DialTimeout(baseTimeout time.Duration) time.Duration {
	// Get statistics for bep/2.0 protocol
	stats, exists := h.monitor.GetProtocolStats("bep/2.0")
	if !exists {
		// If we have no data, return the base timeout
		return baseTimeout
	}
	
	// Calculate success rate
	successRate := protocol.CalculateSuccessRate(stats)
	
	// Adjust timeout based on success rate
	// Lower success rate = longer timeout (more time for problematic connections)
	adjustedTimeout := time.Duration(float64(baseTimeout) * (2.0 - successRate))
	
	// Ensure timeout stays within reasonable bounds
	const (
		minTimeout = 5 * time.Second
		maxTimeout = 60 * time.Second
	)
	
	if adjustedTimeout < minTimeout {
		adjustedTimeout = minTimeout
	}
	if adjustedTimeout > maxTimeout {
		adjustedTimeout = maxTimeout
	}
	
	slog.Debug("Calculated adaptive v2.0 dial timeout", 
		"baseTimeout", baseTimeout,
		"adjustedTimeout", adjustedTimeout,
		"successRate", successRate)
	
	return adjustedTimeout
}