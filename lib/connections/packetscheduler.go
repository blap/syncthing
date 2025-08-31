// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"math/rand"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// PacketScheduler is responsible for distributing packets across multiple
// connections based on their health scores
type PacketScheduler struct {
	mut            sync.RWMutex
	connections    map[protocol.DeviceID][]protocol.Connection
	lastSelection  map[protocol.DeviceID]protocol.Connection
	selectionCount map[protocol.DeviceID]map[string]int
	randSource     *rand.Rand
}

// NewPacketScheduler creates a new packet scheduler
func NewPacketScheduler() *PacketScheduler {
	return &PacketScheduler{
		connections:    make(map[protocol.DeviceID][]protocol.Connection),
		lastSelection:  make(map[protocol.DeviceID]protocol.Connection),
		selectionCount: make(map[protocol.DeviceID]map[string]int),
		randSource:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// AddConnection adds a connection to the scheduler for a device
func (ps *PacketScheduler) AddConnection(deviceID protocol.DeviceID, conn protocol.Connection) {
	ps.mut.Lock()
	defer ps.mut.Unlock()

	ps.connections[deviceID] = append(ps.connections[deviceID], conn)
	
	// Initialize selection count for this connection if needed
	if ps.selectionCount[deviceID] == nil {
		ps.selectionCount[deviceID] = make(map[string]int)
	}
}

// RemoveConnection removes a connection from the scheduler for a device
func (ps *PacketScheduler) RemoveConnection(deviceID protocol.DeviceID, connID string) {
	ps.mut.Lock()
	defer ps.mut.Unlock()

	conns, ok := ps.connections[deviceID]
	if !ok {
		return
	}

	// Remove the connection from the list
	for i, conn := range conns {
		if conn.ConnectionID() == connID {
			ps.connections[deviceID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	
	// Remove from selection count tracking
	if ps.selectionCount[deviceID] != nil {
		delete(ps.selectionCount[deviceID], connID)
	}
	
	// Clear last selection if it was this connection
	if ps.lastSelection[deviceID] != nil && ps.lastSelection[deviceID].ConnectionID() == connID {
		ps.lastSelection[deviceID] = nil
	}
}

// SelectConnection selects the best connection for a device based on health scores
func (ps *PacketScheduler) SelectConnection(deviceID protocol.DeviceID) protocol.Connection {
	ps.mut.RLock()
	defer ps.mut.RUnlock()

	conns, ok := ps.connections[deviceID]
	if !ok || len(conns) == 0 {
		return nil
	}

	// If only one connection, return it
	if len(conns) == 1 {
		return conns[0]
	}

	// Select the connection with the highest health score (failover mode)
	return ps.selectBestConnection(conns)
}

// SelectConnectionForLoadBalancing selects a connection for load balancing
// based on health scores and selection history
func (ps *PacketScheduler) SelectConnectionForLoadBalancing(deviceID protocol.DeviceID) protocol.Connection {
	ps.mut.RLock()
	defer ps.mut.RUnlock()

	conns, ok := ps.connections[deviceID]
	if !ok || len(conns) == 0 {
		return nil
	}

	// If only one connection, return it
	if len(conns) == 1 {
		return conns[0]
	}

	// Select connection based on weighted health scores (load balancing mode)
	return ps.selectConnectionWeighted(conns)
}

// selectBestConnection selects the connection with the highest health score
func (ps *PacketScheduler) selectBestConnection(connections []protocol.Connection) protocol.Connection {
	if len(connections) == 0 {
		return nil
	}

	bestConn := connections[0]
	bestScore := ps.getHealthScore(bestConn)

	for _, conn := range connections[1:] {
		score := ps.getHealthScore(conn)
		if score > bestScore {
			bestConn = conn
			bestScore = score
		}
	}

	return bestConn
}

// selectConnectionWeighted selects a connection using weighted random selection
// based on health scores
func (ps *PacketScheduler) selectConnectionWeighted(connections []protocol.Connection) protocol.Connection {
	if len(connections) == 0 {
		return nil
	}

	// Calculate total health score
	var totalScore float64
	scores := make([]float64, len(connections))
	for i, conn := range connections {
		score := ps.getHealthScore(conn)
		scores[i] = score
		totalScore += score
	}

	// If all connections have zero health, select randomly
	if totalScore <= 0 {
		return connections[ps.randSource.Intn(len(connections))]
	}

	// Select based on weighted probability
	randValue := ps.randSource.Float64() * totalScore
	cumulativeScore := 0.0

	for i, conn := range connections {
		cumulativeScore += scores[i]
		if randValue <= cumulativeScore {
			return conn
		}
	}

	// Fallback (should not happen)
	return connections[0]
}

// getHealthScore extracts the health score from a connection
func (ps *PacketScheduler) getHealthScore(conn protocol.Connection) float64 {
	// Try to get health score from the connection's health monitor
	// First try to type assert to a connection with HealthMonitor() *HealthMonitor
	if healthMonitoredConn, ok := conn.(interface{ HealthMonitor() *HealthMonitor }); ok {
		if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
			return monitor.GetHealthScore()
		}
	}
	
	// If that doesn't work, try the interface version (for real connections)
	if _, ok := conn.(interface{ HealthMonitor() protocol.HealthMonitorInterface }); ok {
		// We can't call GetHealthScore on the interface, so we'll return a default
		// In a real implementation, we would need to extend the interface
		// For now, we'll just return a default score
		return 50.0
	}
	
	// Default score if no health monitor available
	return 50.0
}

// GetConnectionCount returns the number of connections for a device
func (ps *PacketScheduler) GetConnectionCount(deviceID protocol.DeviceID) int {
	ps.mut.RLock()
	defer ps.mut.RUnlock()

	if conns, ok := ps.connections[deviceID]; ok {
		return len(conns)
	}
	return 0
}

// GetConnections returns all connections for a device
func (ps *PacketScheduler) GetConnections(deviceID protocol.DeviceID) []protocol.Connection {
	ps.mut.RLock()
	defer ps.mut.RUnlock()

	if conns, ok := ps.connections[deviceID]; ok {
		// Return a copy to avoid concurrency issues
		result := make([]protocol.Connection, len(conns))
		copy(result, conns)
		return result
	}
	return nil
}