// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ProtocolCompatibility represents the compatibility between two protocol versions
type ProtocolCompatibility struct {
	LocalVersion  string
	RemoteVersion string
	Compatible    bool
	Preferred     string
}

// CompatibilityMatrix defines which protocol versions are compatible
var compatibilityMatrix = []ProtocolCompatibility{
	{"v2.0", "v2.0", true, "bep/2.0"},
	{"v2.0", "v1.0", true, "bep/1.0"},
	{"v1.0", "v2.0", true, "bep/1.0"},
	{"v1.0", "v1.0", true, "bep/1.0"},
}

// V2FeatureSet represents the features supported in v2.0
type V2FeatureSet struct {
	MultipathConnections bool
	EnhancedCompression  bool
	ImprovedIndexing     bool
}

// ProtocolStats tracks statistics for a specific protocol
type ProtocolStats struct {
	SuccessCount   int64
	FailureCount   int64
	LastFailure    time.Time
	FailureReasons map[string]int
	AverageLatency time.Duration
}

// ProtocolHealthMonitor tracks the health of different protocols
type ProtocolHealthMonitor struct {
	ProtocolStats map[string]*ProtocolStats
	mutex         sync.RWMutex
}

// NewProtocolHealthMonitor creates a new protocol health monitor
func NewProtocolHealthMonitor() *ProtocolHealthMonitor {
	return &ProtocolHealthMonitor{
		ProtocolStats: make(map[string]*ProtocolStats),
	}
}

// RecordProtocolAttempt records a protocol connection attempt
func (phm *ProtocolHealthMonitor) RecordProtocolAttempt(protocol string, success bool, err error) {
	phm.mutex.Lock()
	defer phm.mutex.Unlock()

	stats, exists := phm.ProtocolStats[protocol]
	if !exists {
		stats = &ProtocolStats{
			FailureReasons: make(map[string]int),
		}
		phm.ProtocolStats[protocol] = stats
	}

	if success {
		stats.SuccessCount++
		slog.Debug("Protocol connection successful", "protocol", protocol, "successCount", stats.SuccessCount)
	} else {
		stats.FailureCount++
		stats.LastFailure = time.Now()
		if err != nil {
			reason := err.Error()
			stats.FailureReasons[reason]++
			slog.Debug("Protocol connection failed", "protocol", protocol, "reason", reason, "failureCount", stats.FailureCount)
		}
	}
}

// GetProtocolStats returns the statistics for a specific protocol
func (phm *ProtocolHealthMonitor) GetProtocolStats(protocol string) (*ProtocolStats, bool) {
	phm.mutex.RLock()
	defer phm.mutex.RUnlock()

	stats, exists := phm.ProtocolStats[protocol]
	return stats, exists
}

