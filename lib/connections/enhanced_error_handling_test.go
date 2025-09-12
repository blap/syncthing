// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"net"
	"syscall"
	"testing"

	"github.com/syncthing/syncthing/lib/protocol"
)

func TestEnhancedErrorCategorization(t *testing.T) {
	t.Parallel()

	// Test Windows-specific error categorization
	testCases := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "Windows connection reset",
			err:      syscall.Errno(10054), // WSAECONNRESET
			expected: ErrorCategoryConnectionReset,
		},
		{
			name:     "Windows timeout",
			err:      syscall.Errno(10060), // WSAETIMEDOUT
			expected: ErrorCategoryTimeout,
		},
		{
			name:     "Windows connection refused",
			err:      syscall.Errno(10061), // WSAECONNREFUSED
			expected: ErrorCategoryConnectionRefused,
		},
		{
			name:     "Windows network unreachable",
			err:      syscall.Errno(10051), // WSAENETUNREACH
			expected: ErrorCategoryNetworkUnreachable,
		},
		{
			name:     "Windows network down",
			err:      syscall.Errno(10050), // WSAENETDOWN
			expected: ErrorCategoryNetworkDown,
		},
		{
			name:     "Windows host unreachable",
			err:      syscall.Errno(10065), // WSAEHOSTUNREACH
			expected: ErrorCategoryHostUnreachable,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := categorizeWindowsError(tc.err)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestEnhancedRetryStrategies(t *testing.T) {
	t.Parallel()

	deviceID := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	address := "192.168.1.100:22000"

	// Create a health monitor
	cfg := createTestConfig()
	healthMonitor := NewHealthMonitorWithConfig(cfg, deviceID.String())

	// Test retry configuration for different error categories
	testCases := []struct {
		name     string
		err      error
		category ErrorCategory
	}{
		{
			name:     "Connection reset",
			err:      syscall.ECONNRESET,
			category: ErrorCategoryConnectionReset,
		},
		{
			name:     "Timeout",
			err:      context.DeadlineExceeded,
			category: ErrorCategoryTimeout,
		},
		{
			name:     "Connection refused",
			err:      syscall.ECONNREFUSED,
			category: ErrorCategoryConnectionRefused,
		},
		{
			name:     "Network unreachable",
			err:      syscall.ENETUNREACH,
			category: ErrorCategoryNetworkUnreachable,
		},
		{
			name:     "Network down",
			err:      syscall.ENETDOWN,
			category: ErrorCategoryNetworkDown,
		},
		{
			name:     "Host unreachable",
			err:      syscall.EHOSTUNREACH,
			category: ErrorCategoryHostUnreachable,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Record the error
			healthMonitor.RecordConnectionError(deviceID, address, tc.err)

			// Get retry configuration
			config := healthMonitor.GetRetryConfigForConnection(deviceID, address)

			// Verify that we get a valid configuration
			if config.MaxRetries <= 0 {
				t.Errorf("Expected positive MaxRetries, got %d", config.MaxRetries)
			}
			if config.BaseDelay <= 0 {
				t.Errorf("Expected positive BaseDelay, got %v", config.BaseDelay)
			}
			if config.MaxDelay <= 0 {
				t.Errorf("Expected positive MaxDelay, got %v", config.MaxDelay)
			}
			if config.BackoffFactor <= 0 {
				t.Errorf("Expected positive BackoffFactor, got %v", config.BackoffFactor)
			}

			// Get connection health
			health := healthMonitor.GetConnectionHealth(deviceID, address)
			if health == nil {
				t.Error("Expected connection health, got nil")
			} else if health.ErrorCategory != tc.category {
				t.Errorf("Expected error category %v, got %v", tc.category, health.ErrorCategory)
			}
		})
	}
}

func TestConsecutiveErrorHandling(t *testing.T) {
	t.Parallel()

	deviceID := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	address := "192.168.1.100:22000"

	// Create a health monitor
	cfg := createTestConfig()
	healthMonitor := NewHealthMonitorWithConfig(cfg, deviceID.String())

	// Record multiple consecutive errors
	for i := 0; i < 5; i++ {
		healthMonitor.RecordConnectionError(deviceID, address, syscall.ECONNRESET)
	}

	// Get retry configuration
	config := healthMonitor.GetRetryConfigForConnection(deviceID, address)

	// With consecutive errors, we should see adjusted retry parameters
	// The max retries should be increased
	if config.MaxRetries <= 5 {
		t.Errorf("Expected increased MaxRetries with consecutive errors, got %d", config.MaxRetries)
	}

	// The base delay should be increased
	defaultConfig := DefaultRetryConfig()
	if config.BaseDelay <= defaultConfig.BaseDelay {
		t.Errorf("Expected increased BaseDelay with consecutive errors, got %v (default: %v)", config.BaseDelay, defaultConfig.BaseDelay)
	}

	// Get connection health and verify consecutive errors count
	health := healthMonitor.GetConnectionHealth(deviceID, address)
	if health == nil {
		t.Error("Expected connection health, got nil")
	} else if health.ConsecutiveErrors != 5 {
		t.Errorf("Expected 5 consecutive errors, got %d", health.ConsecutiveErrors)
	} else if health.IsHealthy {
		t.Error("Expected connection to be unhealthy after consecutive errors")
	}
}

func TestErrorRateCalculation(t *testing.T) {
	t.Parallel()

	deviceID := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	address := "192.168.1.100:22000"

	// Create a health monitor
	cfg := createTestConfig()
	healthMonitor := NewHealthMonitorWithConfig(cfg, deviceID.String())

	// Record 3 errors and 2 successes
	for i := 0; i < 3; i++ {
		healthMonitor.RecordConnectionError(deviceID, address, syscall.ECONNRESET)
	}
	for i := 0; i < 2; i++ {
		healthMonitor.RecordConnectionSuccess(deviceID, address)
	}

	// Calculate error rate
	errorRate := healthMonitor.GetErrorRate(deviceID, address)

	// With 3 errors and 2 successes, error rate should be 3/5 = 0.6 = 60%
	expectedRate := 0.6
	if errorRate != expectedRate {
		t.Errorf("Expected error rate %f, got %f", expectedRate, errorRate)
	}

	// Get connection health and verify counts
	health := healthMonitor.GetConnectionHealth(deviceID, address)
	if health == nil {
		t.Error("Expected connection health, got nil")
	} else {
		if health.ConsecutiveErrors != 0 {
			t.Errorf("Expected 0 consecutive errors after success, got %d", health.ConsecutiveErrors)
		}
		if health.SuccessCount != 2 {
			t.Errorf("Expected 2 successes, got %d", health.SuccessCount)
		}
		if !health.IsHealthy {
			t.Error("Expected connection to be healthy after successes")
		}
	}
}

func TestHealthMonitorCleanup(t *testing.T) {
	t.Parallel()

	deviceID := protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	// Create a health monitor with small max stats
	healthMonitor := NewHealthMonitor(2) // Only keep 2 stats per device

	// Record stats for multiple addresses
	for i := 0; i < 5; i++ {
		addr := net.JoinHostPort("192.168.1.100", string(rune(22000+i)))
		healthMonitor.RecordConnectionError(deviceID, addr, syscall.ECONNRESET)
	}

	// Get all connection health
	allHealth := healthMonitor.GetAllConnectionHealth()
	deviceHealth := allHealth[deviceID]

	// Should only have 2 stats due to limit
	if len(deviceHealth) != 2 {
		t.Errorf("Expected 2 connection stats due to limit, got %d", len(deviceHealth))
	}
}
