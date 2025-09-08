// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
)

// TestAvailability represents a device availability entry for testing
// Using a different name to avoid conflict with the real Availability struct
type TestAvailability struct {
	ID        protocol.DeviceID
}

// convertTestAvailability converts a slice of TestAvailability to Availability
func convertTestAvailability(testAvail []TestAvailability) []Availability {
	avail := make([]Availability, len(testAvail))
	for i, ta := range testAvail {
		avail[i] = Availability{ID: ta.ID}
	}
	return avail
}

func TestDeviceActivity_LeastBusy(t *testing.T) {
	// Create a new device activity tracker
	da := newDeviceActivity()

	// Create test devices
	device1 := protocol.DeviceID([32]byte{1, 2, 3, 4})
	device2 := protocol.DeviceID([32]byte{5, 6, 7, 8})
	device3 := protocol.DeviceID([32]byte{9, 10, 11, 12})

	// Create availability list
	availability := []TestAvailability{
		{ID: device1},
		{ID: device2},
		{ID: device3},
	}

	// Initially, all devices should have the same load (0), so the first should be selected
	// Fix type mismatch by using convertTestAvailability
	if lb := da.leastBusy(convertTestAvailability(availability)); lb != 0 {
		t.Errorf("Least busy device should be index 0, not %d", lb)
	}

	// Mark device1 as being used
	da.using(Availability{ID: availability[0].ID})

	// Now device2 should be the least busy
	if lb := da.leastBusy(convertTestAvailability(availability)); lb != 1 {
		t.Errorf("Least busy device should be index 1, not %d", lb)
	}

	// Mark device2 as being used as well
	da.using(Availability{ID: availability[1].ID})

	// Now device3 should be the least busy
	if lb := da.leastBusy(convertTestAvailability(availability)); lb != 2 {
		t.Errorf("Least busy device should be index 2, not %d", lb)
	}

	// Mark device3 as being used
	da.using(Availability{ID: availability[2].ID})

	// Now all devices have the same activity (1), so the first should be selected
	if lb := da.leastBusy(convertTestAvailability(availability)); lb != 0 {
		t.Errorf("Least busy device should be index 0, not %d", lb)
	}

	// Mark device1 as done
	da.done(Availability{ID: availability[0].ID})

	// Now device1 should be the least busy (0 activity)
	if lb := da.leastBusy(convertTestAvailability(availability)); lb != 0 {
		t.Errorf("Least busy device should be index 0, not %d", lb)
	}
}

func TestDeviceActivity_CalculateLoadScore(t *testing.T) {
	// Create a new device activity tracker
	da := newDeviceActivity()

	// Create a device entry with specific values
	entry := &DeviceActivityEntry{
		CurrentActivity: 5,
		CPUUsagePercent: 50.0,
		LastUpdate:      time.Now(),
		ActivityAverage: 4.5,
	}

	// Calculate load score
	score := da.calculateLoadScore(entry)

	// Score should be a positive integer based on the weighted formula
	if score < 0 {
		t.Errorf("Load score should be positive, got %d", score)
	}

	// Test with different values
	entry2 := &DeviceActivityEntry{
		CurrentActivity: 10,
		CPUUsagePercent: 80.0,
		LastUpdate:      time.Now(),
		ActivityAverage: 9.0,
	}

	score2 := da.calculateLoadScore(entry2)

	// Higher activity and CPU usage should result in higher score
	if score2 <= score {
		t.Errorf("Higher activity/CPU should result in higher score. Got %d and %d", score2, score)
	}
}

func TestDeviceActivity_UpdateCPUUsage(t *testing.T) {
	// Create a new device activity tracker
	da := newDeviceActivity()

	// Create a test device
	deviceID := protocol.DeviceID([32]byte{1, 2, 3, 4})

	// Update CPU usage for a device that doesn't exist yet
	da.updateCPUUsage(deviceID, 75.5)

	// Get the load score for the device
	score := da.getLoadScore(deviceID)

	// Score should be positive
	if score < 0 {
		t.Errorf("Load score should be positive, got %d", score)
	}

	// Update CPU usage again
	da.updateCPUUsage(deviceID, 30.0)

	// Get the load score again
	score2 := da.getLoadScore(deviceID)

	// The score should be different now
	if score2 == score {
		t.Error("Load score should change when CPU usage is updated")
	}
}

func TestDeviceActivity_GetLeastActiveDevices(t *testing.T) {
	// Create a new device activity tracker
	da := newDeviceActivity()

	// Create test devices
	device1 := protocol.DeviceID([32]byte{1, 2, 3, 4})
	device2 := protocol.DeviceID([32]byte{5, 6, 7, 8})
	device3 := protocol.DeviceID([32]byte{9, 10, 11, 12})

	// Create device list
	devices := []protocol.DeviceID{device1, device2, device3}

	// Initially all devices should be equally active
	result := da.getLeastActiveDevices(devices)

	// Should return all devices
	if len(result) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(result))
	}

	// Mark device2 as being used multiple times
	availability2 := TestAvailability{ID: device2}
	da.using(Availability{ID: availability2.ID})
	da.using(Availability{ID: availability2.ID})
	da.using(Availability{ID: availability2.ID})

	// Mark device3 as being used once
	availability3 := TestAvailability{ID: device3}
	da.using(Availability{ID: availability3.ID})

	// Get sorted list of devices
	result = da.getLeastActiveDevices(devices)

	// Should still return all devices
	if len(result) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(result))
	}

	// Device1 should be first (least active)
	if result[0] != device1 {
		t.Errorf("Expected device1 to be first, got %v", result[0])
	}

	// Device3 should be second (less active than device2)
	if result[1] != device3 {
		t.Errorf("Expected device3 to be second, got %v", result[1])
	}

	// Device2 should be last (most active)
	if result[2] != device2 {
		t.Errorf("Expected device2 to be last, got %v", result[2])
	}
}