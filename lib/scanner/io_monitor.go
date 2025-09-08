// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package scanner

import (
	"sync"
	"time"
)

// ioMonitor tracks I/O activity to implement I/O-aware scheduling
type ioMonitor struct {
	mu           sync.RWMutex
	readBytes    int64
	writeBytes   int64
	lastUpdate   time.Time
	readRate     float64  // bytes per second
	writeRate    float64  // bytes per second
}

// newIOMonitor creates a new I/O monitor
func newIOMonitor() *ioMonitor {
	return &ioMonitor{
		lastUpdate: time.Now(),
	}
}

// recordRead records bytes read from disk
func (m *ioMonitor) recordRead(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	m.readBytes += bytes
	
	// Update rate calculation
	if now.After(m.lastUpdate) {
		elapsed := now.Sub(m.lastUpdate).Seconds()
		if elapsed > 0 {
			m.readRate = float64(bytes) / elapsed
		}
	}
	
	m.lastUpdate = now
}

// recordWrite records bytes written to disk
func (m *ioMonitor) recordWrite(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	m.writeBytes += bytes
	
	// Update rate calculation
	if now.After(m.lastUpdate) {
		elapsed := now.Sub(m.lastUpdate).Seconds()
		if elapsed > 0 {
			m.writeRate = float64(bytes) / elapsed
		}
	}
	
	m.lastUpdate = now
}

// getReadRate returns the current read rate in bytes per second
func (m *ioMonitor) getReadRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.readRate
}

// getWriteRate returns the current write rate in bytes per second
func (m *ioMonitor) getWriteRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.writeRate
}

// getTotalRate returns the total I/O rate in bytes per second
func (m *ioMonitor) getTotalRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.readRate + m.writeRate
}

// shouldThrottle checks if we should throttle based on I/O rates
func (m *ioMonitor) shouldThrottle(maxReadRate, maxWriteRate float64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Throttle if either read or write rate exceeds limits
	return m.readRate > maxReadRate || m.writeRate > maxWriteRate
}

// reset resets the I/O counters
func (m *ioMonitor) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.readBytes = 0
	m.writeBytes = 0
	m.readRate = 0
	m.writeRate = 0
	m.lastUpdate = time.Now()
}