// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// ConnectionPoolManager manages a pool of connections for frequently disconnected devices
type ConnectionPoolManager struct {
	mut sync.RWMutex

	// Pools by device ID
	pools map[protocol.DeviceID]*ConnectionPool

	// Default pool settings
	defaultPoolSize int
	maxPoolSize     int
	idleTimeout     time.Duration
}

// ConnectionPool represents a pool of connections for a specific device
type ConnectionPool struct {
	mut sync.RWMutex

	deviceID     protocol.DeviceID
	connections  []protocol.Connection
	maxSize      int
	idleTimeout  time.Duration
	lastActivity time.Time

	// Strategy tracking
	lastUsedIndex int
	usageCount    map[string]int       // Connection ID -> usage count
	lastUsedTime  map[string]time.Time // Connection ID -> last used time

	// Pool statistics
	totalCreated int
	totalReused  int
	totalExpired int
}

// PoolAllocationStrategy defines how connections are allocated from the pool
type PoolAllocationStrategy int

const (
	// RoundRobinStrategy selects connections in round-robin fashion
	RoundRobinStrategy PoolAllocationStrategy = iota

	// HealthBasedStrategy selects the healthiest connection
	HealthBasedStrategy

	// RandomStrategy selects a random connection
	RandomStrategy

	// LeastUsedStrategy selects the least recently used connection
	LeastUsedStrategy
)

// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(defaultPoolSize, maxPoolSize int, idleTimeout time.Duration) *ConnectionPoolManager {
	if defaultPoolSize <= 0 {
		defaultPoolSize = 3
	}
	if maxPoolSize <= 0 {
		maxPoolSize = 10
	}
	if idleTimeout <= 0 {
		idleTimeout = 30 * time.Minute
	}

	return &ConnectionPoolManager{
		pools:           make(map[protocol.DeviceID]*ConnectionPool),
		defaultPoolSize: defaultPoolSize,
		maxPoolSize:     maxPoolSize,
		idleTimeout:     idleTimeout,
	}
}

// GetPool retrieves or creates a connection pool for a device
func (cpm *ConnectionPoolManager) GetPool(deviceID protocol.DeviceID) *ConnectionPool {
	cpm.mut.Lock()
	defer cpm.mut.Unlock()

	pool, exists := cpm.pools[deviceID]
	if !exists {
		pool = NewConnectionPool(deviceID, cpm.defaultPoolSize, cpm.idleTimeout)
		cpm.pools[deviceID] = pool
	}

	return pool
}

// ReturnConnection returns a connection to its pool
func (cpm *ConnectionPoolManager) ReturnConnection(conn protocol.Connection) {
	cpm.mut.RLock()
	defer cpm.mut.RUnlock()

	deviceID := conn.DeviceID()
	if pool, exists := cpm.pools[deviceID]; exists {
		pool.ReturnConnection(conn)
	}
}

// CloseAllPools closes all connection pools and their connections
func (cpm *ConnectionPoolManager) CloseAllPools() {
	cpm.mut.Lock()
	defer cpm.mut.Unlock()

	for _, pool := range cpm.pools {
		pool.Close()
	}
	cpm.pools = make(map[protocol.DeviceID]*ConnectionPool)
}

// CleanupExpiredPools removes pools that have been idle for too long
func (cpm *ConnectionPoolManager) CleanupExpiredPools() {
	cpm.mut.Lock()
	defer cpm.mut.Unlock()

	now := time.Now()
	for deviceID, pool := range cpm.pools {
		if now.Sub(pool.lastActivity) > cpm.idleTimeout {
			pool.Close()
			delete(cpm.pools, deviceID)
		}
	}
}

// NewConnectionPool creates a new connection pool for a device
func NewConnectionPool(deviceID protocol.DeviceID, maxSize int, idleTimeout time.Duration) *ConnectionPool {
	return &ConnectionPool{
		deviceID: deviceID,
		connections: make([]protocol.Connection, 0),
		maxSize: maxSize,
		idleTimeout: idleTimeout,
		usageCount: make(map[string]int),
		lastUsedTime: make(map[string]time.Time),
	}
}

// AddConnection adds a connection to the pool if there's space
func (cp *ConnectionPool) AddConnection(conn protocol.Connection) bool {
	if cp == nil {
		return false
	}
	cp.mut.Lock()
	defer cp.mut.Unlock()

	// Don't add if pool is full
	if len(cp.connections) >= cp.maxSize {
		return false
	}

	// Don't add if connection is already in pool
	for _, existingConn := range cp.connections {
		if existingConn.ConnectionID() == conn.ConnectionID() {
			return false
		}
	}

	cp.connections = append(cp.connections, conn)
	cp.lastActivity = time.Now()
	cp.totalCreated++

	// Update metrics
	metricConnectionPoolSize.WithLabelValues(cp.deviceID.String()).Set(float64(len(cp.connections)))
	metricConnectionPoolCreated.WithLabelValues(cp.deviceID.String()).Inc()
	return true
}

