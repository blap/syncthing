// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
)

func TestErrorCategorization(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorCategoryUnknown,
		},
		{
			name:     "connection reset",
			err:      syscall.ECONNRESET,
			expected: ErrorCategoryConnectionReset,
		},
		{
			name:     "timeout error",
			err:      context.DeadlineExceeded,
			expected: ErrorCategoryTimeout,
		},
		{
			name:     "connection refused",
			err:      syscall.ECONNREFUSED,
			expected: ErrorCategoryConnectionRefused,
		},
		{
			name:     "network unreachable",
			err:      syscall.ENETUNREACH,
			expected: ErrorCategoryNetworkUnreachable,
		},
		{
			name:     "network down",
			err:      syscall.ENETDOWN,
			expected: ErrorCategoryNetworkDown,
		},
		{
			name:     "host unreachable",
			err:      syscall.EHOSTUNREACH,
			expected: ErrorCategoryHostUnreachable,
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown error"),
			expected: ErrorCategoryUnknown,
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

func TestNetErrorCategorization(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "net.OpError with connection reset",
			err:      &net.OpError{Op: "read", Err: syscall.ECONNRESET},
			expected: ErrorCategoryConnectionReset,
		},
		{
			name:     "net.OpError with timeout",
			err:      &net.OpError{Op: "read", Err: context.DeadlineExceeded},
			expected: ErrorCategoryTimeout,
		},
		{
			name:     "net.OpError with connection refused",
			err:      &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			expected: ErrorCategoryConnectionRefused,
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

func TestErrorCategoryString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		category ErrorCategory
		expected string
	}{
		{ErrorCategoryUnknown, "unknown"},
		{ErrorCategoryConnectionReset, "connection_reset"},
		{ErrorCategoryTimeout, "timeout"},
		{ErrorCategoryConnectionRefused, "connection_refused"},
		{ErrorCategoryNetworkUnreachable, "network_unreachable"},
		{ErrorCategoryNetworkDown, "network_down"},
		{ErrorCategoryHostUnreachable, "host_unreachable"},
		{ErrorCategoryTemporary, "temporary"},
		{ErrorCategoryAuthentication, "authentication"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.expected, func(t *testing.T) {
			t.Parallel()
			result := tc.category.String()
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}