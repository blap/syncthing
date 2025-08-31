// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// EnhancedMockConnection is an enhanced mock connection that includes health monitoring capabilities
type EnhancedMockConnection struct {
	id           string
	deviceID     protocol.DeviceID
	priority     int
	latency      time.Duration
	closed       bool
	closeError   error
	established  time.Time
	healthScore  float64
	healthMonitor *HealthMonitor
}

// NewEnhancedMockConnection creates a new enhanced mock connection
func NewEnhancedMockConnection(id string, deviceID protocol.DeviceID, priority int, healthScore float64) *EnhancedMockConnection {
	// Create a real HealthMonitor for testing
	cfg := config.New(protocol.EmptyDeviceID)
	healthMonitor := NewHealthMonitor(config.Wrap("/tmp/test-config.xml", cfg, protocol.EmptyDeviceID, nil), deviceID.String())
	healthMonitor.SetHealthScore(healthScore)
	
	return &EnhancedMockConnection{
		id:            id,
		deviceID:      deviceID,
		priority:      priority,
		latency:       10 * time.Millisecond, // Default latency
		established:   time.Now(),
		healthScore:   healthScore,
		healthMonitor: healthMonitor,
	}
}

// ID returns the connection ID
func (m *EnhancedMockConnection) ID() string {
	return m.id
}

// DeviceID returns the device ID
func (m *EnhancedMockConnection) DeviceID() protocol.DeviceID {
	return m.deviceID
}

// Priority returns the connection priority
func (m *EnhancedMockConnection) Priority() int {
	return m.priority
}

// Latency returns the connection latency
func (m *EnhancedMockConnection) Latency() time.Duration {
	return m.latency
}

// SetLatency sets the connection latency
func (m *EnhancedMockConnection) SetLatency(latency time.Duration) {
	m.latency = latency
}

// Close closes the connection
func (m *EnhancedMockConnection) Close(err error) {
	m.closed = true
	m.closeError = err
}

// Closed returns a channel that is closed when the connection is closed
func (m *EnhancedMockConnection) Closed() <-chan struct{} {
	ch := make(chan struct{})
	if m.closed {
		close(ch)
	}
	return ch
}

// HealthMonitor returns the health monitor for this connection
func (m *EnhancedMockConnection) HealthMonitor() *HealthMonitor {
	return m.healthMonitor
}

// SetHealthScore sets the health score for this connection
func (m *EnhancedMockConnection) SetHealthScore(score float64) {
	m.healthScore = score
	if m.healthMonitor != nil {
		m.healthMonitor.SetHealthScore(score)
	}
}

// ConnectionID returns the connection ID
func (m *EnhancedMockConnection) ConnectionID() string {
	return m.id
}

// Add all the required methods to satisfy the protocol.Connection interface
func (m *EnhancedMockConnection) Index(ctx context.Context, idx *protocol.Index) error { return nil }
func (m *EnhancedMockConnection) IndexUpdate(ctx context.Context, idxUp *protocol.IndexUpdate) error { return nil }
func (m *EnhancedMockConnection) Request(ctx context.Context, req *protocol.Request) ([]byte, error) { return nil, nil }
func (m *EnhancedMockConnection) ClusterConfig(config *protocol.ClusterConfig, passwords map[string]string) {}
func (m *EnhancedMockConnection) DownloadProgress(ctx context.Context, dp *protocol.DownloadProgress) {}
func (m *EnhancedMockConnection) Start() {}
func (m *EnhancedMockConnection) Statistics() protocol.Statistics { return protocol.Statistics{} }
func (m *EnhancedMockConnection) ConnectionInfo() protocol.ConnectionInfo { return m }
func (m *EnhancedMockConnection) Type() string { return "mock" }
func (m *EnhancedMockConnection) Transport() string { return "mock" }
func (m *EnhancedMockConnection) IsLocal() bool { return false }
func (m *EnhancedMockConnection) RemoteAddr() net.Addr { return nil }
func (m *EnhancedMockConnection) String() string { return fmt.Sprintf("enhanced-mock-connection-%s", m.id) }
func (m *EnhancedMockConnection) Crypto() string { return "mock" }
func (m *EnhancedMockConnection) EstablishedAt() time.Time { return m.established }