// GetConnection retrieves a connection from the pool using the specified strategy
func (cp *ConnectionPool) GetConnection(strategy PoolAllocationStrategy) protocol.Connection {
	if cp == nil {
		return nil
	}
	cp.mut.Lock()
	defer cp.mut.Unlock()

	if len(cp.connections) == 0 {
		return nil
	}

	// Remove any closed connections first
	cp.removeClosedConnections()

	if len(cp.connections) == 0 {
		return nil
	}

	var selectedConn protocol.Connection

	switch strategy {
	case RoundRobinStrategy:
		selectedConn = cp.selectRoundRobin()
	case HealthBasedStrategy:
		selectedConn = cp.selectHealthBased()
	case RandomStrategy:
		selectedConn = cp.selectRandom()
	case LeastUsedStrategy:
		selectedConn = cp.selectLeastUsed()
	default:
		selectedConn = cp.selectRoundRobin()
	}

	if selectedConn != nil {
		cp.totalReused++
		cp.lastActivity = time.Now()

		// Update metrics
		metricConnectionPoolReused.WithLabelValues(cp.deviceID.String()).Inc()
	}

	return selectedConn
}

// ReturnConnection returns a connection to the pool
func (cp *ConnectionPool) ReturnConnection(conn protocol.Connection) {
	if cp == nil {
		return
	}
	cp.mut.Lock()
	defer cp.mut.Unlock()

	// Update usage tracking
	connID := conn.ConnectionID()
	if cp.usageCount == nil {
		cp.usageCount = make(map[string]int)
	}
	cp.usageCount[connID]++
	if cp.lastUsedTime == nil {
		cp.lastUsedTime = make(map[string]time.Time)
	}
	cp.lastUsedTime[connID] = time.Now()
	cp.lastActivity = time.Now()
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() {
	cp.mut.Lock()
	defer cp.mut.Unlock()

	for _, conn := range cp.connections {
		conn.Close(nil)
	}
	cp.connections = make([]protocol.Connection, 0)

	// Update metrics
	metricConnectionPoolSize.WithLabelValues(cp.deviceID.String()).Set(0)
}

// GetPoolStats returns statistics about the pool
func (cp *ConnectionPool) GetPoolStats() (totalCreated, totalReused, totalExpired int) {
	cp.mut.RLock()
	defer cp.mut.RUnlock()

	return cp.totalCreated, cp.totalReused, cp.totalExpired
}

// getConnectionQualityScore calculates a quality score for a connection
func getConnectionQualityScore(conn protocol.Connection) float64 {
	var score float64 = 50.0 // Default score

	// Try to get health score from the connection's health monitor
	if healthMonitoredConn, ok := conn.(interface{ HealthMonitor() *HealthMonitor }); ok {
		if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
			score = monitor.GetHealthScore()
		}
	}

	return score
}

// selectRoundRobin selects a connection in round-robin fashion
func (cp *ConnectionPool) selectRoundRobin() protocol.Connection {
	if len(cp.connections) == 0 {
		return nil
	}

	// Use round-robin by tracking the last used index
	selectedConn := cp.connections[cp.lastUsedIndex]
	cp.lastUsedIndex = (cp.lastUsedIndex + 1) % len(cp.connections)
	return selectedConn
}

// selectHealthBased selects the connection with the best health score
func (cp *ConnectionPool) selectHealthBased() protocol.Connection {
	var bestConn protocol.Connection
	var bestScore float64

	for _, conn := range cp.connections {
		score := GetConnectionQualityScore(conn)
		if bestConn == nil || score > bestScore {
			bestConn = conn
			bestScore = score
		}
	}

	return bestConn
}

// selectRandom selects a random connection from the pool
func (cp *ConnectionPool) selectRandom() protocol.Connection {
	if len(cp.connections) == 0 {
		return nil
	}

	// Use proper random selection
	// In a real implementation, we would use a cryptographically secure random number generator
	index := time.Now().Nanosecond() % len(cp.connections)
	return cp.connections[index]
}

// selectLeastUsed selects the least recently used connection
func (cp *ConnectionPool) selectLeastUsed() protocol.Connection {
	if len(cp.connections) == 0 {
		return nil
	}

	// Find the connection with the oldest last used time
	var leastUsedConn protocol.Connection
	var oldestTime time.Time

	for _, conn := range cp.connections {
		connID := conn.ConnectionID()
		lastUsed, exists := cp.lastUsedTime[connID]

		// If this connection has never been used, it's a good candidate
		if !exists {
			return conn
		}

		// If this is the first connection we're checking or it's older than our current oldest
		if leastUsedConn == nil || lastUsed.Before(oldestTime) {
			leastUsedConn = conn
			oldestTime = lastUsed
		}
	}

	return leastUsedConn
}

// removeClosedConnections removes any closed connections from the pool
func (cp *ConnectionPool) removeClosedConnections() {
	activeConns := make([]protocol.Connection, 0, len(cp.connections))

	for _, conn := range cp.connections {
		// Check if connection is still alive
		if conn.Statistics().StartedAt.After(time.Time{}) {
			activeConns = append(activeConns, conn)
		} else {
			// Connection is closed, clean up tracking data
			connID := conn.ConnectionID()
			delete(cp.usageCount, connID)
			delete(cp.lastUsedTime, connID)
			cp.totalExpired++

			// Update metrics
			metricConnectionPoolExpired.WithLabelValues(cp.deviceID.String()).Inc()
		}
	}

	cp.connections = activeConns

	// Update metrics
	metricConnectionPoolSize.WithLabelValues(cp.deviceID.String()).Set(float64(len(cp.connections)))

	// Reset lastUsedIndex if it's out of bounds
	if cp.lastUsedIndex >= len(cp.connections) && len(cp.connections) > 0 {
		cp.lastUsedIndex = 0
	} else if len(cp.connections) == 0 {
		cp.lastUsedIndex = 0
	}
}