// SelectOptimalProtocol selects the optimal protocol based on historical success rates with enhanced logic
func (phm *ProtocolHealthMonitor) SelectOptimalProtocol(deviceID DeviceID) string {
	phm.mutex.RLock()
	defer phm.mutex.RUnlock()

	// Calculate success rates for each protocol
	v2Stats, hasV2 := phm.ProtocolStats["bep/2.0"]
	v1Stats, hasV1 := phm.ProtocolStats["bep/1.0"]

	// If we have no data, prefer v2.0 but fall back to v1.0
	if !hasV2 && !hasV1 {
		slog.Debug("No protocol statistics available, defaulting to bep/2.0", "device", deviceID.String())
		return "bep/2.0" // Try v2.0 first
	}

	// Calculate success rates
	v2Rate := CalculateSuccessRate(v2Stats)
	v1Rate := CalculateSuccessRate(v1Stats)

	// Get total attempts for each protocol
	v2Attempts := int64(0)
	v1Attempts := int64(0)
	
	if v2Stats != nil {
		v2Attempts = v2Stats.SuccessCount + v2Stats.FailureCount
	}
	
	if v1Stats != nil {
		v1Attempts = v1Stats.SuccessCount + v1Stats.FailureCount
	}

	slog.Debug("Protocol success rates", 
		"device", deviceID.String(),
		"bep/2.0_rate", v2Rate,
		"bep/1.0_rate", v1Rate,
		"bep/2.0_attempts", v2Attempts,
		"bep/1.0_attempts", v1Attempts)

	// If we have very few attempts for either protocol, give v2.0 a chance
	// This helps with new connections where we don't have much data yet
	minAttemptsForConfidence := int64(3)
	if v2Attempts < minAttemptsForConfidence && v1Attempts < minAttemptsForConfidence {
		slog.Debug("Insufficient data for both protocols, giving v2.0 a chance", 
			"device", deviceID.String())
		return "bep/2.0"
	}

	// Prefer the protocol with higher success rate, but with a threshold
	// Only prefer v1.0 if it has a significantly higher success rate AND sufficient attempts
	if v1Rate > 0.9 && v1Rate > v2Rate && v1Attempts >= minAttemptsForConfidence {
		slog.Debug("Selecting bep/1.0 based on high success rate", 
			"device", deviceID.String(), 
			"rate", v1Rate,
			"attempts", v1Attempts)
		return "bep/1.0"
	} else if v2Rate > 0.8 && v2Attempts >= minAttemptsForConfidence {
		slog.Debug("Selecting bep/2.0 based on good success rate", 
			"device", deviceID.String(), 
			"rate", v2Rate,
			"attempts", v2Attempts)
		return "bep/2.0"
	}

	// If both have poor success rates but we have more confidence in v2.0 data, try v2.0
	if v2Attempts > v1Attempts*2 && v2Attempts >= minAttemptsForConfidence {
		slog.Debug("Selecting bep/2.0 based on more data points", 
			"device", deviceID.String(),
			"v2_attempts", v2Attempts,
			"v1_attempts", v1Attempts)
		return "bep/2.0"
	}

	// Default fallback
	slog.Debug("Defaulting to bep/2.0", "device", deviceID.String())
	return "bep/2.0"
}

// CalculateSuccessRate calculates the success rate for protocol statistics
func CalculateSuccessRate(stats *ProtocolStats) float64 {
	if stats == nil {
		return 0.0
	}

	total := stats.SuccessCount + stats.FailureCount
	if total == 0 {
		return 1.0 // No data, assume success
	}

	return float64(stats.SuccessCount) / float64(total)
}

// CheckCompatibility checks if two protocol versions are compatible
func CheckCompatibility(localVersion, remoteVersion string) (compatible bool, preferredProtocol string) {
	// First check the compatibility matrix
	for _, entry := range compatibilityMatrix {
		if entry.LocalVersion == localVersion && entry.RemoteVersion == remoteVersion {
			return entry.Compatible, entry.Preferred
		}
	}

	// If not found in matrix, use fallback logic
	slog.Debug("Compatibility not found in matrix, using fallback logic", 
		"local", localVersion, 
		"remote", remoteVersion)

	// Default to compatible with bep/1.0 for maximum compatibility
	return true, "bep/1.0"
}

// NegotiateProtocol negotiates the best protocol to use based on connection state
func NegotiateProtocol(connectionProtocol string, localVersion, remoteVersion string) string {
	// Primary preference based on what was negotiated during TLS handshake
	if connectionProtocol == "bep/2.0" || connectionProtocol == "bep/1.0" {
		slog.Debug("Using negotiated protocol", "protocol", connectionProtocol)
		return connectionProtocol
	}

	// Fallback to compatibility check
	compatible, preferred := CheckCompatibility(localVersion, remoteVersion)
	if compatible {
		slog.Debug("Using preferred protocol from compatibility check", "protocol", preferred)
		return preferred
	}

	// Ultimate fallback - default to v1.0 for maximum compatibility
	slog.Debug("Falling back to bep/1.0 for maximum compatibility")
	return "bep/1.0"
}

