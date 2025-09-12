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

// TransferState represents the state of an active file transfer
type TransferState struct {
	// File identification
	Folder string `json:"folder"`
	File   string `json:"file"`

	// Transfer progress
	TotalSize   int64 `json:"totalSize"`
	Transferred int64 `json:"transferred"`
	BlockSize   int   `json:"blockSize"`
	NextBlock   int   `json:"nextBlock"`

	// Transfer metadata
	StartTime    time.Time `json:"startTime"`
	LastActivity time.Time `json:"lastActivity"`
	TransferRate float64   `json:"transferRate"` // bytes per second

	// Request tracking
	PendingRequests map[int32]PendingRequest `json:"pendingRequests"`
}

// PendingRequest represents a pending block request
type PendingRequest struct {
	BlockIndex int       `json:"blockIndex"`
	Offset     int64     `json:"offset"`
	Size       int       `json:"size"`
	Hash       []byte    `json:"hash"`
	Timestamp  time.Time `json:"timestamp"`
}

// ConnectionMigrationManager manages the migration of transfers between connections
type ConnectionMigrationManager struct {
	mut sync.RWMutex

	// Active transfers by connection
	activeTransfers map[string]map[string]*TransferState // connectionID -> transferKey -> TransferState

	// Migration state
	migrationInProgress map[string]bool // connectionID -> inProgress
}

// NewConnectionMigrationManager creates a new connection migration manager
func NewConnectionMigrationManager() *ConnectionMigrationManager {
	return &ConnectionMigrationManager{
		activeTransfers:     make(map[string]map[string]*TransferState),
		migrationInProgress: make(map[string]bool),
	}
}

// RegisterTransfer registers a new transfer with the migration manager
func (cmm *ConnectionMigrationManager) RegisterTransfer(conn protocol.Connection, folder, file string, totalSize int64, blockSize int) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()

	// Initialize transfers map for this connection if needed
	if cmm.activeTransfers[connID] == nil {
		cmm.activeTransfers[connID] = make(map[string]*TransferState)
	}

	// Create transfer key
	transferKey := folder + "/" + file

	// Create new transfer state
	transferState := &TransferState{
		Folder:          folder,
		File:            file,
		TotalSize:       totalSize,
		BlockSize:       blockSize,
		NextBlock:       0,
		StartTime:       time.Now(),
		LastActivity:    time.Now(),
		PendingRequests: make(map[int32]PendingRequest),
	}

	cmm.activeTransfers[connID][transferKey] = transferState
}

// UpdateTransferProgress updates the progress of an active transfer
func (cmm *ConnectionMigrationManager) UpdateTransferProgress(conn protocol.Connection, folder, file string, transferred int64, nextBlock int) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()
	transferKey := folder + "/" + file

	// Find the transfer state
	if transfers, ok := cmm.activeTransfers[connID]; ok {
		if transferState, ok := transfers[transferKey]; ok {
			transferState.Transferred = transferred
			transferState.NextBlock = nextBlock
			transferState.LastActivity = time.Now()

			// Update transfer rate (bytes per second)
			duration := time.Since(transferState.StartTime).Seconds()
			if duration > 0 {
				transferState.TransferRate = float64(transferred) / duration
			}
		}
	}
}

// AddPendingRequest adds a pending request to a transfer
func (cmm *ConnectionMigrationManager) AddPendingRequest(conn protocol.Connection, folder, file string, requestID int32, blockIndex int, offset int64, size int, hash []byte) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()
	transferKey := folder + "/" + file

	// Find the transfer state
	if transfers, ok := cmm.activeTransfers[connID]; ok {
		if transferState, ok := transfers[transferKey]; ok {
			transferState.PendingRequests[requestID] = PendingRequest{
				BlockIndex: blockIndex,
				Offset:     offset,
				Size:       size,
				Hash:       hash,
				Timestamp:  time.Now(),
			}
		}
	}
}

