// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package fs

import (
	"testing"
	"time"
)

// TestOverflowTrackerInitialization tests the initialization of the overflowTracker
func TestOverflowTrackerInitialization(t *testing.T) {
	// Test default initialization
	tracker := newOverflowTracker()
	if tracker == nil {
		t.Fatal("newOverflowTracker returned nil")
	}

	if tracker.minBufferSize != 500 {
		t.Errorf("Expected minBufferSize 500, got %d", tracker.minBufferSize)
	}

	if tracker.maxBufferSize != 10000 {
		t.Errorf("Expected maxBufferSize 10000, got %d", tracker.maxBufferSize)
	}

	if tracker.resizeFactor != 1.5 {
		t.Errorf("Expected resizeFactor 1.5, got %f", tracker.resizeFactor)
	}
}

// TestOverflowTrackerWithConfig tests the initialization of the overflowTracker with configuration
func TestOverflowTrackerWithConfig(t *testing.T) {
	// Test initialization with custom configuration
	tracker := newOverflowTrackerWithConfig(1000, 20000, 200)
	if tracker == nil {
		t.Fatal("newOverflowTrackerWithConfig returned nil")
	}

	if tracker.minBufferSize != 1000 {
		t.Errorf("Expected minBufferSize 1000, got %d", tracker.minBufferSize)
	}

	if tracker.maxBufferSize != 20000 {
		t.Errorf("Expected maxBufferSize 20000, got %d", tracker.maxBufferSize)
	}

	if tracker.resizeFactor != 2.0 {
		t.Errorf("Expected resizeFactor 2.0, got %f", tracker.resizeFactor)
	}
}

// TestOverflowTrackerRecordOverflow tests the recordOverflow functionality
func TestOverflowTrackerRecordOverflow(t *testing.T) {
	tracker := newOverflowTracker()

	// Record first overflow
	tracker.recordOverflow()

	if tracker.count != 1 {
		t.Errorf("Expected count 1, got %d", tracker.count)
	}

	if tracker.lastOverflow.IsZero() {
		t.Error("Expected lastOverflow to be set")
	}

	// Record second overflow quickly after the first
	time.Sleep(10 * time.Millisecond)
	tracker.recordOverflow()

	if tracker.count != 2 {
		t.Errorf("Expected count 2, got %d", tracker.count)
	}

	if tracker.consecutiveOverflows != 2 {
		t.Errorf("Expected consecutiveOverflows 2, got %d", tracker.consecutiveOverflows)
	}

	// Check that overflow history is maintained
	if len(tracker.overflowHistory) != 2 {
		t.Errorf("Expected overflowHistory length 2, got %d", len(tracker.overflowHistory))
	}
}

// TestOverflowTrackerShouldIncreaseBuffer tests the shouldIncreaseBuffer functionality
func TestOverflowTrackerShouldIncreaseBuffer(t *testing.T) {
	tracker := newOverflowTracker()

	// Initially should not increase buffer
	if tracker.shouldIncreaseBuffer() {
		t.Error("Expected shouldIncreaseBuffer to return false initially")
	}

	// Record multiple rapid overflows to trigger buffer increase
	now := time.Now()
	tracker.lastOverflow = now.Add(-5 * time.Second)
	tracker.consecutiveOverflows = 3
	tracker.frequency = 10 * time.Second
	tracker.overflowRate = 3.0

	if !tracker.shouldIncreaseBuffer() {
		t.Error("Expected shouldIncreaseBuffer to return true with rapid overflows")
	}
}

// TestOverflowTrackerShouldDecreaseBuffer tests the shouldDecreaseBuffer functionality
func TestOverflowTrackerShouldDecreaseBuffer(t *testing.T) {
	tracker := newOverflowTracker()

	// Should not decrease buffer initially
	lastEvent := time.Now()
	if tracker.shouldDecreaseBuffer(lastEvent) {
		t.Error("Expected shouldDecreaseBuffer to return false initially")
	}

	// Set conditions for buffer decrease
	tracker.lastOverflow = time.Now().Add(-10 * time.Minute)
	tracker.adaptiveBuffer = tracker.minBufferSize * 3
	tracker.overflowRate = 0.05

	if !tracker.shouldDecreaseBuffer(lastEvent.Add(-15 * time.Minute)) {
		t.Error("Expected shouldDecreaseBuffer to return true with low activity")
	}
}

