// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// mockTLSConn is a mock TLS connection that returns EOF on Handshake
type mockTLSConn struct {
	net.Conn
}

func (m *mockTLSConn) Handshake() error {
	return io.EOF
}

func (m *mockTLSConn) HandshakeContext(ctx context.Context) error {
	return io.EOF
}

func (m *mockTLSConn) ConnectionState() tls.ConnectionState {
	return tls.ConnectionState{}
}

func (m *mockTLSConn) Close() error {
	return nil
}

func (m *mockTLSConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 22000}
}

func (m *mockTLSConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(192, 168, 137, 1), Port: 56624}
}

func (m *mockTLSConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockTLSConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockTLSConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// TestTLSEOFHandling tests that EOF errors during TLS handshake are properly categorized
func TestTLSEOFHandling(t *testing.T) {
	t.Parallel()

	// Create a mock TLS connection that returns EOF
	mockConn := &mockTLSConn{}

	// Test that EOF is categorized as connection reset
	err := mockConn.Handshake()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify that EOF is categorized as connection reset
	category := categorizeError(err)
	if category != ErrorCategoryConnectionReset {
		t.Errorf("Expected ErrorCategoryConnectionReset, got %v", category)
	}

	// Test with net.OpError wrapping EOF
	opErr := &net.OpError{
		Op:  "read",
		Net: "tcp",
		Addr: &net.TCPAddr{
			IP:   net.IPv4(192, 168, 137, 1),
			Port: 56624,
		},
		Err: io.EOF,
	}

	category = categorizeError(opErr)
	if category != ErrorCategoryConnectionReset {
		t.Errorf("Expected ErrorCategoryConnectionReset for net.OpError with EOF, got %v", category)
	}
}

// TestTLSHandshakeRetryWithEOF tests that EOF errors during TLS handshake trigger appropriate retry logic
func TestTLSHandshakeRetryWithEOF(t *testing.T) {
	t.Parallel()

	// Create a mock TLS connection that returns EOF
	mockConn := &mockTLSConn{}

	// Test that EOF gets appropriate retry configuration
	err := mockConn.Handshake()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Categorize the error
	category := categorizeError(err)

	// Get retry configuration based on category
	config := AdaptiveRetryConfig(category)

	// For connection reset errors, we should get specific retry parameters
	if config.MaxRetries != 3 {
		t.Errorf("Expected 3 max retries for connection reset, got %d", config.MaxRetries)
	}

	if config.BaseDelay != 500*time.Millisecond {
		t.Errorf("Expected 500ms base delay for connection reset, got %v", config.BaseDelay)
	}

	if config.BackoffFactor != 1.5 {
		t.Errorf("Expected 1.5 backoff factor for connection reset, got %v", config.BackoffFactor)
	}
}

// TestEOFInRealisticScenario tests EOF handling in a more realistic scenario
func TestEOFInRealisticScenario(t *testing.T) {
	t.Parallel()

	// Simulate the specific error from the issue:
	// "Failed TLS handshake (address=192.168.137.1:56624 error=EOF log.pkg=connections)"

	// Create a realistic error scenario
	err := io.EOF

	// Verify categorization
	category := categorizeError(err)
	if category != ErrorCategoryConnectionReset {
		t.Errorf("Expected EOF to be categorized as connection reset, got %v", category)
	}

	// Verify retry configuration
	config := AdaptiveRetryConfig(category)
	if config.MaxRetries == 0 {
		t.Error("Expected non-zero retry count for EOF errors")
	}

	// This should now use the connection reset retry strategy instead of the default unknown error strategy
	if config.MaxRetries != 3 {
		t.Errorf("Expected connection reset retry strategy (3 retries), got %d retries", config.MaxRetries)
	}
}

// TestErrorCategorizationConsistency tests that our error categorization is consistent
func TestErrorCategorizationConsistency(t *testing.T) {
	t.Parallel()

	// Test various EOF scenarios
	testCases := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "Direct EOF",
			err:      io.EOF,
			expected: ErrorCategoryConnectionReset,
		},
		{
			name:     "Wrapped EOF",
			err:      &net.OpError{Op: "read", Err: io.EOF},
			expected: ErrorCategoryConnectionReset,
		},
		{
			name:     "Connection reset",
			err:      errors.New("connection reset by peer"),
			expected: ErrorCategoryUnknown, // This is not syscall.ECONNRESET
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := categorizeError(tc.err)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}
