// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryConfig(t *testing.T) {
	t.Parallel()

	// Test default retry config
	config := DefaultRetryConfig()
	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries=5, got %d", config.MaxRetries)
	}
	if config.BaseDelay != 1*time.Second {
		t.Errorf("Expected BaseDelay=1s, got %v", config.BaseDelay)
	}
	if config.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay=60s, got %v", config.MaxDelay)
	}
	if config.Jitter != 0.1 {
		t.Errorf("Expected Jitter=0.1, got %v", config.Jitter)
	}
	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor=2.0, got %v", config.BackoffFactor)
	}
}

func TestAdaptiveRetryConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		category ErrorCategory
		maxRetries int
		baseDelay time.Duration
	}{
		{ErrorCategoryConnectionReset, 3, 500 * time.Millisecond},
		{ErrorCategoryTimeout, 3, 2 * time.Second},
		{ErrorCategoryConnectionRefused, 5, 1 * time.Second},
		{ErrorCategoryNetworkUnreachable, 4, 5 * time.Second},
		{ErrorCategoryNetworkDown, 4, 5 * time.Second},
		{ErrorCategoryHostUnreachable, 4, 2 * time.Second},
		{ErrorCategoryUnknown, 3, 1 * time.Second},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.category.String(), func(t *testing.T) {
			t.Parallel()
			config := AdaptiveRetryConfig(tc.category)
			if config.MaxRetries != tc.maxRetries {
				t.Errorf("Expected MaxRetries=%d, got %d", tc.maxRetries, config.MaxRetries)
			}
			if config.BaseDelay != tc.baseDelay {
				t.Errorf("Expected BaseDelay=%v, got %v", tc.baseDelay, config.BaseDelay)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	t.Parallel()

	config := RetryConfig{
		BaseDelay:     1 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        0.0, // No jitter for predictable tests
	}

	testCases := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 32 * time.Second},
		{6, 60 * time.Second}, // Capped at MaxDelay
	}

	for _, tc := range testCases {
		tc := tc
		t.Run("", func(t *testing.T) {
			t.Parallel()
			result := calculateBackoff(config, tc.attempt)
			if result != tc.expected {
				t.Errorf("Attempt %d: Expected %v, got %v", tc.attempt, tc.expected, result)
			}
		})
	}
}

func TestRetrySuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond, // Very short for testing
		MaxDelay:   10 * time.Millisecond,
		Jitter:     0.0,
	}

	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on third attempt
	}

	err := Retry(ctx, config, fn)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestRetryFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := RetryConfig{
		MaxRetries: 2,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Jitter:     0.0,
	}

	callCount := 0
	expectedErr := errors.New("persistent error")
	fn := func(ctx context.Context) error {
		callCount++
		return expectedErr
	}

	err := Retry(ctx, config, fn)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if callCount != 3 { // Initial call + 2 retries
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestRetryWithContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	config := RetryConfig{
		MaxRetries: 10,
		BaseDelay:  100 * time.Millisecond, // Long delay to test cancellation
		MaxDelay:   1 * time.Second,
		Jitter:     0.0,
	}

	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		// Cancel context on second call to test cancellation
		if callCount == 2 {
			cancel()
		}
		return errors.New("error")
	}

	err := Retry(ctx, config, fn)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}