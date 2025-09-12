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
)

// ErrorCategory represents different categories of network errors
type ErrorCategory int

const (
	ErrorCategoryUnknown ErrorCategory = iota
	ErrorCategoryConnectionReset
	ErrorCategoryTimeout
	ErrorCategoryConnectionRefused
	ErrorCategoryNetworkUnreachable
	ErrorCategoryNetworkDown
	ErrorCategoryHostUnreachable
	ErrorCategoryTemporary
	ErrorCategoryAuthentication
)

// String returns a string representation of the error category
func (ec ErrorCategory) String() string {
	switch ec {
	case ErrorCategoryConnectionReset:
		return "connection_reset"
	case ErrorCategoryTimeout:
		return "timeout"
	case ErrorCategoryConnectionRefused:
		return "connection_refused"
	case ErrorCategoryNetworkUnreachable:
		return "network_unreachable"
	case ErrorCategoryNetworkDown:
		return "network_down"
	case ErrorCategoryHostUnreachable:
		return "host_unreachable"
	case ErrorCategoryTemporary:
		return "temporary"
	case ErrorCategoryAuthentication:
		return "authentication"
	default:
		return "unknown"
	}
}

// isTimeoutError checks if an error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for context deadline exceeded
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	
	// Check for net.Error timeout
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	
	// Check for syscall timeout errors
	if errno, ok := err.(syscall.Errno); ok {
		// Common timeout error codes across platforms
		return errno == syscall.ETIMEDOUT
	}
	
	return false
}

// isConnectionResetError checks if an error indicates a connection was reset
func isConnectionResetError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for connection reset errors
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	
	// Check for net.OpError with connection reset
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNRESET) {
			return true
		}
	}
	
	return false
}

// isConnectionRefusedError checks if an error indicates a connection was refused
func isConnectionRefusedError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for connection refused errors
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	
	// Check for net.OpError with connection refused
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return true
		}
	}
	
	return false
}

// isNetworkUnreachableError checks if an error indicates network is unreachable
func isNetworkUnreachableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for network unreachable errors
	if errors.Is(err, syscall.ENETUNREACH) {
		return true
	}
	
	// Check for net.OpError with network unreachable
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ENETUNREACH) {
			return true
		}
	}
	
	return false
}

// isNetworkDownError checks if an error indicates network is down
func isNetworkDownError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for network down errors
	if errors.Is(err, syscall.ENETDOWN) {
		return true
	}
	
	// Check for net.OpError with network down
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ENETDOWN) {
			return true
		}
	}
	
	return false
}

// isHostUnreachableError checks if an error indicates host is unreachable
func isHostUnreachableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for host unreachable errors
	if errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}
	
	// Check for net.OpError with host unreachable
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.EHOSTUNREACH) {
			return true
		}
	}
	
	return false
}

// isTemporaryError checks if an error is temporary and can be retried
func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for temporary errors
	var tempErr interface{ Temporary() bool }
	if errors.As(err, &tempErr) && tempErr.Temporary() {
		return true
	}
	
	// Check for net.Error temporary
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Temporary() {
		return true
	}
	
	return false
}

// categorizeError categorizes network errors into different types for targeted retry strategies
func categorizeError(err error) ErrorCategory {
	switch {
	case isConnectionResetError(err):
		return ErrorCategoryConnectionReset
	case isTimeoutError(err):
		return ErrorCategoryTimeout
	case isConnectionRefusedError(err):
		return ErrorCategoryConnectionRefused
	case isNetworkUnreachableError(err):
		return ErrorCategoryNetworkUnreachable
	case isNetworkDownError(err):
		return ErrorCategoryNetworkDown
	case isHostUnreachableError(err):
		return ErrorCategoryHostUnreachable
	case isTemporaryError(err):
		return ErrorCategoryTemporary
	default:
		// Try Windows-specific error categorization if available
		return categorizeWindowsError(err)
	}
}