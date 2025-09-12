// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries    int           // Maximum number of retries
	BaseDelay     time.Duration // Base delay between retries
	MaxDelay      time.Duration // Maximum delay between retries
	Jitter        float64       // Jitter factor (0.0 to 1.0)
	BackoffFactor float64       // Exponential backoff factor
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    5,
		BaseDelay:     1 * time.Second,
		MaxDelay:      60 * time.Second,
		Jitter:        0.1,
		BackoffFactor: 2.0,
	}
}

// AdaptiveRetryConfig returns an adaptive retry configuration based on error category
func AdaptiveRetryConfig(category ErrorCategory) RetryConfig {
	config := DefaultRetryConfig()
	
	switch category {
	case ErrorCategoryConnectionReset:
		// Connection resets might be temporary, retry quickly
		config.MaxRetries = 3
		config.BaseDelay = 500 * time.Millisecond
		config.MaxDelay = 10 * time.Second
		config.BackoffFactor = 1.5
	case ErrorCategoryTimeout:
		// Timeouts might indicate network congestion, use longer delays
		config.MaxRetries = 3
		config.BaseDelay = 2 * time.Second
		config.MaxDelay = 30 * time.Second
		config.BackoffFactor = 2.5
	case ErrorCategoryConnectionRefused:
		// Connection refused might indicate service is down, retry with moderate delays
		config.MaxRetries = 5
		config.BaseDelay = 1 * time.Second
		config.MaxDelay = 60 * time.Second
		config.BackoffFactor = 2.0
	case ErrorCategoryNetworkUnreachable, ErrorCategoryNetworkDown:
		// Network issues might take time to resolve, use longer delays
		config.MaxRetries = 4
		config.BaseDelay = 5 * time.Second
		config.MaxDelay = 120 * time.Second
		config.BackoffFactor = 3.0
	case ErrorCategoryHostUnreachable:
		// Host unreachable might be DNS or routing issues, retry with moderate delays
		config.MaxRetries = 4
		config.BaseDelay = 2 * time.Second
		config.MaxDelay = 60 * time.Second
		config.BackoffFactor = 2.0
	default:
		// Unknown errors, use default configuration
		config.MaxRetries = 3
		config.BaseDelay = 1 * time.Second
		config.MaxDelay = 30 * time.Second
		config.BackoffFactor = 2.0
	}
	
	return config
}

// calculateBackoff calculates the backoff time for a retry attempt
func calculateBackoff(config RetryConfig, attempt int) time.Duration {
	// Calculate exponential backoff
	delay := float64(config.BaseDelay) * math.Pow(config.BackoffFactor, float64(attempt))
	
	// Cap at maximum delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	// Add jitter to prevent thundering herd
	if config.Jitter > 0 && config.Jitter <= 1.0 {
		// Generate random jitter factor between (1-jitter) and (1+jitter)
		jitterFactor := 1.0
		if n, err := rand.Int(rand.Reader, big.NewInt(1000)); err == nil {
			jitterRange := config.Jitter * 2.0
			jitterFactor = 1.0 - config.Jitter + (float64(n.Int64())/1000.0)*jitterRange
		}
		delay *= jitterFactor
	}
	
	return time.Duration(delay)
}

// RetryFunc is a function that can be retried
type RetryFunc func(ctx context.Context) error

// Retry executes a function with retry logic based on the provided configuration
func Retry(ctx context.Context, config RetryConfig, fn RetryFunc) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err := fn(ctx)
		if err == nil {
			// Success, return nil
			return nil
		}
		
		// Store the last error
		lastErr = err
		
		// If this was the last attempt, don't sleep and retry
		if attempt == config.MaxRetries {
			break
		}
		
		// Calculate backoff time
		backoff := calculateBackoff(config, attempt)
		
		// Create a timer for the backoff
		timer := time.NewTimer(backoff)
		
		// Wait for either the backoff to complete or context cancellation
		select {
		case <-timer.C:
			// Backoff completed, continue to next attempt
		case <-ctx.Done():
			// Context cancelled, stop retrying
			timer.Stop()
			return ctx.Err()
		}
	}
	
	// All retries exhausted, return the last error
	return lastErr
}

// RetryWithBackoff executes a function with exponential backoff and jitter
func RetryWithBackoff(ctx context.Context, maxRetries int, baseDelay, maxDelay time.Duration, fn RetryFunc) error {
	config := RetryConfig{
		MaxRetries:    maxRetries,
		BaseDelay:     baseDelay,
		MaxDelay:      maxDelay,
		Jitter:        0.1,
		BackoffFactor: 2.0,
	}
	return Retry(ctx, config, fn)
}