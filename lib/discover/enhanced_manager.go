// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"context"
	"crypto/tls"
	"log/slog"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/stringutil"
	"github.com/syncthing/syncthing/lib/svcutil"
)

// EnhancedManager extends the standard manager with multi-server support and improved caching
type EnhancedManager struct {
	*manager
	peerAssistedFinder Finder // Additional peer-assisted discovery
}

// NewEnhancedManager creates a new enhanced discovery manager
func NewEnhancedManager(myID protocol.DeviceID, cfg config.Wrapper, cert tls.Certificate, evLogger events.Logger, lister AddressLister, registry *registry.Registry, connSvc protocol.ConnectionServiceSubsetInterface) Manager {
	// Create the standard manager first
	stdManager := &manager{
		Supervisor:    suture.New("discover.EnhancedManager", svcutil.SpecWithDebugLogger()),
		myID:          myID,
		cfg:           cfg,
		cert:          cert,
		evLogger:      evLogger,
		addressLister: lister,
		registry:      registry,
		connSvc:       connSvc,
		finders:       make(map[string]cachedFinder),
		connectionCache: newConnectionCache(60 * time.Minute),
	}
	
	// Create the enhanced manager
	enhancedManager := &EnhancedManager{
		manager: stdManager,
	}
	
	// Add peer-assisted discovery if available
	if peerAssistedFinder := NewPeerAssistedDiscovery(myID, nil, evLogger, connSvc); peerAssistedFinder != nil {
		enhancedManager.peerAssistedFinder = peerAssistedFinder
		// PeerAssistedDiscovery is not a suture.Service, so we don't add it as a service
	}
	
	enhancedManager.Add(svcutil.AsService(enhancedManager.serve, enhancedManager.String()))
	return enhancedManager
}

// serve runs the enhanced discovery manager
func (m *EnhancedManager) serve(ctx context.Context) error {
	m.cfg.Subscribe(m)
	m.CommitConfiguration(config.Configuration{}, m.cfg.RawCopy())
	<-ctx.Done()
	m.cfg.Unsubscribe(m)
	return nil
}

// Lookup performs enhanced discovery with multi-server querying and improved caching
func (m *EnhancedManager) Lookup(ctx context.Context, deviceID protocol.DeviceID) (addresses []string, err error) {
	// First, check our connection cache for recently successful connections
	if m.cfg.Options().DiscoveryCacheEnabled {
		if cachedAddresses, found := m.connectionCache.Get(deviceID); found {
			slog.DebugContext(ctx, "Found device in connection cache", "device", deviceID)
			return cachedAddresses, nil
		}
	}

	// Use channels to collect results from multiple sources concurrently
	type result struct {
		addresses []string
		err       error
		source    string
	}
	
	results := make(chan result, len(m.finders)+2) // +2 for peer-assisted and local
	var wg sync.WaitGroup
	
	// Query all standard finders concurrently
	m.mut.RLock()
	for identity, finder := range m.finders {
		wg.Add(1)
		go func(identity string, finder cachedFinder) {
			defer wg.Done()
			
			// Check cache first
			if cacheEntry, ok := finder.cache.Get(deviceID); ok {
				// We have a cache entry. Lets see what it says.
				if cacheEntry.found && time.Since(cacheEntry.when) < finder.cacheTime {
					// It's a positive, valid entry. Use it.
					slog.DebugContext(ctx, "Found cached discovery entry", "device", deviceID, "finder", identity)
					results <- result{addresses: cacheEntry.Addresses, source: identity + " (cached)"}
					return
				}
				
				valid := time.Now().Before(cacheEntry.validUntil) || time.Since(cacheEntry.when) < finder.negCacheTime
				if !cacheEntry.found && valid {
					// It's a negative, valid entry. We should not make another attempt right now.
					slog.DebugContext(ctx, "Negative cache entry", "device", deviceID, "finder", identity)
					return
				}
				// It's expired. Continue to actual lookup.
			}
			
			// Perform the actual lookup
			if addrs, err := finder.Lookup(ctx, deviceID); err == nil {
				slog.DebugContext(ctx, "Got finder result", "device", deviceID, "finder", identity, "addresses", addrs)
				// Cache the result
				finder.cache.Set(deviceID, CacheEntry{
					Addresses: addrs,
					when:      time.Now(),
					found:     len(addrs) > 0,
				})
				results <- result{addresses: addrs, source: identity}
			} else {
				// Lookup returned error, add a negative cache entry.
				entry := CacheEntry{
					when:  time.Now(),
					found: false,
				}
				if err, ok := err.(cachedError); ok {
					entry.validUntil = time.Now().Add(err.CacheFor())
				}
				finder.cache.Set(deviceID, entry)
				results <- result{err: err, source: identity}
			}
		}(identity, finder)
	}
	m.mut.RUnlock()
	
	// Query peer-assisted discovery if available
	if m.peerAssistedFinder != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if addrs, err := m.peerAssistedFinder.Lookup(ctx, deviceID); err == nil {
				slog.DebugContext(ctx, "Got peer-assisted discovery result", "device", deviceID, "addresses", addrs)
				results <- result{addresses: addrs, source: "peer-assisted"}
			} else {
				results <- result{err: err, source: "peer-assisted"}
			}
		}()
	}
	
	// Close the results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Collect all results with a timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	collectedAddresses := make([]string, 0)
	sources := make([]string, 0)
	
	for {
		select {
		case res, ok := <-results:
			if !ok {
				// Channel closed, we're done
				goto processResults
			}
			
			if res.err != nil {
				slog.DebugContext(ctx, "Discovery error", "source", res.source, "error", res.err)
				continue
			}
			
			if len(res.addresses) > 0 {
				collectedAddresses = append(collectedAddresses, res.addresses...)
				sources = append(sources, res.source)
			}
			
		case <-timeoutCtx.Done():
			// Timeout reached, process what we have
			slog.DebugContext(ctx, "Discovery timeout reached", "device", deviceID)
			goto processResults
		}
	}
	
processResults:
	// Remove duplicates and sort
	addresses = stringutil.UniqueTrimmedStrings(collectedAddresses)
	
	slog.DebugContext(ctx, "Enhanced discovery results", "device", deviceID, "addresses", addresses, "sources", sources)
	
	// Update connection cache with successful results
	if len(addresses) > 0 && m.cfg.Options().DiscoveryCacheEnabled {
		m.connectionCache.Add(deviceID, addresses)
	}
	
	return addresses, nil
}

// CommitConfiguration handles configuration changes for enhanced discovery
func (m *EnhancedManager) CommitConfiguration(from, to config.Configuration) bool {
	// Call the parent implementation first
	m.manager.CommitConfiguration(from, to)
	
	// Handle peer-assisted discovery configuration
	// PeerAssistedDiscovery doesn't have CommitConfiguration method
	_ = m.peerAssistedFinder
	
	return true
}

func (m *EnhancedManager) String() string {
	return "enhanced discovery manager"
}