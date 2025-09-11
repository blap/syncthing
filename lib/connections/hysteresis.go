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

// HysteresisController prevents rapid switching between connections by implementing
// hysteresis - a system that resists change unless the change is significant enough
type HysteresisController struct {
	cfg      config.Wrapper
	deviceID protocol.DeviceID
	mut      sync.RWMutex
	
	// Hysteresis state
	lastSwitchTime     time.Time
	switchCount        int
	primaryConnection  protocol.Connection
	proposedConnection protocol.Connection
	
	// Configuration
	switchCooldownPeriod time.Duration
	maxSwitchesPerMinute int
	hysteresisThreshold  int // Priority difference required to switch
}

// NewHysteresisController creates a new hysteresis controller
func NewHysteresisController(cfg config.Wrapper, deviceID protocol.DeviceID) *HysteresisController {
	opts := cfg.Options()
	
	return &HysteresisController{
		cfg:                  cfg,
		deviceID:             deviceID,
		switchCooldownPeriod: time.Duration(opts.ConnectionReplacementAgeThreshold) * time.Second,
		maxSwitchesPerMinute: 5, // Allow up to 5 switches per minute
		hysteresisThreshold:  opts.ConnectionReplacementPriorityThreshold,
	}
}

// ShouldSwitchConnection determines if we should switch to a new connection
// based on hysteresis principles
func (hc *HysteresisController) ShouldSwitchConnection(currentConn, newConn protocol.Connection) bool {
	hc.mut.Lock()
	defer hc.mut.Unlock()
	
	now := time.Now()
	
	// If we don't have a primary connection yet, accept the new one
	if hc.primaryConnection == nil {
		hc.primaryConnection = newConn
		hc.lastSwitchTime = now
		return true
	}
	
	// If this is the same connection, no switch needed
	if currentConn != nil && currentConn.ConnectionID() == newConn.ConnectionID() {
		return false
	}
	
	// Check if we're within the cooldown period
	if now.Sub(hc.lastSwitchTime) < hc.switchCooldownPeriod {
		// Within cooldown, only switch if the new connection is significantly better
		currentPriority := currentConn.Priority()
		newPriority := newConn.Priority()
		
		// Require a larger priority difference to overcome hysteresis during cooldown
		adjustedThreshold := hc.hysteresisThreshold * 2
		if newPriority <= currentPriority-adjustedThreshold {
			hc.recordSwitch(newConn, now)
			return true
		}
		
		return false
	}
	
	// Check switch rate limiting
	if hc.isSwitchRateExceeded(now) {
		// Rate limit exceeded, only allow much better connections
		currentPriority := currentConn.Priority()
		newPriority := newConn.Priority()
		
		// Require a much larger priority difference
		adjustedThreshold := hc.hysteresisThreshold * 3
		if newPriority <= currentPriority-adjustedThreshold {
			hc.recordSwitch(newConn, now)
			return true
		}
		
		return false
	}
	
	// Normal switching logic - check if new connection is better
	currentPriority := currentConn.Priority()
	newPriority := newConn.Priority()
	
	// Standard hysteresis threshold
	if newPriority <= currentPriority-hc.hysteresisThreshold {
		hc.recordSwitch(newConn, now)
		return true
	}
	
	return false
}

// recordSwitch records a connection switch for rate limiting
func (hc *HysteresisController) recordSwitch(newConn protocol.Connection, switchTime time.Time) {
	// Update state
	hc.primaryConnection = newConn
	hc.lastSwitchTime = switchTime
	hc.switchCount++
	
	// Reset switch count if we've moved to a new minute
	if switchTime.Sub(hc.lastSwitchTime).Minutes() >= 1 {
		hc.switchCount = 1
	}
}

