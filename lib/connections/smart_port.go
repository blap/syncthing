// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/syncthing/syncthing/lib/config"
)

// getSmartPort returns a port based on the smart port strategy:
// 1. Prefer standard ports (22000 for TCP/QUIC) by default
// 2. Only use random ports when there's a conflict detected
// 3. Automatically detect port conflicts and resolve them
func getSmartPort(cfg config.Wrapper, scheme string) (int, error) {
	// Get the default port for this scheme
	defaultPort := getDefaultPortForScheme(scheme)
	
	// First check if the default port is available
	if isPortFreeSmart(defaultPort) {
		return defaultPort, nil
	}
	
	// If the default port is not available, we need to find an alternative
	// Only use random ports if they're enabled in the configuration
	if cfg.Options().RandomPortsEnabled {
		return getRandomPortForSchemeSmart(cfg, scheme)
	}
	
	// If random ports are not enabled and default port is taken, 
	// we have no alternative but to return an error
	return 0, fmt.Errorf("default port %d is not available and random ports are disabled", defaultPort)
}

// getDefaultPortForScheme returns the default port for a given scheme
func getDefaultPortForScheme(scheme string) int {
	switch scheme {
	case "tcp", "tcp4", "tcp6":
		return config.DefaultTCPPort // 22000
	case "quic", "quic4", "quic6":
		return config.DefaultQUICPort // 22000
	default:
		// Fallback to TCP default for unknown schemes
		return config.DefaultTCPPort
	}
}

// isPortFreeSmart checks if a port is free by attempting to listen on it
func isPortFreeSmart(port int) bool {
	// Try TCP first
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	
	// Try UDP
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return false
	}
	udpConn.Close()
	
	return true
}

// getRandomPortForSchemeSmart returns a random port for a specific scheme (tcp/quic)
// with improved conflict detection
func getRandomPortForSchemeSmart(cfg config.Wrapper, _ string) (int, error) {
	opts := cfg.Options()
	
	// Validate port range
	if opts.RandomPortRangeStart < 1024 || opts.RandomPortRangeEnd > 65535 || 
		opts.RandomPortRangeStart >= opts.RandomPortRangeEnd {
		// Return error when range is invalid
		return 0, fmt.Errorf("invalid random port range: %d-%d", 
			opts.RandomPortRangeStart, opts.RandomPortRangeEnd)
	}
	
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
	
	// Try up to 20 times to find a free port (increased from 10 for better chances)
	for i := 0; i < 20; i++ {
		port := opts.RandomPortRangeStart + rand.Intn(opts.RandomPortRangeEnd-opts.RandomPortRangeStart+1)
		
		// Check if the port is free
		if isPortFreeSmart(port) {
			return port, nil
		}
	}
	
	return 0, fmt.Errorf("unable to find a free port in range %d-%d after 20 attempts", 
		opts.RandomPortRangeStart, opts.RandomPortRangeEnd)
}