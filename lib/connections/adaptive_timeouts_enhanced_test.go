// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestAdaptiveTimeouts_WithVersionCompatibilityIssues(t *testing.T) {
	t.Parallel()

	// Test with version compatibility issues
	at := newAdaptiveTimeouts()
	
	// Simulate version compatibility issues
	at.mut.Lock()
	at.versionCompatibilityIssues = 3 // 3 compatibility issues
	at.connectionSuccessRate = 0.5   // Default success rate
	at.mut.Unlock()
	
	timeout := at.calculateAdaptiveTLSHandshakeTimeout()
	
	// Should be increased due to version compatibility issues
	// Base timeout (15s) + version adjustment (3 * 5s) = 30s
	// With default success rate (0.5), adjusted timeout = 30s * (2.0 - 0.5) = 45s
	expectedMin := 35 * time.Second
	expectedMax := 55 * time.Second
	
	if timeout < expectedMin || timeout > expectedMax {
		t.Errorf("Expected timeout between %v and %v with version issues, got %v", expectedMin, expectedMax, timeout)
	}
	
	// Test reducing version compatibility issues on success
	at.updateConnectionSuccessRate(true, false) // Success, not a version issue
	
	at.mut.Lock()
	issuesAfterSuccess := at.versionCompatibilityIssues
	at.mut.Unlock()
	
	if issuesAfterSuccess >= 3 {
		t.Errorf("Expected version compatibility issues to decrease after success, got %d", issuesAfterSuccess)
	}
}

func TestAdaptiveTimeouts_VersionIssueDetection(t *testing.T) {
	t.Parallel()

	// Test various error types that indicate version compatibility issues
	versionErrors := []error{
		io.EOF,
		errors.New("EOF"),
		errors.New("protocol version mismatch"),
		errors.New("version not supported"),
	}
	
	nonVersionErrors := []error{
		errors.New("network timeout"),
		errors.New("connection refused"),
		errors.New("invalid certificate"),
		errors.New("TLS handshake failed"), // This is not necessarily a version issue
	}
	
	for _, err := range versionErrors {
		isVersionIssue := err != nil && (errors.Is(err, io.EOF) || strings.Contains(err.Error(), "EOF") || 
			strings.Contains(err.Error(), "protocol") || strings.Contains(err.Error(), "version"))
		
		if !isVersionIssue {
			t.Errorf("Expected error '%v' to be detected as version issue", err)
		}
	}
	
	for _, err := range nonVersionErrors {
		isVersionIssue := err != nil && (errors.Is(err, io.EOF) || strings.Contains(err.Error(), "EOF") || 
			strings.Contains(err.Error(), "protocol") || strings.Contains(err.Error(), "version"))
		
		if isVersionIssue {
			t.Errorf("Expected error '%v' to NOT be detected as version issue", err)
		}
	}
}

func TestAdaptiveTimeouts_EnhancedBounds(t *testing.T) {
	t.Parallel()

	at := newAdaptiveTimeouts()
	
	// Test maximum bounds with extreme values
	at.mut.Lock()
	at.versionCompatibilityIssues = 10 // Way above normal
	at.mut.Unlock()
	
	timeout := at.calculateAdaptiveTLSHandshakeTimeout()
	
	// Should be capped at maxTLSHandshakeTimeout (45s)
	if timeout > maxTLSHandshakeTimeout {
		t.Errorf("Expected timeout to be capped at %v, got %v", maxTLSHandshakeTimeout, timeout)
	}
	
	// Test minimum bounds
	at.mut.Lock()
	at.tlsHandshakeTimeout = minTLSHandshakeTimeout
	at.connectionSuccessRate = 1.0 // Perfect success rate
	at.versionCompatibilityIssues = 0
	at.mut.Unlock()
	
	timeout = at.calculateAdaptiveTLSHandshakeTimeout()
	
	// Should be at minimum timeout
	if timeout < minTLSHandshakeTimeout {
		t.Errorf("Expected timeout to be at least %v, got %v", minTLSHandshakeTimeout, timeout)
	}
}