// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// createTestConfig creates a test configuration for health monitor tests
func createTestConfig() config.Wrapper {
	cfg := config.New(protocol.EmptyDeviceID)
	cfg.Options.AdaptiveKeepAliveEnabled = true
	cfg.Options.AdaptiveKeepAliveMinS = 10
	cfg.Options.AdaptiveKeepAliveMaxS = 60
	return config.Wrap("/tmp/test-config.xml", cfg, protocol.EmptyDeviceID, nil)
}

func TestHealthMonitorRecordError(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"
	err := syscall.ECONNRESET

	// Record an error
	monitor.RecordConnectionError(deviceID, address, err)

	// Check the health status
	health := monitor.GetConnectionHealth(deviceID, address)
	if health == nil {
		t.Fatal("Expected health record, got nil")
	}

	if health.DeviceID != deviceID {
		t.Errorf("Expected DeviceID %v, got %v", deviceID, health.DeviceID)
	}
	if health.Address != address {
		t.Errorf("Expected Address %s, got %s", address, health.Address)
	}
	if health.LastError != err {
		t.Errorf("Expected LastError %v, got %v", err, health.LastError)
	}
	if health.ErrorCategory != ErrorCategoryConnectionReset {
		t.Errorf("Expected ErrorCategory %v, got %v", ErrorCategoryConnectionReset, health.ErrorCategory)
	}
	if health.ConsecutiveErrors != 1 {
		t.Errorf("Expected ConsecutiveErrors 1, got %d", health.ConsecutiveErrors)
	}
	if health.IsHealthy != false {
		t.Errorf("Expected IsHealthy false, got %v", health.IsHealthy)
	}
}

func TestHealthMonitorRecordSuccess(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"

	// Record a success
	monitor.RecordConnectionSuccess(deviceID, address)

	// Check the health status
	health := monitor.GetConnectionHealth(deviceID, address)
	if health == nil {
		t.Fatal("Expected health record, got nil")
	}

	if health.DeviceID != deviceID {
		t.Errorf("Expected DeviceID %v, got %v", deviceID, health.DeviceID)
	}
	if health.Address != address {
		t.Errorf("Expected Address %s, got %s", address, health.Address)
	}
	if health.SuccessCount != 1 {
		t.Errorf("Expected SuccessCount 1, got %d", health.SuccessCount)
	}
	if health.ConsecutiveErrors != 0 {
		t.Errorf("Expected ConsecutiveErrors 0, got %d", health.ConsecutiveErrors)
	}
	if health.IsHealthy != true {
		t.Errorf("Expected IsHealthy true, got %v", health.IsHealthy)
	}
}

func TestHealthMonitorRecordMultipleErrors(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"

	// Record multiple errors
	for i := 0; i < 3; i++ {
		monitor.RecordConnectionError(deviceID, address, syscall.ECONNRESET)
	}

	// Check the health status
	health := monitor.GetConnectionHealth(deviceID, address)
	if health == nil {
		t.Fatal("Expected health record, got nil")
	}

	if health.ConsecutiveErrors != 3 {
		t.Errorf("Expected ConsecutiveErrors 3, got %d", health.ConsecutiveErrors)
	}
	if health.SuccessCount != 0 {
		t.Errorf("Expected SuccessCount 0, got %d", health.SuccessCount)
	}
	if health.IsHealthy != false {
		t.Errorf("Expected IsHealthy false, got %v", health.IsHealthy)
	}
}

func TestHealthMonitorRecordErrorThenSuccess(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"

	// Record an error
	monitor.RecordConnectionError(deviceID, address, syscall.ECONNRESET)

	// Record a success
	monitor.RecordConnectionSuccess(deviceID, address)

	// Check the health status
	health := monitor.GetConnectionHealth(deviceID, address)
	if health == nil {
		t.Fatal("Expected health record, got nil")
	}

	if health.ConsecutiveErrors != 0 {
		t.Errorf("Expected ConsecutiveErrors 0, got %d", health.ConsecutiveErrors)
	}
	if health.SuccessCount != 1 {
		t.Errorf("Expected SuccessCount 1, got %d", health.SuccessCount)
	}
	if health.IsHealthy != true {
		t.Errorf("Expected IsHealthy true, got %v", health.IsHealthy)
	}
}