// Enhanced v2.0 protocol negotiation
func negotiateV2Protocol(localHello, remoteHello Hello) (string, error) {
	// Check if both ends are v2.0 capable
	localIsV2 := isV2Client(localHello.ClientVersion)
	remoteIsV2 := isV2Client(remoteHello.ClientVersion)
	
	if localIsV2 && remoteIsV2 {
		// Both are v2.0, prefer bep/2.0
		slog.Debug("Both devices are v2.0 compatible, using bep/2.0")
		return "bep/2.0", nil
	}
	
	// If only one is v2.0, check if we can downgrade gracefully
	if localIsV2 || remoteIsV2 {
		slog.Debug("One device is v2.0, checking compatibility", 
			"localV2", localIsV2, 
			"remoteV2", remoteIsV2)
		
		// Check compatibility matrix
		compatible, preferred := CheckCompatibility(
			localHello.ClientVersion, 
			remoteHello.ClientVersion)
			
		if compatible {
			return preferred, nil
		}
		
		// If not compatible, fall back to bep/1.0
		return "bep/1.0", fmt.Errorf("v2.0 devices not fully compatible, falling back to bep/1.0")
	}
	
	// Neither is v2.0, use bep/1.0
	return "bep/1.0", nil
}

// DetectV2Features detects which v2.0 features are supported by both devices
func DetectV2Features(localHello, remoteHello Hello) V2FeatureSet {
	localFeatures := parseV2Features(localHello.ClientVersion)
	remoteFeatures := parseV2Features(remoteHello.ClientVersion)
	
	// Only enable features supported by both devices
	return V2FeatureSet{
		MultipathConnections: localFeatures.MultipathConnections && remoteFeatures.MultipathConnections,
		EnhancedCompression:  localFeatures.EnhancedCompression && remoteFeatures.EnhancedCompression,
		ImprovedIndexing:     localFeatures.ImprovedIndexing && remoteFeatures.ImprovedIndexing,
	}
}

// parseV2Features parses v2.0 features from version string with more precise detection
func parseV2Features(version string) V2FeatureSet {
	// Parse version to determine supported features
	// This could be based on specific version numbers or feature flags
	major, minor, patch, err := parseSemVer(version)
	if err != nil {
		// If we can't parse the version, be conservative with features
		slog.Debug("Could not parse version, using conservative feature set", "version", version, "error", err)
		return V2FeatureSet{} // Default to no v2.0 features
	}
	
	// More precise feature availability based on version:
	// Multipath connections available from v2.0.0
	// Enhanced compression from v2.1.0
	// Improved indexing from v2.2.0
	features := V2FeatureSet{
		MultipathConnections: major >= 2,
		EnhancedCompression:  major > 2 || (major == 2 && minor >= 1),
		ImprovedIndexing:     major > 2 || (major == 2 && minor >= 2),
	}
	
	slog.Debug("Parsed v2 features", 
		"version", version,
		"major", major,
		"minor", minor,
		"patch", patch,
		"multipath", features.MultipathConnections,
		"compression", features.EnhancedCompression,
		"indexing", features.ImprovedIndexing)
	
	return features
}

// HandleV2Error handles errors specific to v2.0 connections with enhanced recovery
func HandleV2Error(err error, localHello, remoteHello Hello) error {
	// Log detailed error information for v2.0 connections
	slog.Warn("v2.0 connection error", 
		"localVersion", localHello.ClientVersion,
		"remoteVersion", remoteHello.ClientVersion,
		"error", err)
	
	// Check if this is a known v2.0 issue that can be recovered from
	if isErrorRecoverable(err) {
		slog.Info("Attempting recovery for v2.0 connection error")
		// Implement recovery logic
		return attemptV2Recovery(err, localHello, remoteHello)
	}
	
	// If not recoverable, log and return
	slog.Error("Unrecoverable v2.0 connection error", 
		"localVersion", localHello.ClientVersion,
		"remoteVersion", remoteHello.ClientVersion,
		"error", err)
	
	return err
}

