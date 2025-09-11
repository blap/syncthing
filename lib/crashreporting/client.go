// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package crashreporting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	// Default timeout values
	headRequestTimeout = 10 * time.Second
	putRequestTimeout  = time.Minute

	// Retry settings
	initialRetryDelay = 1 * time.Second
	maxRetryDelay     = 30 * time.Second
	maxRetryAttempts  = 3
)

// Client handles crash reporting with robust error handling and retry logic
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new crash reporting client
func NewClient(baseURL string) *Client {
	// Create HTTP client with custom transport for better connection handling
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   2 * time.Minute,
		},
		baseURL: baseURL,
	}
}

// ReportCrash attempts to upload a crash report with retry logic and error handling
func (c *Client) ReportCrash(ctx context.Context, data []byte) error {
	// Remove log lines for privacy
	data = filterLogLines(data)

	// Generate hash for deduplication
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	slog.Debug("Reporting crash", "id", hash[:8])

	// Check if already reported
	exists, err := c.checkIfReported(ctx, hash)
	if err != nil {
		return fmt.Errorf("failed to check if crash already reported: %w", err)
	}
	if exists {
		slog.Debug("Crash already reported", "id", hash[:8])
		return nil
	}

	// Upload with retry logic
	return c.uploadWithRetry(ctx, hash, data)
}

// checkIfReported checks if a crash report with the given hash already exists
func (c *Client) checkIfReported(ctx context.Context, hash string) (bool, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, hash)

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	// Set timeout for HEAD request
	headCtx, cancel := context.WithTimeout(ctx, headRequestTimeout)
	defer cancel()
	req = req.WithContext(headCtx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("HEAD request failed: %w", err)
	}
	defer resp.Body.Close()

	// If we get a 200 OK, the report already exists
	return resp.StatusCode == http.StatusOK, nil
}

// uploadWithRetry uploads crash data with exponential backoff retry logic
func (c *Client) uploadWithRetry(ctx context.Context, hash string, data []byte) error {
	url := fmt.Sprintf("%s/%s", c.baseURL, hash)

	var lastErr error
	delay := initialRetryDelay

	for attempt := 0; attempt < maxRetryAttempts; attempt++ {
		if attempt > 0 {
			slog.Debug("Retrying crash report upload",
				"attempt", attempt+1,
				"delay", delay.String())

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}

			// Exponential backoff, capped at maxRetryDelay
			delay *= 2
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
		}

		err := c.upload(ctx, url, data)
		if err == nil {
			slog.Info("Successfully reported crash", "id", hash[:8])
			return nil
		}

		lastErr = err
		slog.Warn("Failed to upload crash report",
			"attempt", attempt+1,
			"error", err)

		// Don't retry on client errors (4xx), only on server errors (5xx) or network issues
		if httpErr, ok := err.(*HTTPError); ok {
			if httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
				// Client error, don't retry
				break
			}
		}
	}

	return fmt.Errorf("failed to upload crash report after %d attempts: %w", maxRetryAttempts, lastErr)
}

// upload sends the crash data to the reporting server
func (c *Client) upload(ctx context.Context, url string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w", err)
	}

	// Set timeout for PUT request
	putCtx, cancel := context.WithTimeout(ctx, putRequestTimeout)
	defer cancel()
	req = req.WithContext(putCtx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for better error messages
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	return nil
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// filterLogLines returns the data without any log lines between the first
// line and the panic trace. This is done in-place: the original data slice
// is destroyed.
func filterLogLines(data []byte) []byte {
	filtered := data[:0]
	matched := false
	for _, line := range bytes.Split(data, []byte("\n")) {
		switch {
		case !matched && bytes.HasPrefix(line, []byte("Panic ")):
			// This begins the panic trace, set the matched flag and append.
			matched = true
			fallthrough
		case len(filtered) == 0 || matched:
			// This is the first line or inside the panic trace.
			if len(filtered) > 0 {
				// We add the newline before rather than after because
				// bytes.Split sees the \n as *separator* and not line
				// ender, so ir will generate a last empty line that we
				// don't really want. (We want to keep blank lines in the
				// middle of the trace though.)
				filtered = append(filtered, '\n')
			}
			// Remove the device ID prefix. The "plus two" stuff is because
			// the line will look like "[foo] whatever" and the end variable
			// will end up pointing at the ] and we want to step over that
			// and the following space.
			if end := bytes.Index(line, []byte("]")); end > 1 && end < len(line)-2 && bytes.HasPrefix(line, []byte("[")) {
				line = line[end+2:]
			}
			filtered = append(filtered, line...)
		}
	}
	return filtered
}