func TestHealthMonitorGetAllConnectionHealth(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID1 := protocol.LocalDeviceID
	deviceID2 := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	address1 := "192.168.1.100:22000"
	address2 := "192.168.1.101:22000"

	// Record some health data
	monitor.RecordConnectionError(deviceID1, address1, syscall.ECONNRESET)
	monitor.RecordConnectionSuccess(deviceID1, address2)
	monitor.RecordConnectionError(deviceID2, address1, syscall.ECONNREFUSED)

	// Get all health data
	allHealth := monitor.GetAllConnectionHealth()
	if len(allHealth) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(allHealth))
	}

	if len(allHealth[deviceID1]) != 2 {
		t.Errorf("Expected 2 addresses for device 1, got %d", len(allHealth[deviceID1]))
	}

	if len(allHealth[deviceID2]) != 1 {
		t.Errorf("Expected 1 address for device 2, got %d", len(allHealth[deviceID2]))
	}
}

func TestHealthMonitorGetRetryConfigForConnection(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"

	// Record multiple errors
	for i := 0; i < 3; i++ {
		monitor.RecordConnectionError(deviceID, address, syscall.ECONNRESET)
	}

	// Get retry config
	config := monitor.GetRetryConfigForConnection(deviceID, address)

	// Should have increased max retries due to consecutive errors
	if config.MaxRetries < 5 {
		t.Errorf("Expected MaxRetries >= 5, got %d", config.MaxRetries)
	}

	// Should have increased base delay due to consecutive errors
	if config.BaseDelay < 1*time.Second {
		t.Errorf("Expected BaseDelay >= 1s, got %v", config.BaseDelay)
	}
}

func TestHealthMonitorIsConnectionHealthy(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"

	// Initially should be healthy (no data)
	if !monitor.IsConnectionHealthy(deviceID, address) {
		t.Error("Expected connection to be healthy initially")
	}

	// Record an error
	monitor.RecordConnectionError(deviceID, address, errors.New("error"))

	// Should not be healthy now
	if monitor.IsConnectionHealthy(deviceID, address) {
		t.Error("Expected connection to be unhealthy after error")
	}

	// Record a success
	monitor.RecordConnectionSuccess(deviceID, address)

	// Should be healthy now
	if !monitor.IsConnectionHealthy(deviceID, address) {
		t.Error("Expected connection to be healthy after success")
	}
}

func TestHealthMonitorGetErrorRate(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"

	// Initially should have 0% error rate
	if rate := monitor.GetErrorRate(deviceID, address); rate != 0.0 {
		t.Errorf("Expected 0.0 error rate, got %f", rate)
	}

	// Record 2 errors and 3 successes
	for i := 0; i < 2; i++ {
		monitor.RecordConnectionError(deviceID, address, errors.New("error"))
	}
	for i := 0; i < 3; i++ {
		monitor.RecordConnectionSuccess(deviceID, address)
	}

	// Should have 40% error rate (2 errors out of 5 total attempts)
	expectedRate := 2.0 / 5.0
	if rate := monitor.GetErrorRate(deviceID, address); rate != expectedRate {
		t.Errorf("Expected %f error rate, got %f", expectedRate, rate)
	}
}

func TestHealthMonitorCleanupOldStats(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig()
	monitor := NewHealthMonitorWithConfig(cfg, "test-device")
	deviceID := protocol.LocalDeviceID
	address := "192.168.1.100:22000"

	// Record an error with a recent timestamp
	monitor.RecordConnectionError(deviceID, address, errors.New("error"))

	// Check that the record exists
	if health := monitor.GetConnectionHealth(deviceID, address); health == nil {
		t.Fatal("Expected health record to exist")
	}

	// Clean up stats older than 1 hour (our record is newer, so it should remain)
	monitor.CleanupOldStats(1 * time.Hour)

	// Check that the record still exists
	if health := monitor.GetConnectionHealth(deviceID, address); health == nil {
		t.Error("Expected health record to still exist after cleanup")
	}

	// Clean up stats older than 1 nanosecond (our record is older, so it should be removed)
	monitor.CleanupOldStats(1 * time.Nanosecond)

	// Check that the record no longer exists
	if health := monitor.GetConnectionHealth(deviceID, address); health != nil {
		t.Error("Expected health record to be removed after cleanup")
	}
}