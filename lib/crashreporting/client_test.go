// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package crashreporting

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_ReportCrash(t *testing.T) {
	// Create a test server that simulates the crash reporting service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			// For HEAD requests, return 404 to indicate report doesn't exist yet
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			// For PUT requests, read the body and return 200 OK
			_, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("Failed to read request body: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient(server.URL)

	// Test data
	testData := []byte("Panic: test panic\n[device123] Some log line\n[device123] Another log line")

	// Test reporting a crash
	ctx := context.Background()
	err := client.ReportCrash(ctx, testData)
	if err != nil {
		t.Errorf("ReportCrash failed: %v", err)
	}
}

func TestClient_CheckIfReported(t *testing.T) {
	// Create a test server that simulates the crash reporting service
	var headRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			headRequests++
			// For the first request, return 404 (not reported)
			// For the second request, return 200 (already reported)
			if headRequests == 1 {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient(server.URL)

	// Test data
	testData := []byte("Panic: test panic\n[device123] Some log line")

	// First check should return false (not reported)
	ctx := context.Background()
	err := client.ReportCrash(ctx, testData)
	if err != nil {
		t.Errorf("First ReportCrash failed: %v", err)
	}

	// Second check with same data should not upload again (already reported)
	err = client.ReportCrash(ctx, testData)
	if err != nil {
		t.Errorf("Second ReportCrash failed: %v", err)
	}

	// Should have received 2 HEAD requests
	if headRequests != 2 {
		t.Errorf("Expected 2 HEAD requests, got %d", headRequests)
	}
}

func TestClient_UploadWithRetry(t *testing.T) {
	// Create a test server that fails the first request but succeeds on retry
	var putRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			// HEAD requests return 404 (not reported)
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			putRequests++
			// Fail the first request, succeed on retry
			if putRequests == 1 {
				// Simulate a network error by closing the connection
				w.WriteHeader(http.StatusBadGateway)
			} else {
				// Success on retry
				w.WriteHeader(http.StatusOK)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient(server.URL)

	// Test data
	testData := []byte("Panic: test panic\n[device123] Some log line")

	// Test reporting a crash - should retry and succeed
	ctx := context.Background()
	err := client.ReportCrash(ctx, testData)
	if err != nil {
		t.Errorf("ReportCrash failed: %v", err)
	}

	// Should have received 2 PUT requests (1 failed, 1 succeeded)
	if putRequests != 2 {
		t.Errorf("Expected 2 PUT requests, got %d", putRequests)
	}
}

func TestClient_HTTPError(t *testing.T) {
	// Create a test server that returns a client error (400)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			// HEAD requests return 404 (not reported)
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			// Return a client error that shouldn't be retried
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient(server.URL)

	// Test data
	testData := []byte("Panic: test panic\n[device123] Some log line")

	// Test reporting a crash - should fail immediately with client error
	ctx := context.Background()
	err := client.ReportCrash(ctx, testData)
	if err == nil {
		t.Error("Expected ReportCrash to fail with client error")
	}

	// Check that the error is an HTTPError
	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("Expected HTTP 400 error, got: %v", err)
	}
}

func TestFilterLogLines(t *testing.T) {
	// Test data with log lines that should be filtered
	input := []byte(`Some log line at the start
Another log line
Panic: test panic
[device123] Stack trace line 1
[device123] Stack trace line 2
[device123] Stack trace line 3
More log lines at the end`)

	// Expected output (only panic and stack trace)
	expected := []byte(`Panic: test panic
Stack trace line 1
Stack trace line 2
Stack trace line 3`)

	// Filter the log lines
	output := filterLogLines(input)

	// Compare output with expected
	if string(output) != string(expected) {
		t.Errorf("filterLogLines returned unexpected result.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestFilterLogLinesNoPanic(t *testing.T) {
	// Test data without panic lines
	input := []byte(`Some log line at the start
Another log line
More log lines at the end`)

	// Expected output (empty since no panic)
	expected := []byte(`Some log line at the start
Another log line
More log lines at the end`)

	// Filter the log lines
	output := filterLogLines(input)

	// Compare output with expected
	if string(output) != string(expected) {
		t.Errorf("filterLogLines returned unexpected result.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}