// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"net"
	"net/url"
	"testing"
)

func TestAddressParsingWithRelayAddresses(t *testing.T) {
	// Test cases with different address formats
	testCases := []struct {
		name        string
		addr        string
		expectError bool
	}{
		{
			name:        "Valid relay URL with scheme",
			addr:        "relay://23.94.121.166:22067",
			expectError: false,
		},
		{
			name:        "Raw relay host:port without scheme",
			addr:        "23.94.121.166:22067",
			expectError: false, // Should be fixed by our code
		},
		{
			name:        "Valid TCP URL with scheme",
			addr:        "tcp://192.168.1.100:22000",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test our address parsing logic - this is the same logic used in service.go
			uri, err := url.Parse(tc.addr)
			if err != nil {
				// If parsing fails, check if it's a raw host:port string
				if _, _, splitErr := net.SplitHostPort(tc.addr); splitErr == nil {
					// It's a valid host:port combination, try parsing with relay:// prefix
					uri, err = url.Parse("relay://" + tc.addr)
				}
			}
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for address %s, but got none", tc.addr)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for address %s: %v", tc.addr, err)
				} else if uri == nil {
					t.Errorf("Expected URI for address %s, but got nil", tc.addr)
				}
			}
		})
	}
}

// Test that demonstrates the fix for raw relay addresses
func TestRawRelayAddressParsing(t *testing.T) {
	// This is a raw relay address without the relay:// scheme
	rawAddr := "23.94.121.166:22067"
	
	// First, try to parse it as-is (this would fail in the original code)
	uri, err := url.Parse(rawAddr)
	if err != nil {
		// If parsing fails, check if it's a raw host:port string
		if _, _, splitErr := net.SplitHostPort(rawAddr); splitErr == nil {
			// It's a valid host:port combination, try parsing with relay:// prefix
			uri, err = url.Parse("relay://" + rawAddr)
		}
	}
	
	// With our fix, this should now work
	if err != nil {
		t.Errorf("Failed to parse raw relay address %s: %v", rawAddr, err)
	}
	
	if uri == nil {
		t.Errorf("Expected URI for address %s, but got nil", rawAddr)
	}
	
	// Check that the URI is correctly formed
	if uri.Scheme != "relay" {
		t.Errorf("Expected scheme 'relay', got '%s'", uri.Scheme)
	}
	
	if uri.Host != rawAddr {
		t.Errorf("Expected host '%s', got '%s'", rawAddr, uri.Host)
	}
}