// TestOverflowTrackerGetSystemPressure tests the getSystemPressure functionality
func TestOverflowTrackerGetSystemPressure(t *testing.T) {
	tracker := newOverflowTracker()

	// Test with low pressure
	pressure := tracker.getSystemPressure()
	if pressure < 0.0 || pressure > 1.0 {
		t.Errorf("Expected system pressure between 0.0 and 1.0, got %f", pressure)
	}

	// Test with high pressure
	tracker.overflowRate = 15.0
	tracker.adaptiveBuffer = tracker.maxBufferSize
	tracker.consecutiveOverflows = 15

	pressure = tracker.getSystemPressure()
	if pressure < 0.0 || pressure > 1.0 {
		t.Errorf("Expected system pressure between 0.0 and 1.0, got %f", pressure)
	}
}

// TestOverflowTrackerIncreaseBuffer tests the increaseBuffer functionality
func TestOverflowTrackerIncreaseBuffer(t *testing.T) {
	tracker := newOverflowTracker()
	originalSize := tracker.adaptiveBuffer

	// Test buffer increase
	newSize := tracker.increaseBuffer()
	if newSize <= originalSize {
		t.Errorf("Expected buffer size to increase from %d, got %d", originalSize, newSize)
	}

	// Test that buffer doesn't exceed maximum
	tracker.adaptiveBuffer = tracker.maxBufferSize - 100
	newSize = tracker.increaseBuffer()
	if newSize > tracker.maxBufferSize {
		t.Errorf("Expected buffer size not to exceed max %d, got %d", tracker.maxBufferSize, newSize)
	}
}

// TestOverflowTrackerDecreaseBuffer tests the decreaseBuffer functionality
func TestOverflowTrackerDecreaseBuffer(t *testing.T) {
	tracker := newOverflowTracker()
	tracker.adaptiveBuffer = 2000

	// Test buffer decrease
	newSize := tracker.decreaseBuffer()
	if newSize >= 2000 {
		t.Errorf("Expected buffer size to decrease from 2000, got %d", newSize)
	}

	// Test that buffer doesn't go below minimum
	tracker.adaptiveBuffer = tracker.minBufferSize + 10
	newSize = tracker.decreaseBuffer()
	if newSize < tracker.minBufferSize {
		t.Errorf("Expected buffer size not to go below min %d, got %d", tracker.minBufferSize, newSize)
	}
}

// TestOverflowTrackerGetOptimalBufferSize tests the getOptimalBufferSize functionality
func TestOverflowTrackerGetOptimalBufferSize(t *testing.T) {
	tracker := newOverflowTracker()

	// Test with small folder
	size := tracker.getOptimalBufferSize(500)
	if size < tracker.minBufferSize || size > tracker.maxBufferSize {
		t.Errorf("Expected buffer size between %d and %d, got %d", tracker.minBufferSize, tracker.maxBufferSize, size)
	}

	// Test with large folder
	size = tracker.getOptimalBufferSize(100000)
	if size < tracker.minBufferSize || size > tracker.maxBufferSize {
		t.Errorf("Expected buffer size between %d and %d, got %d", tracker.minBufferSize, tracker.maxBufferSize, size)
	}
}

// TestOverflowTrackerUpdateBufferSizeBasedOnResources tests the updateBufferSizeBasedOnResources functionality
func TestOverflowTrackerUpdateBufferSizeBasedOnResources(t *testing.T) {
	tracker := newOverflowTracker()
	originalSize := tracker.adaptiveBuffer

	// Test with small difference - should not change
	size := tracker.updateBufferSizeBasedOnResources(1000)
	if size != originalSize {
		t.Errorf("Expected buffer size to remain %d with small difference, got %d", originalSize, size)
	}

	// Test with large difference - should change
	tracker.adaptiveBuffer = 1000
	size = tracker.updateBufferSizeBasedOnResources(50000)
	if size == 1000 {
		t.Error("Expected buffer size to change with large difference")
	}
}

