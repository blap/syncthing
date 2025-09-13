// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/syncthing/syncthing/internal/gen/bep"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
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
	// Statistics for monitoring success rates
	stats      peerAssistedStats
	// Peer quality metrics
	peerMetrics map[protocol.DeviceID]*peerQualityMetrics
}

// peerQualityMetrics tracks quality metrics for peers
type peerQualityMetrics struct {
	mut         sync.Mutex
	latency     time.Duration
	packetLoss  float64
	queryCount  int64
	successCount int64
}

// peerAssistedStats tracks statistics for peer-assisted discovery
type peerAssistedStats struct {
	mut           sync.Mutex
	successCount  int64
	failureCount  int64
	timeoutCount  int64
	queryCount    int64
}

// NewPeerAssistedDiscovery creates a new peer-assisted discovery instance
func NewPeerAssistedDiscovery(myID protocol.DeviceID, discoverer Finder, evLogger events.Logger, connSvc protocol.ConnectionServiceSubsetInterface) *PeerAssistedDiscovery {
	return &PeerAssistedDiscovery{
		myID:        myID,
		discoverer:  discoverer,
		evLogger:    evLogger,
		connSvc:     connSvc,
		cache:       newCache(),
		pending:     make(map[protocol.DeviceID]chan []string),
		peerMetrics: make(map[protocol.DeviceID]*peerQualityMetrics),
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

	// Enhance peer selection with quality metrics
	selectedPeers := p.selectQualityPeers(peers, 3) // Select top 3 peers

	slog.DebugContext(ctx, "Querying connected peers for device", 
		"targetDevice", deviceID, 
		"numAvailablePeers", len(peers),
		"numSelectedPeers", len(selectedPeers))

	// Create a channel to collect responses
	resultChan := make(chan []string, len(selectedPeers))

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

	// Send QUERY_DEVICE messages to selected peers
	query := &bep.QueryDevice{
		Id: deviceID[:],
	}

	// Get connections for each selected peer and send query
	connections := make([]protocol.Connection, 0, len(selectedPeers))
	for _, peer := range selectedPeers {
		peerConnections := p.connSvc.GetConnectionsForDevice(peer)
		if len(peerConnections) > 0 {
			// Use the first connection to the peer
			connections = append(connections, peerConnections[0])
		}
	}

	// Send queries to all available connections
	sentQueries := 0
	for _, conn := range connections {
		if err := conn.QueryDevice(ctx, query); err != nil {
			slog.DebugContext(ctx, "Failed to send QUERY_DEVICE to peer", 
				"peer", conn.DeviceID(), 
				"targetDevice", deviceID, 
				"error", err)
			continue
		}
		sentQueries++
		slog.DebugContext(ctx, "Sent QUERY_DEVICE to peer", 
			"peer", conn.DeviceID(), 
			"targetDevice", deviceID)
	}

	if sentQueries == 0 {
		slog.DebugContext(ctx, "Failed to send QUERY_DEVICE to any peers", "targetDevice", deviceID)
		// Update statistics
		p.stats.mut.Lock()
		p.stats.failureCount++
		p.stats.queryCount++
		p.stats.mut.Unlock()
		
		// Cache negative result for 1 minute
		p.cache.Set(deviceID, CacheEntry{
			Addresses: nil,
			when:      time.Now(),
			found:     false,
		})
		return nil, nil
	}

	// Wait for responses with a timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var addresses []string
	collectedResponses := 0
	expectedResponses := sentQueries

	// Collect responses until timeout or we have enough
	for collectedResponses < expectedResponses {
		select {
		case responseAddresses := <-resultChan:
			collectedResponses++
			if len(responseAddresses) > 0 {
				addresses = append(addresses, responseAddresses...)
				// Got valid addresses, we can stop waiting for more responses
				break
			}
		case <-ctxWithTimeout.Done():
			// Timeout reached
			slog.DebugContext(ctx, "Timeout waiting for peer responses", "targetDevice", deviceID)
			goto processResults
		}
	}

processResults:
	// Remove duplicates
	uniqueAddresses := make([]string, 0, len(addresses))
	seen := make(map[string]bool)
	for _, addr := range addresses {
		if !seen[addr] {
			seen[addr] = true
			uniqueAddresses = append(uniqueAddresses, addr)
		}
	}
	addresses = uniqueAddresses

	// Update statistics
	p.stats.mut.Lock()
	if len(addresses) > 0 {
		p.stats.successCount++
	} else {
		p.stats.failureCount++
	}
	p.stats.queryCount++
	p.stats.mut.Unlock()

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

// selectQualityPeers selects peers based on quality metrics
func (p *PeerAssistedDiscovery) selectQualityPeers(peers []protocol.DeviceID, maxPeers int) []protocol.DeviceID {
	// If we don't have metrics for peers, return the first maxPeers peers
	p.mut.Lock()
	if len(p.peerMetrics) == 0 {
		p.mut.Unlock()
		if len(peers) <= maxPeers {
			return peers
		}
		return peers[:maxPeers]
	}
	p.mut.Unlock()

	// Score peers based on quality metrics
	type peerScore struct {
		peer  protocol.DeviceID
		score float64
	}
	
	scores := make([]peerScore, 0, len(peers))
	
	for _, peer := range peers {
		score := p.calculatePeerScore(peer)
		scores = append(scores, peerScore{peer: peer, score: score})
	}
	
	// Sort peers by score (higher is better)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	
	// Select top peers
	selected := make([]protocol.DeviceID, 0, maxPeers)
	for i := 0; i < len(scores) && i < maxPeers; i++ {
		selected = append(selected, scores[i].peer)
	}
	
	return selected
}

// calculatePeerScore calculates a quality score for a peer based on metrics
func (p *PeerAssistedDiscovery) calculatePeerScore(peer protocol.DeviceID) float64 {
	p.mut.Lock()
	metrics, ok := p.peerMetrics[peer]
	p.mut.Unlock()
	
	if !ok {
		// No metrics available, return neutral score
		return 0.5
	}
	
	metrics.mut.Lock()
	defer metrics.mut.Unlock()
	
	// Calculate success rate if we have query data
	var successRate float64 = 1.0
	if metrics.queryCount > 0 {
		successRate = float64(metrics.successCount) / float64(metrics.queryCount)
	}
	
	// Factor in latency (lower is better)
	latencyScore := 1.0
	if metrics.latency > 0 {
		// Normalize latency score (assume 100ms is good, 1000ms is poor)
		latencyScore = 1.0 / (1.0 + float64(metrics.latency/time.Millisecond)/100.0)
	}
	
	// Factor in packet loss (lower is better)
	packetLossScore := 1.0 - metrics.packetLoss/100.0
	
	// Weighted score: 40% success rate, 30% latency, 30% packet loss
	score := 0.4*successRate + 0.3*latencyScore + 0.3*packetLossScore
	
	return score
}

// HandleQueryDevice handles an incoming QUERY_DEVICE message
func (p *PeerAssistedDiscovery) HandleQueryDevice(query *bep.QueryDevice) error {
	deviceID, err := protocol.DeviceIDFromBytes(query.Id)
	if err != nil {
		slog.Debug("Invalid device ID in QUERY_DEVICE message", "error", err)
		return err
	}

	slog.Debug("Handling QUERY_DEVICE message", "queryDeviceID", deviceID)

	// Check if we're connected to the requested device
	connections := p.connSvc.GetConnectionsForDevice(deviceID)
	if len(connections) == 0 {
		// We're not connected to this device
		slog.Debug("Not connected to requested device", "queryDeviceID", deviceID)
		return nil
	}

	// We are connected to this device, get its addresses
	var addresses []string
	for _, conn := range connections {
		addresses = append(addresses, conn.RemoteAddr().String())
	}

	slog.Debug("Found device addresses", "queryDeviceID", deviceID, "addresses", addresses)

	// Send a RESPONSE_DEVICE message back to the querying peer
	// response := &bep.ResponseDevice{
	// 	Id:        query.Id,
	// 	Addresses: addresses,
	// }

	// TODO: Actually send the response through the connection service
	// For now, we'll just log that we would send it
	slog.Debug("Would send RESPONSE_DEVICE with addresses", "targetDevice", deviceID, "addresses", addresses)

	return nil
}

// HandleResponseDevice handles an incoming RESPONSE_DEVICE message
func (p *PeerAssistedDiscovery) HandleResponseDevice(response *bep.ResponseDevice) error {
	deviceID, err := protocol.DeviceIDFromBytes(response.Id)
	if err != nil {
		slog.Debug("Invalid device ID in RESPONSE_DEVICE message", "error", err)
		return err
	}

	slog.Debug("Handling RESPONSE_DEVICE message", "deviceID", deviceID, "addresses", response.Addresses)

	// Check if we have a pending request for this device
	p.mut.Lock()
	resultChan, ok := p.pending[deviceID]
	p.mut.Unlock()

	if !ok {
		// No pending request, ignore the response
		slog.Debug("No pending request for device response", "deviceID", deviceID)
		return nil
	}

	// Send the addresses to the waiting goroutine
	select {
	case resultChan <- response.Addresses:
	default:
		// Channel is full, ignore
	}

	return nil
}

// GetStats returns statistics for peer-assisted discovery
func (p *PeerAssistedDiscovery) GetStats() (success, failure, timeout, total int64) {
	p.stats.mut.Lock()
	defer p.stats.mut.Unlock()
	return p.stats.successCount, p.stats.failureCount, p.stats.timeoutCount, p.stats.queryCount
}

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