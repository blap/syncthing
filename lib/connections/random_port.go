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

// getRandomPort returns a random port within the configured range
func getRandomPort(cfg config.Wrapper) (int, error) {
	opts := cfg.Options()
	
	// If random ports are not enabled, return 0 to use the default behavior
	if !opts.RandomPortsEnabled {
		return 0, nil
	}
	
	// Validate port range
	if opts.RandomPortRangeStart < 1024 || opts.RandomPortRangeEnd > 65535 || 
		opts.RandomPortRangeStart >= opts.RandomPortRangeEnd {
		// Return 0 to fall back to default behavior when range is invalid
		return 0, nil
	}
	
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
	
	// Try up to 10 times to find a free port
	for i := 0; i < 10; i++ {
		port := opts.RandomPortRangeStart + rand.Intn(opts.RandomPortRangeEnd-opts.RandomPortRangeStart+1)
		
		// Check if the port is free
		if isPortFree(port) {
			return port, nil
		}
	}
	
	return 0, fmt.Errorf("unable to find a free port in range %d-%d after 10 attempts", 
		opts.RandomPortRangeStart, opts.RandomPortRangeEnd)
}

// isPortFree checks if a port is free by attempting to listen on it
func isPortFree(port int) bool {
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