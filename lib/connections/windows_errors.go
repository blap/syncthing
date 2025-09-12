// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package connections

import (
	"errors"
	"syscall"
)

// Windows-specific error codes
// These are the Windows Sockets error codes that we need to handle
const (
	WSAECONNRESET  = syscall.Errno(10054) // Connection reset by peer
	WSAETIMEDOUT   = syscall.Errno(10060) // Connection timed out
	WSAECONNREFUSED = syscall.Errno(10061) // Connection refused
	WSAENETUNREACH = syscall.Errno(10051) // Network is unreachable
	WSAENETDOWN    = syscall.Errno(10050) // Network is down
	WSAEHOSTUNREACH = syscall.Errno(10065) // No route to host
)

// isWindowsNetworkError checks if an error is a specific Windows network error
func isWindowsNetworkError(err error, errorCode syscall.Errno) bool {
	if err == nil {
		return false
	}
	
	// Check if the error is a syscall.Errno
	if errno, ok := err.(syscall.Errno); ok {
		return errno == errorCode
	}
	
	// Check if the error wraps a syscall.Errno
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == errorCode
	}
	
	return false
}

// isWindowsConnectionReset checks if an error indicates a connection was reset
func isWindowsConnectionReset(err error) bool {
	return isWindowsNetworkError(err, WSAECONNRESET)
}

// isWindowsConnectionTimeout checks if an error indicates a connection timed out
func isWindowsConnectionTimeout(err error) bool {
	return isWindowsNetworkError(err, WSAETIMEDOUT)
}

// isWindowsConnectionRefused checks if an error indicates a connection was refused
func isWindowsConnectionRefused(err error) bool {
	return isWindowsNetworkError(err, WSAECONNREFUSED)
}

// isWindowsNetworkUnreachable checks if an error indicates network is unreachable
func isWindowsNetworkUnreachable(err error) bool {
	return isWindowsNetworkError(err, WSAENETUNREACH)
}

// isWindowsNetworkDown checks if an error indicates network is down
func isWindowsNetworkDown(err error) bool {
	return isWindowsNetworkError(err, WSAENETDOWN)
}

// isWindowsHostUnreachable checks if an error indicates host is unreachable
func isWindowsHostUnreachable(err error) bool {
	return isWindowsNetworkError(err, WSAEHOSTUNREACH)
}

// categorizeWindowsError categorizes Windows network errors into different types
// for targeted retry strategies
func categorizeWindowsError(err error) ErrorCategory {
	switch {
	case isWindowsConnectionReset(err):
		return ErrorCategoryConnectionReset
	case isWindowsConnectionTimeout(err):
		return ErrorCategoryTimeout
	case isWindowsConnectionRefused(err):
		return ErrorCategoryConnectionRefused
	case isWindowsNetworkUnreachable(err):
		return ErrorCategoryNetworkUnreachable
	case isWindowsNetworkDown(err):
		return ErrorCategoryNetworkDown
	case isWindowsHostUnreachable(err):
		return ErrorCategoryHostUnreachable
	default:
		return ErrorCategoryUnknown
	}
}