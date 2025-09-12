// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"
	"time"
)

func TestAdaptiveTimeouts_CalculateAdaptiveTLSHandshakeTimeout(t *testing.T) {
	t.Parallel()

	// Test with default success rate
	at := newAdaptiveTimeouts()
	timeout := at.calculateAdaptiveTLSHandshakeTimeout()
	
	// Should be around the default timeout (10 seconds)
	// With 0.5 success rate, timeout should be 10 * (2.0 - 0.5) = 15 seconds
	if timeout < 13*time.Second || timeout > 17*time.Second {
		t.Errorf("Expected timeout around 15s, got %v", timeout)
	}
	
	// Test with poor success rate (should increase timeout)
	at.mut.Lock()
	at.connectionSuccessRate = 0.3
	at.mut.Unlock()
	
	timeout = at.calculateAdaptiveTLSHandshakeTimeout()
	// With 0.3 success rate, timeout should be 10 * (2.0 - 0.3) = 17 seconds
	if timeout < 15*time.Second || timeout > 19*time.Second {
		t.Errorf("Expected timeout around 17s for poor success rate, got %v", timeout)
	}
	
	// Test with good success rate (should decrease timeout)
	at.mut.Lock()
	at.connectionSuccessRate = 0.9
	at.mut.Unlock()
	
	timeout = at.calculateAdaptiveTLSHandshakeTimeout()
	// With 0.9 success rate, timeout should be 10 * (2.0 - 0.9) = 11 seconds
	if timeout < 9*time.Second || timeout > 13*time.Second {
		t.Errorf("Expected timeout around 11s for good success rate, got %v", timeout)
	}
}

func TestAdaptiveTimeouts_UpdateConnectionSuccessRate(t *testing.T) {
	t.Parallel()

	at := newAdaptiveTimeouts()
	
	// Test initial success rate
	initialRate := at.connectionSuccessRate
	if initialRate != 0.5 {
		t.Errorf("Expected initial success rate of 0.5, got %f", initialRate)
	}
	
	// Test updating with success
	at.updateConnectionSuccessRate(true)
	successRateAfterSuccess := at.connectionSuccessRate
	
	// Test updating with failure
	at.updateConnectionSuccessRate(false)
	successRateAfterFailure := at.connectionSuccessRate
	
	// Success should increase the rate (0.5 * 0.9 + 0.1 = 0.55)
	if successRateAfterSuccess <= 0.5 {
		t.Errorf("Success should increase success rate: %f -> %f", 0.5, successRateAfterSuccess)
	}
	
	// Failure should decrease the rate (0.55 * 0.9 = 0.495)
	if successRateAfterFailure >= successRateAfterSuccess {
		t.Errorf("Failure should decrease success rate: %f -> %f", successRateAfterSuccess, successRateAfterFailure)
	}
}

func TestAdaptiveTimeouts_ProgressiveDialTimeouts(t *testing.T) {
	t.Parallel()

	at := newAdaptiveTimeouts()
	address := "192.168.1.100:22000"
	
	// Test initial timeout
	initialTimeout := at.getProgressiveDialTimeout(address)
	expectedBase := 20 * time.Second
	
	if initialTimeout != expectedBase {
		t.Errorf("Expected initial timeout of %v, got %v", expectedBase, initialTimeout)
	}
	
	// Test increased timeout after failures
	at.recordConnectionFailure(address)
	at.recordConnectionFailure(address)
	
	increasedTimeout := at.getProgressiveDialTimeout(address)
	
	if increasedTimeout <= initialTimeout {
		t.Errorf("Expected increased timeout after failures: %v -> %v", initialTimeout, increasedTimeout)
	}
	
	// Test timeout reduction after success
	at.recordConnectionSuccess(address)
	
	reducedTimeout := at.getProgressiveDialTimeout(address)
	
	if reducedTimeout >= increasedTimeout {
		t.Errorf("Expected reduced timeout after success: %v -> %v", increasedTimeout, reducedTimeout)
	}
}

func TestAdaptiveTimeouts_CalculateAdaptiveConnectionLoopSleep(t *testing.T) {
	t.Parallel()

	at := newAdaptiveTimeouts()
	
	// Test with default success rate
	sleep := at.calculateAdaptiveConnectionLoopSleep()
	
	// Should be around the standard sleep time (1 minute)
	if sleep < 50*time.Second || sleep > 70*time.Second {
		t.Errorf("Expected sleep around 1 minute, got %v", sleep)
	}
	
	// Test with poor success rate (should increase sleep)
	at.mut.Lock()
	at.connectionSuccessRate = 0.2
	at.mut.Unlock()
	
	increasedSleep := at.calculateAdaptiveConnectionLoopSleep()
	
	if increasedSleep <= sleep {
		t.Errorf("Expected increased sleep for poor success rate: %v -> %v", sleep, increasedSleep)
	}
	
	// Test with good success rate (should decrease sleep)
	at.mut.Lock()
	at.connectionSuccessRate = 0.9
	at.mut.Unlock()
	
	decreasedSleep := at.calculateAdaptiveConnectionLoopSleep()
	
	if decreasedSleep >= sleep {
		t.Errorf("Expected decreased sleep for good success rate: %v -> %v", sleep, decreasedSleep)
	}
}

func TestGlobalFunctions(t *testing.T) {
	t.Parallel()

	address := "192.168.1.100:22000"
	
	// Test global functions when service is not available
	timeout := getProgressiveDialTimeoutForAddress(address)
	if timeout != 20*time.Second {
		t.Errorf("Expected default timeout when service not available, got %v", timeout)
	}
	
	// Test recording functions don't panic when service is not available
	recordConnectionFailureForAddress(address)
	recordConnectionSuccessForAddress(address)
}