// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

// ContinuousMonitoringFramework provides automated monitoring of connection health and performance
type ContinuousMonitoringFramework struct {
	cfg       config.Wrapper
	evLogger  events.Logger
	service   Service
	interval  time.Duration
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mut       sync.RWMutex
	isRunning bool
}

// NewContinuousMonitoringFramework creates a new monitoring framework
func NewContinuousMonitoringFramework(cfg config.Wrapper, evLogger events.Logger, service Service) *ContinuousMonitoringFramework {
	return &ContinuousMonitoringFramework{
		cfg:      cfg,
		evLogger: evLogger,
		service:  service,
		interval: 30 * time.Second, // Default monitoring interval
		stopChan: make(chan struct{}),
	}
}

// Start begins the continuous monitoring process
func (cmf *ContinuousMonitoringFramework) Start() {
	cmf.mut.Lock()
	if cmf.isRunning {
		cmf.mut.Unlock()
		return
	}
	cmf.isRunning = true
	cmf.mut.Unlock()

	cmf.wg.Add(1)
	go cmf.monitoringLoop()
}

// Stop halts the continuous monitoring process
func (cmf *ContinuousMonitoringFramework) Stop() {
	cmf.mut.Lock()
	if !cmf.isRunning {
		cmf.mut.Unlock()
		return
	}
	cmf.isRunning = false
	
	// Close the stop channel only if it hasn't been closed already
	select {
	case <-cmf.stopChan:
		// Channel already closed
	default:
		close(cmf.stopChan)
	}
	cmf.mut.Unlock()

	cmf.wg.Wait()
}

// SetInterval updates the monitoring interval
func (cmf *ContinuousMonitoringFramework) SetInterval(interval time.Duration) {
	cmf.mut.Lock()
	cmf.interval = interval
	cmf.mut.Unlock()
}

// IsRunning returns whether the monitoring framework is currently active
func (cmf *ContinuousMonitoringFramework) IsRunning() bool {
	cmf.mut.RLock()
	defer cmf.mut.RUnlock()
	return cmf.isRunning
}

// monitoringLoop is the main monitoring loop that periodically checks connection health
func (cmf *ContinuousMonitoringFramework) monitoringLoop() {
	defer cmf.wg.Done()

	ticker := time.NewTicker(cmf.interval)
	defer ticker.Stop()

	for {
		select {
		case <-cmf.stopChan:
			return
		case <-ticker.C:
			cmf.performMonitoringCycle()
		}
	}
}

// performMonitoringCycle executes a single monitoring cycle
func (cmf *ContinuousMonitoringFramework) performMonitoringCycle() {
	
	// Get current connection status
	connectionStatus := cmf.service.ConnectionStatus()
	
	// Get connected devices
	connectedDevices := cmf.service.GetConnectedDevices()
	
	// Log monitoring cycle
	cmf.evLogger.Log(events.DeviceConnected, map[string]interface{}{
		"connectedDevices": len(connectedDevices),
		"timestamp":        time.Now(),
	})
	
	// Check each connected device
	for _, deviceID := range connectedDevices {
		cmf.checkDeviceHealth(context.Background(), deviceID)
	}
	
	// Check overall connection health
	cmf.checkOverallHealth(context.Background(), connectionStatus, connectedDevices)
	
	// Trigger any necessary actions based on health status
	cmf.handleHealthIssues(context.Background(), connectionStatus)
}

// checkDeviceHealth evaluates the health of a specific device connection
func (cmf *ContinuousMonitoringFramework) checkDeviceHealth(_ context.Context, deviceID protocol.DeviceID) {
	// Get connections for this device
	connections := cmf.service.GetConnectionsForDevice(deviceID)
	
	if len(connections) == 0 {
		// No active connections to this device
		return
	}
	
	// Check each connection to this device
	for _, conn := range connections {
		// Get connection statistics
		statistics := conn.Statistics()
		
		// Log connection statistics
		cmf.evLogger.Log(events.RemoteIndexUpdated, map[string]interface{}{
			"device":      deviceID.String(),
			"type":        conn.Type(),
			"statistics":  statistics,
			"timestamp":   time.Now(),
		})
		
		// Check for any concerning patterns
		cmf.analyzeConnectionPatterns(context.Background(), deviceID, conn, statistics)
	}
}