// RemovePendingRequest removes a completed request from a transfer
func (cmm *ConnectionMigrationManager) RemovePendingRequest(conn protocol.Connection, folder, file string, requestID int32) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()
	transferKey := folder + "/" + file

	// Find the transfer state
	if transfers, ok := cmm.activeTransfers[connID]; ok {
		if transferState, ok := transfers[transferKey]; ok {
			delete(transferState.PendingRequests, requestID)
		}
	}
}

// GetTransferState retrieves the state of a transfer for migration
func (cmm *ConnectionMigrationManager) GetTransferState(conn protocol.Connection, folder, file string) (*TransferState, bool) {
	cmm.mut.RLock()
	defer cmm.mut.RUnlock()

	connID := conn.ConnectionID()
	transferKey := folder + "/" + file

	// Find the transfer state
	if transfers, ok := cmm.activeTransfers[connID]; ok {
		if transferState, ok := transfers[transferKey]; ok {
			// Return a copy of the transfer state
			stateCopy := *transferState
			return &stateCopy, true
		}
	}

	return nil, false
}

// GetAllTransferStates retrieves all transfer states for a connection
func (cmm *ConnectionMigrationManager) GetAllTransferStates(conn protocol.Connection) map[string]*TransferState {
	cmm.mut.RLock()
	defer cmm.mut.RUnlock()

	connID := conn.ConnectionID()

	// Return a copy of all transfer states
	result := make(map[string]*TransferState)
	if transfers, ok := cmm.activeTransfers[connID]; ok {
		for key, state := range transfers {
			stateCopy := *state
			result[key] = &stateCopy
		}
	}

	return result
}

// RemoveTransfer removes a transfer from tracking
func (cmm *ConnectionMigrationManager) RemoveTransfer(conn protocol.Connection, folder, file string) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()
	transferKey := folder + "/" + file

	// Remove the transfer state
	if transfers, ok := cmm.activeTransfers[connID]; ok {
		delete(transfers, transferKey)

		// Clean up empty maps
		if len(transfers) == 0 {
			delete(cmm.activeTransfers, connID)
		}
	}
}

// RemoveAllTransfersForConnection removes all transfers for a connection
func (cmm *ConnectionMigrationManager) RemoveAllTransfersForConnection(conn protocol.Connection) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()
	delete(cmm.activeTransfers, connID)
}

// StartMigration marks a connection as being in migration
func (cmm *ConnectionMigrationManager) StartMigration(conn protocol.Connection) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()
	cmm.migrationInProgress[connID] = true
}

// CompleteMigration marks a connection as having completed migration
func (cmm *ConnectionMigrationManager) CompleteMigration(conn protocol.Connection) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	connID := conn.ConnectionID()
	delete(cmm.migrationInProgress, connID)
}

// IsMigrationInProgress checks if migration is in progress for a connection
func (cmm *ConnectionMigrationManager) IsMigrationInProgress(conn protocol.Connection) bool {
	cmm.mut.RLock()
	defer cmm.mut.RUnlock()

	connID := conn.ConnectionID()
	return cmm.migrationInProgress[connID]
}

// MigrateTransfers migrates all transfers from one connection to another
func (cmm *ConnectionMigrationManager) MigrateTransfers(oldConn, newConn protocol.Connection) map[string]*TransferState {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	oldConnID := oldConn.ConnectionID()
	newConnID := newConn.ConnectionID()

	// Get transfers from old connection
	transfers, ok := cmm.activeTransfers[oldConnID]
	if !ok {
		return nil
	}

	// Move transfers to new connection
	cmm.activeTransfers[newConnID] = transfers
	delete(cmm.activeTransfers, oldConnID)

	// Return a copy of the migrated transfers
	result := make(map[string]*TransferState)
	for key, state := range transfers {
		stateCopy := *state
		result[key] = &stateCopy
	}

	return result
}

