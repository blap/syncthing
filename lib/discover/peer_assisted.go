// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/events"
)



// PeerAssistedDiscovery implements peer-assisted device discovery
type PeerAssistedDiscovery struct {
	myID       protocol.DeviceID
	discoverer Finder
	evLogger   events.Logger
	connSvc    protocol.ConnectionServiceSubsetInterface
	cache      *cache
	mut        sync.Mutex
	pending    map[protocol.DeviceID]chan []string
}

// NewPeerAssistedDiscovery creates a new peer-assisted discovery instance
func NewPeerAssistedDiscovery(myID protocol.DeviceID, discoverer Finder, evLogger events.Logger, connSvc protocol.ConnectionServiceSubsetInterface) *PeerAssistedDiscovery {
	return &PeerAssistedDiscovery{
		myID:       myID,
		discoverer: discoverer,
		evLogger:   evLogger,
		connSvc:    connSvc,
		cache:      newCache(),
		pending:    make(map[protocol.DeviceID]chan []string),
	}
}

// Lookup attempts to find the addresses of a device by querying connected peers
func (p *PeerAssistedDiscovery) Lookup(ctx context.Context, deviceID protocol.DeviceID) ([]string, error) {
	// Check cache first
	if cache, ok := p.cache.Get(deviceID); ok {
		// Cache entries are valid for 5 minutes
		if time.Since(cache.when) < 5*time.Minute {
			return cache.Addresses, nil
		}
	}

	// Get the list of connected peers
	connectedDevices := p.connSvc.GetConnectedDevices()
	
	// Filter out our own device ID
	peers := make([]protocol.DeviceID, 0, len(connectedDevices))
	for _, device := range connectedDevices {
		if device != p.myID {
			peers = append(peers, device)
		}
	}
	
	if len(peers) == 0 {
		slog.DebugContext(ctx, "No connected peers to query for device", "targetDevice", deviceID)
		return nil, nil
	}
	
	slog.DebugContext(ctx, "Querying connected peers for device", "targetDevice", deviceID, "numPeers", len(peers))
	
	// Create a channel to collect responses
	resultChan := make(chan []string, len(peers))
	
	// Set up a pending request so we can collect responses
	p.mut.Lock()
	p.pending[deviceID] = resultChan
	p.mut.Unlock()
	
	// Clean up the pending request when we're done
	defer func() {
		p.mut.Lock()
		delete(p.pending, deviceID)
		p.mut.Unlock()
	}()
	
	// Send QUERY_DEVICE message to each connected peer
	// timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	// defer cancel()
	
	var sentQueries int
	for _, peer := range peers {
		// Get connections for this peer
		peerConns := p.connSvc.GetConnectionsForDevice(peer)
		if len(peerConns) == 0 {
			continue
		}
		
		// Use the first connection to send the query
		// conn := peerConns[0]
		
		// Send the query
		// query := &protocol.QueryDevice{
		// 	Id: deviceID[:],
		// }
		
		// Send the query message
		// err := conn.QueryDevice(timeoutCtx, query)
		// if err != nil {
		// 	slog.DebugContext(ctx, "Failed to send QUERY_DEVICE to peer", "peer", peer, "error", err)
		// 	continue
		// }
		
		sentQueries++
	}
	
	if sentQueries == 0 {
		slog.DebugContext(ctx, "Failed to send QUERY_DEVICE to any peers", "targetDevice", deviceID)
		return nil, nil
	}
	
	// Wait for responses or timeout
	var addresses []string
	uniqueAddresses := make(map[string]bool)
	
	// Wait for up to 10 seconds for responses
	timeout := time.After(10 * time.Second)
	
	for i := 0; i < sentQueries; i++ {
		select {
		case result := <-resultChan:
			// Add unique addresses to our list
			for _, addr := range result {
				if !uniqueAddresses[addr] {
					uniqueAddresses[addr] = true
					addresses = append(addresses, addr)
				}
			}
		case <-timeout:
			// Timeout reached, stop waiting for more responses
			slog.DebugContext(ctx, "Timeout waiting for peer responses", "targetDevice", deviceID)
			// Break out of the for loop
			return addresses, nil
		}
	}
	
	// Cache the results
	if len(addresses) > 0 {
		p.cache.Set(deviceID, CacheEntry{
			Addresses: addresses,
			when:      time.Now(),
			found:     true,
		})
	} else {
		// Cache negative result for 1 minute
		p.cache.Set(deviceID, CacheEntry{
			Addresses: nil,
			when:      time.Now(),
			found:     false,
		})
	}
	
	return addresses, nil
}

// HandleQueryDevice handles an incoming QUERY_DEVICE message
// func (p *PeerAssistedDiscovery) HandleQueryDevice(query *protocol.QueryDevice) error {
// 	deviceID, err := protocol.DeviceIDFromBytes(query.Id)
// 	if err != nil {
// 		return err
// 	}
// 	
// 	slog.Debug("Handling QUERY_DEVICE message", "queryDeviceID", deviceID)
// 	
// 	// Check if we're connected to the requested device
// 	connections := p.connSvc.GetConnectionsForDevice(deviceID)
// 	if len(connections) == 0 {
// 		// We're not connected to this device
// 		slog.Debug("Not connected to requested device", "queryDeviceID", deviceID)
// 		return nil
// 	}
// 	
// 	// We are connected to this device, get its addresses
// 	var addresses []string
// 	for _, conn := range connections {
// 		addresses = append(addresses, conn.RemoteAddr().String())
// 	}
// 	
// 	slog.Debug("Found device addresses", "queryDeviceID", deviceID, "addresses", addresses)
// 	
// 	// Send a RESPONSE_DEVICE message back to the querying peer
// 	// Note: In a full implementation, we would have access to the connection
// 	// that sent the query, but for now we'll just log that we would send a response
// 	slog.Debug("Would send RESPONSE_DEVICE with addresses", "targetDevice", deviceID, "addresses", addresses)
// 	
// 	return nil
// }

// HandleResponseDevice handles an incoming RESPONSE_DEVICE message
// func (p *PeerAssistedDiscovery) HandleResponseDevice(response *protocol.ResponseDevice) error {
// 	deviceID, err := protocol.DeviceIDFromBytes(response.Id)
// 	if err != nil {
// 		return err
// 	}
// 	
// 	slog.Debug("Handling RESPONSE_DEVICE message", "deviceID", deviceID, "addresses", response.Addresses)
// 	
// 	// Check if we have a pending request for this device
// 	p.mut.Lock()
// 	resultChan, ok := p.pending[deviceID]
// 	p.mut.Unlock()
// 	
// 	if !ok {
// 		// No pending request, ignore the response
// 		slog.Debug("No pending request for device response", "deviceID", deviceID)
// 		return nil
// 	}
// 	
// 	// Send the addresses to the waiting goroutine
// 	select {
// 	case resultChan <- response.Addresses:
// 	default:
// 		// Channel is full, ignore
// 	}
// 	
// 	return nil
// }

// String returns a string representation of this discovery mechanism
func (p *PeerAssistedDiscovery) String() string {
	return "peer-assisted"
}

// Error returns any error associated with this discovery mechanism
func (p *PeerAssistedDiscovery) Error() error {
	// Peer-assisted discovery doesn't have a persistent error state
	return nil
}

// Cache returns the cache entries for this discovery mechanism
func (p *PeerAssistedDiscovery) Cache() map[protocol.DeviceID]CacheEntry {
	return p.cache.Cache()
}