// checkOverallHealth evaluates the overall connection health
func (cmf *ContinuousMonitoringFramework) checkOverallHealth(_ context.Context, connectionStatus map[string]ConnectionStatusEntry, connectedDevices []protocol.DeviceID) {
	// Count total connections
	totalConnections := len(connectionStatus)
	
	// Count devices with errors
	errorDevices := 0
	for _, status := range connectionStatus {
		if status.Error != nil {
			errorDevices++
		}
	}
	
	// Log overall health metrics
	cmf.evLogger.Log(events.StateChanged, map[string]interface{}{
		"totalConnections": totalConnections,
		"connectedDevices": len(connectedDevices),
		"errorDevices":     errorDevices,
		"healthScore":      cmf.calculateOverallHealthScore(totalConnections, len(connectedDevices), errorDevices),
		"timestamp":        time.Now(),
	})
}

// calculateOverallHealthScore computes a health score for the entire connection system
func (cmf *ContinuousMonitoringFramework) calculateOverallHealthScore(totalConnections, connectedDevices, errorDevices int) float64 {
	if totalConnections == 0 {
		return 100.0 // Perfect score if no connections are expected
	}
	
	// Calculate health based on ratio of successful connections to total connections
	successRate := float64(connectedDevices) / float64(totalConnections)
	errorRate := float64(errorDevices) / float64(totalConnections)
	
	// Health score is primarily based on success rate, with penalty for errors
	healthScore := successRate*80.0 + (1.0-errorRate)*20.0
	
	// Ensure score is between 0 and 100
	if healthScore < 0.0 {
		healthScore = 0.0
	}
	if healthScore > 100.0 {
		healthScore = 100.0
	}
	
	return healthScore
}

// analyzeConnectionPatterns looks for concerning patterns in connection statistics
func (cmf *ContinuousMonitoringFramework) analyzeConnectionPatterns(_ context.Context, deviceID protocol.DeviceID, conn protocol.Connection, statistics protocol.Statistics) {
	// Calculate throughput (bytes per second)
	connectionDuration := time.Since(statistics.StartedAt)
	if connectionDuration > 0 {
		inBytesPerSecond := float64(statistics.InBytesTotal) / connectionDuration.Seconds()
		outBytesPerSecond := float64(statistics.OutBytesTotal) / connectionDuration.Seconds()
		
		// Check for low throughput
		if inBytesPerSecond < 1024 && outBytesPerSecond < 1024 {
			// Only log if connection has been active for a while
			if connectionDuration > 5*time.Minute {
				cmf.evLogger.Log(events.Failure, map[string]interface{}{
					"device":          deviceID.String(),
					"message":         fmt.Sprintf("Low throughput: in=%.2f B/s, out=%.2f B/s", inBytesPerSecond, outBytesPerSecond),
					"type":            conn.Type(),
					"connectionAge":   connectionDuration.String(),
					"timestamp":       time.Now(),
				})
			}
		}
	}
}

// handleHealthIssues takes action based on detected health issues
func (cmf *ContinuousMonitoringFramework) handleHealthIssues(_ context.Context, connectionStatus map[string]ConnectionStatusEntry) {
	// Count connections with errors
	errorCount := 0
	for _, status := range connectionStatus {
		if status.Error != nil {
			errorCount++
		}
	}
	
	// If more than 30% of connections have errors, trigger DialNow to attempt reconnections
	if len(connectionStatus) > 0 && float64(errorCount)/float64(len(connectionStatus)) > 0.3 {
		// Log the decision to trigger reconnections
		cmf.evLogger.Log(events.StateChanged, map[string]interface{}{
			"message":          "Reconnection triggered due to high error ratio",
			"errorConnections": errorCount,
			"totalConnections": len(connectionStatus),
			"errorRatio":       float64(errorCount) / float64(len(connectionStatus)),
			"timestamp":        time.Now(),
		})
		
		// Trigger immediate dialing to attempt to reestablish connections
		cmf.service.DialNow()
	}
}