// MigrateSingleTransfer migrates a single transfer from one connection to another
func (cmm *ConnectionMigrationManager) MigrateSingleTransfer(oldConn, newConn protocol.Connection, folder, file string) (*TransferState, bool) {
	cmm.mut.Lock()
	defer cmm.mut.Unlock()

	oldConnID := oldConn.ConnectionID()
	newConnID := newConn.ConnectionID()
	transferKey := folder + "/" + file

	// Get the transfer from old connection
	oldTransfers, ok := cmm.activeTransfers[oldConnID]
	if !ok {
		return nil, false
	}

	transferState, ok := oldTransfers[transferKey]
	if !ok {
		return nil, false
	}

	// Initialize transfers map for new connection if needed
	if cmm.activeTransfers[newConnID] == nil {
		cmm.activeTransfers[newConnID] = make(map[string]*TransferState)
	}

	// Move transfer to new connection
	cmm.activeTransfers[newConnID][transferKey] = transferState
	delete(oldTransfers, transferKey)

	// Clean up empty maps
	if len(oldTransfers) == 0 {
		delete(cmm.activeTransfers, oldConnID)
	}

	// Update metrics
	metricConnectionMigrationCount.WithLabelValues(oldConn.DeviceID().String()).Inc()

	// Return a copy of the migrated transfer
	stateCopy := *transferState
	return &stateCopy, true
}

// GetBestConnectionForTransfer determines the best connection for a transfer based on connection quality
func (cmm *ConnectionMigrationManager) GetBestConnectionForTransfer(deviceID protocol.DeviceID, service interface {
	GetConnectionsForDevice(protocol.DeviceID) []protocol.Connection
}, folder, file string,
) protocol.Connection {
	// Get all connections for this device
	connections := service.GetConnectionsForDevice(deviceID)
	if len(connections) == 0 {
		return nil
	}

	// If only one connection, return it
	if len(connections) == 1 {
		return connections[0]
	}

	// Select the best connection based on health score and traffic metrics
	var bestConn protocol.Connection
	var bestScore float64

	for _, conn := range connections {
		// Get connection quality score
		var score float64

		// Try to get health score from the connection's health monitor
		if healthMonitoredConn, ok := conn.(interface{ HealthMonitor() *HealthMonitor }); ok {
			if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
				score = monitor.GetHealthScore()
			}
		}

		// Prefer connections with higher scores
		if bestConn == nil || score > bestScore {
			bestConn = conn
			bestScore = score
		}
	}

	return bestConn
}

// ShouldMigrateTransfer determines if a transfer should be migrated to a better connection
func (cmm *ConnectionMigrationManager) ShouldMigrateTransfer(conn protocol.Connection, service interface {
	GetConnectionsForDevice(protocol.DeviceID) []protocol.Connection
}, folder, file string,
) bool {
	// Get the current transfer state
	_, exists := cmm.GetTransferState(conn, folder, file)
	if !exists {
		return false
	}

	// Get the best connection for this transfer
	bestConn := cmm.GetBestConnectionForTransfer(conn.DeviceID(), service, folder, file)
	if bestConn == nil || bestConn.ConnectionID() == conn.ConnectionID() {
		return false
	}

	// Get scores for both connections
	currentScore := getConnectionQualityScore(conn)
	bestScore := GetConnectionQualityScore(bestConn)

	// Migrate if the best connection is significantly better
	// (at least 20% improvement)
	return bestScore > currentScore*1.2
}

// GetConnectionQualityScore calculates a quality score for a connection
func GetConnectionQualityScore(conn protocol.Connection) float64 {
	var score float64 = 50.0 // Default score

	// Try to get health score from the connection's health monitor
	if healthMonitoredConn, ok := conn.(interface{ HealthMonitor() *HealthMonitor }); ok {
		if monitor := healthMonitoredConn.HealthMonitor(); monitor != nil {
			score = monitor.GetHealthScore()
		}
	}

	return score
}