// isSwitchRateExceeded checks if we're exceeding the switch rate limit
func (hc *HysteresisController) isSwitchRateExceeded(now time.Time) bool {
	// Reset count if we've moved to a new minute
	if now.Sub(hc.lastSwitchTime).Minutes() >= 1 {
		hc.switchCount = 0
		return false
	}
	
	return hc.switchCount >= hc.maxSwitchesPerMinute
}

// GetAdjustedPriorityThreshold returns an adjusted priority threshold based on
// current switching behavior to implement adaptive hysteresis
func (hc *HysteresisController) GetAdjustedPriorityThreshold(baseThreshold int) int {
	hc.mut.RLock()
	defer hc.mut.RUnlock()
	
	now := time.Now()
	
	// If we're in cooldown period, increase threshold
	if now.Sub(hc.lastSwitchTime) < hc.switchCooldownPeriod {
		return baseThreshold * 2
	}
	
	// If we've had many recent switches, increase threshold
	if hc.switchCount > hc.maxSwitchesPerMinute/2 {
		return baseThreshold * 3
	}
	
	// If the current connection has been stable for a while, reduce threshold
	if hc.primaryConnection != nil {
		establishedAt := hc.primaryConnection.EstablishedAt()
		if now.Sub(establishedAt) > time.Minute*5 {
			// Stable connection, allow easier switching
			return baseThreshold / 2
		}
	}
	
	return baseThreshold
}

// Reset resets the hysteresis controller state
func (hc *HysteresisController) Reset() {
	hc.mut.Lock()
	defer hc.mut.Unlock()
	
	hc.lastSwitchTime = time.Time{}
	hc.switchCount = 0
	hc.primaryConnection = nil
	hc.proposedConnection = nil
}

// GetCurrentPrimaryConnection returns the current primary connection
func (hc *HysteresisController) GetCurrentPrimaryConnection() protocol.Connection {
	hc.mut.RLock()
	defer hc.mut.RUnlock()
	return hc.primaryConnection
}

// GetSwitchMetrics returns metrics about connection switching behavior
type SwitchMetrics struct {
	LastSwitchTime      time.Time
	SwitchCount         int
	CooldownRemaining   time.Duration
	CurrentThreshold    int
	IsRateLimited       bool
	PrimaryConnectionID string
}

// GetSwitchMetrics returns metrics about the current switching state
func (hc *HysteresisController) GetSwitchMetrics() SwitchMetrics {
	hc.mut.RLock()
	defer hc.mut.RUnlock()
	
	now := time.Now()
	cooldownRemaining := hc.switchCooldownPeriod - now.Sub(hc.lastSwitchTime)
	if cooldownRemaining < 0 {
		cooldownRemaining = 0
	}
	
	var primaryConnID string
	if hc.primaryConnection != nil {
		primaryConnID = hc.primaryConnection.ConnectionID()
	}
	
	return SwitchMetrics{
		LastSwitchTime:      hc.lastSwitchTime,
		SwitchCount:         hc.switchCount,
		CooldownRemaining:   cooldownRemaining,
		CurrentThreshold:    hc.GetAdjustedPriorityThreshold(hc.hysteresisThreshold),
		IsRateLimited:       hc.isSwitchRateExceeded(now),
		PrimaryConnectionID: primaryConnID,
	}
}

// ProposeConnection proposes a new connection for switching consideration
// but doesn't immediately switch - allows for batch evaluation
func (hc *HysteresisController) ProposeConnection(conn protocol.Connection) {
	hc.mut.Lock()
	defer hc.mut.Unlock()
	hc.proposedConnection = conn
}

// GetProposedConnection returns the currently proposed connection
func (hc *HysteresisController) GetProposedConnection() protocol.Connection {
	hc.mut.RLock()
	defer hc.mut.RUnlock()
	return hc.proposedConnection
}

// ClearProposedConnection clears the proposed connection
func (hc *HysteresisController) ClearProposedConnection() {
	hc.mut.Lock()
	defer hc.mut.Unlock()
	hc.proposedConnection = nil
}