// NegotiateFeaturesForMixedVersions negotiates features between v1.x and v2.x devices with graceful degradation
func NegotiateFeaturesForMixedVersions(localHello, remoteHello Hello) (string, V2FeatureSet, error) {
	localIsV2 := isV2Client(localHello.ClientVersion)
	remoteIsV2 := isV2Client(remoteHello.ClientVersion)
	
	slog.Debug("Feature negotiation for mixed versions", 
		"localVersion", localHello.ClientVersion,
		"remoteVersion", remoteHello.ClientVersion,
		"localIsV2", localIsV2,
		"remoteIsV2", remoteIsV2)
	
	// If both are v2.0, use full feature set
	if localIsV2 && remoteIsV2 {
		features := DetectV2Features(localHello, remoteHello)
		slog.Debug("Both devices are v2.0, using bep/2.0 with features", 
			"multipath", features.MultipathConnections,
			"compression", features.EnhancedCompression,
			"indexing", features.ImprovedIndexing)
		return "bep/2.0", features, nil
	}
	
	// If only one is v2.0, negotiate compatible features
	if localIsV2 || remoteIsV2 {
		// Use conservative feature set for mixed environments
		conservativeFeatures := V2FeatureSet{
			MultipathConnections: false, // Disable advanced features for mixed versions
			EnhancedCompression:  false,
			ImprovedIndexing:     false,
		}
		
		slog.Debug("Mixed version environment, using conservative feature set")
		return "bep/1.0", conservativeFeatures, nil
	}
	
	// Both are v1.x, use bep/1.0 with no v2 features
	slog.Debug("Both devices are v1.x, using bep/1.0")
	return "bep/1.0", V2FeatureSet{}, nil
}

// GracefulProtocolFallback handles protocol fallback for mixed-version environments
func GracefulProtocolFallback(localHello, remoteHello Hello, initialError error) (string, error) {
	slog.Warn("Attempting graceful protocol fallback", 
		"localVersion", localHello.ClientVersion,
		"remoteVersion", remoteHello.ClientVersion,
		"initialError", initialError)
	
	// Try bep/1.0 as fallback
	slog.Info("Falling back to bep/1.0 for compatibility")
	return "bep/1.0", nil
}

// isErrorRecoverable checks if a v2.0 error is recoverable with more comprehensive detection
func isErrorRecoverable(err error) bool {
	// Define which errors are recoverable for v2.0 connections
	// For example, temporary network issues, timeout errors, etc.
	errorStr := err.Error()
	
	// Common recoverable network errors
	recoverablePatterns := []string{
		"timeout",
		"temporary",
		"eof",
		"connection reset",
		"broken pipe",
		"network is unreachable",
		"no route to host",
		"connection refused",
	}
	
	for _, pattern := range recoverablePatterns {
		if strings.Contains(strings.ToLower(errorStr), pattern) {
			return true
		}
	}
	
	return false
}

// attemptV2Recovery attempts to recover from a v2.0 connection error with multiple strategies
func attemptV2Recovery(err error, localHello, remoteHello Hello) error {
	slog.Info("Attempting v2.0 connection recovery",
		"localVersion", localHello.ClientVersion,
		"remoteVersion", remoteHello.ClientVersion)
	
	// Strategy 1: Check if this is a TLS handshake issue that might be resolved by retrying
	if strings.Contains(err.Error(), "TLS") || strings.Contains(err.Error(), "handshake") {
		slog.Info("TLS handshake issue detected, suggesting protocol fallback")
		// In a full implementation, we might try with different TLS settings
		// For now, we'll just log and return the original error
		return fmt.Errorf("tls handshake failed between %s and %s: %w", 
			localHello.ClientVersion, remoteHello.ClientVersion, err)
	}
	
	// Strategy 2: For EOF errors, which might indicate the remote side closed connection
	// possibly due to protocol mismatch, suggest falling back to bep/1.0
	if strings.Contains(err.Error(), "EOF") {
		slog.Info("EOF error detected, suggesting bep/1.0 fallback")
		// In a full implementation, we might automatically retry with bep/1.0
		return fmt.Errorf("connection closed unexpectedly between %s and %s (possibly protocol mismatch): %w", 
			localHello.ClientVersion, remoteHello.ClientVersion, err)
	}
	
	// Strategy 3: For timeout errors, suggest increasing timeout values
	if strings.Contains(err.Error(), "timeout") {
		slog.Info("Timeout error detected, suggesting increased timeout values")
		// In a full implementation, we might retry with increased timeouts
		return fmt.Errorf("connection timeout between %s and %s: %w", 
			localHello.ClientVersion, remoteHello.ClientVersion, err)
	}
	
	// For now, just return the original error
	// In a full implementation, we would attempt recovery
	return err
}