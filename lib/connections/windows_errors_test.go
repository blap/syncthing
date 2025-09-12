// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package connections

import (
	"syscall"
	"testing"
)

func TestWindowsErrorDetection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		checkFn  func(error) bool
		expected bool
	}{
		{
			name:     "WSAECONNRESET detection",
			err:      WSAECONNRESET,
			checkFn:  isWindowsConnectionReset,
			expected: true,
		},
		{
			name:     "WSAETIMEDOUT detection",
			err:      WSAETIMEDOUT,
			checkFn:  isWindowsConnectionTimeout,
			expected: true,
		},
		{
			name:     "WSAECONNREFUSED detection",
			err:      WSAECONNREFUSED,
			checkFn:  isWindowsConnectionRefused,
			expected: true,
		},
		{
			name:     "WSAENETUNREACH detection",
			err:      WSAENETUNREACH,
			checkFn:  isWindowsNetworkUnreachable,
			expected: true,
		},
		{
			name:     "WSAENETDOWN detection",
			err:      WSAENETDOWN,
			checkFn:  isWindowsNetworkDown,
			expected: true,
		},
		{
			name:     "WSAEHOSTUNREACH detection",
			err:      WSAEHOSTUNREACH,
			checkFn:  isWindowsHostUnreachable,
			expected: true,
		},
		{
			name:     "nil error",
			err:      nil,
			checkFn:  isWindowsConnectionReset,
			expected: false,
		},
		{
			name:     "non-Windows error",
			err:      syscall.EINVAL,
			checkFn:  isWindowsConnectionReset,
			expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.checkFn(tc.err)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestWindowsErrorCategorization(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "WSAECONNRESET categorization",
			err:      WSAECONNRESET,
			expected: ErrorCategoryConnectionReset,
		},
		{
			name:     "WSAETIMEDOUT categorization",
			err:      WSAETIMEDOUT,
			expected: ErrorCategoryTimeout,
		},
		{
			name:     "WSAECONNREFUSED categorization",
			err:      WSAECONNREFUSED,
			expected: ErrorCategoryConnectionRefused,
		},
		{
			name:     "WSAENETUNREACH categorization",
			err:      WSAENETUNREACH,
			expected: ErrorCategoryNetworkUnreachable,
		},
		{
			name:     "WSAENETDOWN categorization",
			err:      WSAENETDOWN,
			expected: ErrorCategoryNetworkDown,
		},
		{
			name:     "WSAEHOSTUNREACH categorization",
			err:      WSAEHOSTUNREACH,
			expected: ErrorCategoryHostUnreachable,
		},
		{
			name:     "nil error categorization",
			err:      nil,
			expected: ErrorCategoryUnknown,
		},
		{
			name:     "non-Windows error categorization",
			err:      syscall.EINVAL,
			expected: ErrorCategoryUnknown,
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