// TestOverflowTrackerGetAdaptiveResizeFactor tests the getAdaptiveResizeFactor functionality
func TestOverflowTrackerGetAdaptiveResizeFactor(t *testing.T) {
	tracker := newOverflowTracker()

	// Test with low pressure
	tracker.overflowRate = 1.0
	tracker.adaptiveBuffer = tracker.minBufferSize + 100
	tracker.consecutiveOverflows = 1

	factor := tracker.getAdaptiveResizeFactor()
	if factor != 1.1 {
		t.Errorf("Expected resize factor 1.1 for low pressure, got %f", factor)
	}

	// Test with medium pressure
	tracker.overflowRate = 5.0
	tracker.adaptiveBuffer = tracker.minBufferSize + 1000
	tracker.consecutiveOverflows = 5

	factor = tracker.getAdaptiveResizeFactor()
	if factor != 1.5 {
		t.Errorf("Expected resize factor 1.5 for medium pressure, got %f", factor)
	}

	// Test with high pressure
	tracker.overflowRate = 8.0
	tracker.adaptiveBuffer = tracker.maxBufferSize - 100
	tracker.consecutiveOverflows = 9

	factor = tracker.getAdaptiveResizeFactor()
	if factor != 2.0 {
		t.Errorf("Expected resize factor 2.0 for high pressure, got %f", factor)
	}
}

// TestOverflowTrackerGetBufferSize tests the getBufferSize functionality
func TestOverflowTrackerGetBufferSize(t *testing.T) {
	tracker := newOverflowTracker()

	// Test initial buffer size
	size := tracker.getBufferSize()
	if size != tracker.adaptiveBuffer {
		t.Errorf("Expected buffer size %d, got %d", tracker.adaptiveBuffer, size)
	}

	// Test after increasing buffer
	tracker.increaseBuffer()
	newSize := tracker.getBufferSize()
	if newSize <= size {
		t.Errorf("Expected buffer size to increase from %d, got %d", size, newSize)
	}
}

// TestOverflowTrackerResetConsecutiveOverflows tests the resetConsecutiveOverflows functionality
func TestOverflowTrackerResetConsecutiveOverflows(t *testing.T) {
	tracker := newOverflowTracker()

	// Set consecutive overflows
	tracker.consecutiveOverflows = 5

	// Reset consecutive overflows
	tracker.resetConsecutiveOverflows()

	if tracker.consecutiveOverflows != 0 {
		t.Errorf("Expected consecutiveOverflows to be 0 after reset, got %d", tracker.consecutiveOverflows)
	}
}

// TestOverflowTrackerAvgOverflowInterval tests the average overflow interval calculation
func TestOverflowTrackerAvgOverflowInterval(t *testing.T) {
	tracker := newOverflowTracker()

	// Record multiple overflows with known intervals
	now := time.Now()
	tracker.overflowHistory = []time.Time{
		now.Add(-30 * time.Second),
		now.Add(-20 * time.Second),
		now.Add(-10 * time.Second),
		now,
	}

	// Manually trigger the calculation that happens in recordOverflow
	tracker.calculateAvgOverflowInterval()

	// With 4 timestamps, we should have 3 intervals of 10 seconds each
	expectedInterval := 10 * time.Second
	if tracker.avgOverflowInterval != expectedInterval {
		t.Errorf("Expected avgOverflowInterval %v, got %v", expectedInterval, tracker.avgOverflowInterval)
	}
}

// TestOverflowTrackerOverflowRate tests the overflow rate calculation
func TestOverflowTrackerOverflowRate(t *testing.T) {
	tracker := newOverflowTracker()

	// Record multiple overflows over a known time period
	now := time.Now()
	tracker.overflowHistory = []time.Time{
		now.Add(-60 * time.Second), // 1 minute ago
		now.Add(-45 * time.Second), // 45 seconds ago
		now.Add(-30 * time.Second), // 30 seconds ago
		now.Add(-15 * time.Second), // 15 seconds ago
		now,                        // now
	}

	// Manually trigger the calculation that happens in recordOverflow
	tracker.calculateOverflowRate()

	// With 5 overflows over 60 seconds, we should have 4 intervals in 1 minute
	// That's 4 overflows per minute
	expectedRate := 4.0
	if tracker.overflowRate != expectedRate {
		t.Errorf("Expected overflowRate %f, got %f", expectedRate, tracker.overflowRate)
	}
}
