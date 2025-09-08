// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"math/rand"
	"time"
)

// calculateExponentialBackoff calculates the backoff time using exponential backoff strategy
func calculateExponentialBackoff(attempt int, maxBackoffSeconds int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	baseDelay := 1 // Start with 1 second
	backoffSeconds := baseDelay * (1 << uint(attempt-1)) // 2^(attempt-1)
	
	// Cap at maximum backoff
	if backoffSeconds > maxBackoffSeconds {
		backoffSeconds = maxBackoffSeconds
	}
	
	return time.Duration(backoffSeconds) * time.Second
}

// addJitter adds random jitter to the backoff time to prevent thundering herd problems
func addJitter(backoff time.Duration) time.Duration {
	// Add jitter of ±25% of the backoff time
	jitterRange := int64(backoff) / 4
	if jitterRange <= 0 {
		return backoff
	}
	
	// Generate random jitter between -jitterRange and +jitterRange
	jitter := rand.Int63n(2*jitterRange+1) - jitterRange
	
	// Apply jitter to backoff time
	jitteredBackoff := int64(backoff) + jitter
	
	// Ensure we don't go below zero
	if jitteredBackoff < 0 {
		jitteredBackoff = 0
	}
	
	return time.Duration(jitteredBackoff)
}