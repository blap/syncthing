// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// DeviceActivityEntry tracks activity metrics for a device
type DeviceActivityEntry struct {
	// Current activity count (number of outstanding requests)
	CurrentActivity int
	
	// CPU usage percentage for this device's operations
	CPUUsagePercent float64
	
	// Last update time for metrics
	LastUpdate time.Time
	
	// Moving average of activity levels
	ActivityAverage float64
}

// deviceActivity tracks the number of outstanding requests per device and can
// answer which device is least busy. It is safe for use from multiple
// goroutines.
type deviceActivity struct {
	act map[protocol.DeviceID]*DeviceActivityEntry
	mut sync.Mutex
}

func newDeviceActivity() *deviceActivity {
	return &deviceActivity{
		act: make(map[protocol.DeviceID]*DeviceActivityEntry),
	}
}

// Returns the index of the least busy device, or -1 if all are too busy.
func (m *deviceActivity) leastBusy(availability []Availability) int {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	low := 2<<30 - 1
	best := -1
	
	for i := range availability {
		deviceEntry, exists := m.act[availability[i].ID]
		if !exists {
			// No activity recorded yet, this device is least busy
			return i
		}
		
		// Calculate a weighted score based on activity and CPU usage
		score := m.calculateLoadScore(deviceEntry)
		
		if score < low {
			low = score
			best = i
		}
	}
	
	return best
}

// calculateLoadScore calculates a load score for a device based on activity and CPU usage
func (m *deviceActivity) calculateLoadScore(entry *DeviceActivityEntry) int {
	// Weight factors
	activityWeight := 0.7
	cpuWeight := 0.3
	
	// Normalize values (higher is worse)
	normalizedActivity := float64(entry.CurrentActivity) / 100.0 // Assume 100 is max reasonable activity
	normalizedCPU := entry.CPUUsagePercent / 100.0
	
	// Calculate weighted score (0-1000 scale)
	score := int((activityWeight*normalizedActivity + cpuWeight*normalizedCPU) * 1000)
	
	return score
}

// using indicates that a device is being used for an operation
func (m *deviceActivity) using(availability Availability) {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	entry, exists := m.act[availability.ID]
	if !exists {
		entry = &DeviceActivityEntry{
			CurrentActivity: 1,
			LastUpdate:      time.Now(),
			ActivityAverage: 1.0,
		}
		m.act[availability.ID] = entry
	} else {
		entry.CurrentActivity++
		entry.LastUpdate = time.Now()
		
		// Update moving average (simple exponential moving average)
		alpha := 0.3 // Smoothing factor
		entry.ActivityAverage = alpha*float64(entry.CurrentActivity) + (1-alpha)*entry.ActivityAverage
	}
}

// done indicates that a device has finished an operation
func (m *deviceActivity) done(availability Availability) {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	entry, exists := m.act[availability.ID]
	if !exists {
		return
	}
	
	if entry.CurrentActivity > 0 {
		entry.CurrentActivity--
		entry.LastUpdate = time.Now()
		
		// Update moving average
		alpha := 0.3 // Smoothing factor
		entry.ActivityAverage = alpha*float64(entry.CurrentActivity) + (1-alpha)*entry.ActivityAverage
	}
}

// updateCPUUsage updates the CPU usage percentage for a device
func (m *deviceActivity) updateCPUUsage(deviceID protocol.DeviceID, cpuPercent float64) {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	entry, exists := m.act[deviceID]
	if !exists {
		entry = &DeviceActivityEntry{
			LastUpdate:      time.Now(),
			CPUUsagePercent: cpuPercent,
		}
		m.act[deviceID] = entry
	} else {
		entry.CPUUsagePercent = cpuPercent
		entry.LastUpdate = time.Now()
	}
}

// getLoadScore returns the current load score for a device
func (m *deviceActivity) getLoadScore(deviceID protocol.DeviceID) int {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	entry, exists := m.act[deviceID]
	if !exists {
		return 0 // No activity, lowest load
	}
	
	return m.calculateLoadScore(entry)
}

// getLeastActiveDevices returns a list of device IDs sorted by activity level (least active first)
func (m *deviceActivity) getLeastActiveDevices(devices []protocol.DeviceID) []protocol.DeviceID {
	m.mut.Lock()
	defer m.mut.Unlock()
	
	// Create a slice of device-score pairs
	type deviceScore struct {
		deviceID protocol.DeviceID
		score    int
	}
	
	scores := make([]deviceScore, 0, len(devices))
	
	for _, deviceID := range devices {
		entry, exists := m.act[deviceID]
		score := 0
		if exists {
			score = m.calculateLoadScore(entry)
		}
		scores = append(scores, deviceScore{deviceID, score})
	}
	
	// Sort by score (ascending - lower scores are better)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score > scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	// Extract sorted device IDs
	result := make([]protocol.DeviceID, len(scores))
	for i, ds := range scores {
		result[i] = ds.deviceID
	}
	
	return result
}