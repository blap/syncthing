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
	"sort"
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
	// Embedded fields should be listed first
	*manager
	// Configuration for discovery source priorities and timeouts
	discoveryConfig   discoveryConfiguration
	// Additional peer-assisted discovery
	peerAssistedFinder Finder
	// Error tracking for discovery sources
	errorTracker      *discoveryErrorTracker
}

// discoveryConfiguration holds configuration for discovery sources
type discoveryConfiguration struct {
	// Priority order for discovery sources (higher number = higher priority)
	sourcePriorities map[string]int
	// Timeout for each discovery source
	sourceTimeouts map[string]time.Duration
	// Default timeout for discovery sources
	defaultTimeout time.Duration
}

// discoveryErrorTracker tracks errors for discovery sources
type discoveryErrorTracker struct {
	mut             sync.Mutex
	sourceErrors    map[string][]discoveryError
	maxErrorsPerSource int
}

// discoveryError represents a tracked error
type discoveryError struct {
	timestamp time.Time
	err       error
	deviceID  protocol.DeviceID
}

// discoverySource represents a discovery source with priority
type discoverySource struct {
	identity string
	finder   cachedFinder
	priority int
	timeout  time.Duration
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
	
	// Create default discovery configuration
	discoveryConfig := discoveryConfiguration{
		sourcePriorities: map[string]int{
			"local":  3, // Highest priority
			"global": 2, // Medium priority
			"peer":   1, // Lower priority
		},
		sourceTimeouts: map[string]time.Duration{
			"local":  5 * time.Second,
			"global": 10 * time.Second,
			"peer":   8 * time.Second,
		},
		defaultTimeout: 10 * time.Second,
	}
	
	// Create error tracker
	errorTracker := &discoveryErrorTracker{
		sourceErrors:       make(map[string][]discoveryError),
		maxErrorsPerSource: 50, // Keep track of last 50 errors per source
	}
	
	// Create the enhanced manager
	enhancedManager := &EnhancedManager{
		manager:         stdManager,
		discoveryConfig: discoveryConfig,
		errorTracker:    errorTracker,
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
	
	// Get all discovery sources and sort by priority
	sources := m.getDiscoverySources()
	
	// Query all standard finders concurrently with priority and timeout handling
	for _, source := range sources {
		wg.Add(1)
		go func(source discoverySource) {
			defer wg.Done()
			
			// Create a timeout context for this source
			timeoutCtx, cancel := context.WithTimeout(ctx, source.timeout)
			defer cancel()
			
			// Check cache first
			if cacheEntry, ok := source.finder.cache.Get(deviceID); ok {
				// We have a cache entry. Lets see what it says.
				if cacheEntry.found && time.Since(cacheEntry.when) < source.finder.cacheTime {
					// It's a positive, valid entry. Use it.
					slog.DebugContext(ctx, "Found cached discovery entry", "device", deviceID, "finder", source.identity)
					results <- result{addresses: cacheEntry.Addresses, source: source.identity + " (cached)"}
					return
				}
				
				valid := time.Now().Before(cacheEntry.validUntil) || time.Since(cacheEntry.when) < source.finder.negCacheTime
				if !cacheEntry.found && valid {
					// It's a negative, valid entry. We should not make another attempt right now.
					slog.DebugContext(ctx, "Negative cache entry", "device", deviceID, "finder", source.identity)
					return
				}
				// It's expired. Continue to actual lookup.
			}
			
			// Perform the actual lookup with timeout
			startTime := time.Now()
			if addrs, err := source.finder.Lookup(timeoutCtx, deviceID); err == nil {
				duration := time.Since(startTime)
				slog.DebugContext(ctx, "Got finder result", 
					"device", deviceID, 
					"finder", source.identity, 
					"addresses", addrs,
					"duration", duration)
				
				// Cache the result
				source.finder.cache.Set(deviceID, CacheEntry{
					Addresses: addrs,
					when:      time.Now(),
					found:     len(addrs) > 0,
				})
				results <- result{addresses: addrs, source: source.identity}
			} else {
				duration := time.Since(startTime)
				// Log error with structured information
				slog.WarnContext(ctx, "Discovery source error", 
					"device", deviceID,
					"finder", source.identity,
					"error", err,
					"duration", duration)
				
				// Track the error
				m.trackError(source.identity, deviceID, err)
				
				// Lookup returned error, add a negative cache entry.
				entry := CacheEntry{
					when:  time.Now(),
					found: false,
				}
				if err, ok := err.(cachedError); ok {
					entry.validUntil = time.Now().Add(err.CacheFor())
				}
				source.finder.cache.Set(deviceID, entry)
				results <- result{err: err, source: source.identity}
			}
		}(source)
	}
	
	// Query peer-assisted discovery if available
	if m.peerAssistedFinder != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Create a timeout context for peer-assisted discovery
			peerTimeout := m.discoveryConfig.sourceTimeouts["peer"]
			if peerTimeout == 0 {
				peerTimeout = m.discoveryConfig.defaultTimeout
			}
			timeoutCtx, cancel := context.WithTimeout(ctx, peerTimeout)
			defer cancel()
			
			startTime := time.Now()
			if addrs, err := m.peerAssistedFinder.Lookup(timeoutCtx, deviceID); err == nil {
				duration := time.Since(startTime)
				slog.DebugContext(ctx, "Got peer-assisted discovery result", 
					"device", deviceID, 
					"addresses", addrs,
					"duration", duration)
				results <- result{addresses: addrs, source: "peer-assisted"}
			} else {
				duration := time.Since(startTime)
				// Log error with structured information
				slog.WarnContext(ctx, "Peer-assisted discovery error", 
					"device", deviceID,
					"error", err,
					"duration", duration)
				
				// Track the error
				m.trackError("peer-assisted", deviceID, err)
				
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
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second) // Overall timeout
	defer cancel()
	
	collectedAddresses := make([]string, 0)
	sourcesUsed := make([]string, 0)
	errors := make([]string, 0)
	
	for {
		select {
		case res, ok := <-results:
			if !ok {
				// Channel closed, we're done
				goto processResults
			}
			
			if res.err != nil {
				slog.DebugContext(ctx, "Discovery error", "source", res.source, "error", res.err)
				errors = append(errors, res.source+": "+res.err.Error())
				continue
			}
			
			if len(res.addresses) > 0 {
				collectedAddresses = append(collectedAddresses, res.addresses...)
				sourcesUsed = append(sourcesUsed, res.source)
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
	
	// Create structured log with all relevant information
	logFields := []any{
		"device", deviceID,
		"addresses", addresses,
		"sources", sourcesUsed,
		"numAddresses", len(addresses),
		"numSources", len(sourcesUsed),
	}
	
	if len(errors) > 0 {
		logFields = append(logFields, "errors", errors, "numErrors", len(errors))
	}
	
	slog.DebugContext(ctx, "Enhanced discovery completed", logFields...)
	
	// Update connection cache with successful results
	if len(addresses) > 0 && m.cfg.Options().DiscoveryCacheEnabled {
		m.connectionCache.Add(deviceID, addresses)
	}
	
	return addresses, nil
}

// trackError tracks an error from a discovery source
func (m *EnhancedManager) trackError(source string, deviceID protocol.DeviceID, err error) {
	m.errorTracker.mut.Lock()
	defer m.errorTracker.mut.Unlock()
	
	// Add the error to the tracking list
	m.errorTracker.sourceErrors[source] = append(m.errorTracker.sourceErrors[source], discoveryError{
		timestamp: time.Now(),
		err:       err,
		deviceID:  deviceID,
	})
	
	// Trim the list if it's too long
	if len(m.errorTracker.sourceErrors[source]) > m.errorTracker.maxErrorsPerSource {
		// Remove oldest errors
		m.errorTracker.sourceErrors[source] = m.errorTracker.sourceErrors[source][len(m.errorTracker.sourceErrors[source])-m.errorTracker.maxErrorsPerSource:]
	}
}

// getDiscoverySources returns all discovery sources sorted by priority
func (m *EnhancedManager) getDiscoverySources() []discoverySource {
	m.mut.RLock()
	defer m.mut.RUnlock()
	
	sources := make([]discoverySource, 0, len(m.finders))
	
	for identity, finder := range m.finders {
		// Determine priority
		priority := m.discoveryConfig.sourcePriorities[identity]
		if priority == 0 {
			// Default priority
			priority = 1
		}
		
		// Determine timeout
		timeout := m.discoveryConfig.sourceTimeouts[identity]
		if timeout == 0 {
			timeout = m.discoveryConfig.defaultTimeout
		}
		
		sources = append(sources, discoverySource{
			identity: identity,
			finder:   finder,
			priority: priority,
			timeout:  timeout,
		})
	}
	
	// Sort by priority (highest first)
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].priority > sources[j].priority
	})
	
	return sources
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