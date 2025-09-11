// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package stun

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/ccding/go-stun/stun"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/svcutil"
)

// EnhancedSTUNService provides improved NAT type detection and multiple server support
type EnhancedSTUNService struct {
	name       string
	cfg        config.Wrapper
	subscriber Subscriber
	client     *stun.Client

	lastWriter LastWriter

	natType NATType
	addr    *Host
	
	// Enhanced fields
	mut              sync.RWMutex
	serverResults    map[string]*ServerResult
	detectionHistory []DetectionRecord
}

// ServerResult stores the result from a specific STUN server
type ServerResult struct {
	ServerAddr   string
	NATType      NATType
	ExternalAddr *Host
	Timestamp    time.Time
	Success      bool
	Error        error
}

// DetectionRecord stores historical NAT detection results
type DetectionRecord struct {
	Timestamp   time.Time
	NATType     NATType
	Server      string
	Confidence  float64
}

// NewEnhanced creates a new enhanced STUN service
func NewEnhanced(cfg config.Wrapper, subscriber Subscriber, conn net.PacketConn, lastWriter LastWriter) *EnhancedSTUNService {
	// Construct the client to use the stun conn
	client := stun.NewClientWithConnection(conn)
	client.SetSoftwareName("") // Explicitly unset this, seems to freak some servers out.

	// Return the service and the other conn to the client
	name := "EnhancedStun@"
	if local := conn.LocalAddr(); local != nil {
		name += local.Network() + "://" + local.String()
	} else {
		name += "unknown"
	}
	
	s := &EnhancedSTUNService{
		name: name,
		cfg:  cfg,
		subscriber: subscriber,
		client: client,
		lastWriter: lastWriter,
		natType: NATUnknown,
		addr: nil,
		serverResults: make(map[string]*ServerResult),
		detectionHistory: make([]DetectionRecord, 0),
	}
	return s
}

// Serve runs the enhanced STUN service
func (s *EnhancedSTUNService) Serve(ctx context.Context) error {
	defer func() {
		s.setNATType(NATUnknown)
		s.setExternalAddress(nil, "")
	}()

	timer := time.NewTimer(time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}

		if s.cfg.Options().IsStunDisabled() {
			timer.Reset(time.Second)
			continue
		}

		slog.DebugContext(ctx, "Starting enhanced STUN", "service", s)

		// Query multiple STUN servers for better NAT type detection
		results := s.queryMultipleServers(ctx)
		
		// Analyze results to determine the most accurate NAT type
		natType, extAddr, confidence := s.analyzeResults(results)
		
		// Store the detection result
		s.storeDetectionResult(natType, confidence)
		
		// Update NAT type and external address
		s.setNATType(natType)
		s.setExternalAddress(extAddr, "multiple-servers")
		
		slog.DebugContext(ctx, "Enhanced STUN detection complete", 
			"natType", natType, 
			"externalAddr", extAddr, 
			"confidence", confidence)

		// If NAT is not punchable, we don't need to keep running keepalives
		if !s.isNATTypePunchable(natType) {
			slog.DebugContext(ctx, "NAT type not punchable, skipping keepalives", "natType", natType)
			timer.Reset(stunRetryInterval)
			continue
		}

		// Run keepalive with the best server result
		if len(results) > 0 {
			var bestResult *ServerResult
			for _, result := range results {
				if result.Success && (bestResult == nil || result.Timestamp.After(bestResult.Timestamp)) {
					bestResult = result
				}
			}
			
			if bestResult != nil {
				if err := s.stunKeepAlive(ctx, bestResult.ServerAddr, bestResult.ExternalAddr); err != nil {
					slog.DebugContext(ctx, "STUN keepalive failed", "error", err)
				}
			}
		}

		// Sleep before next detection cycle
		timer.Reset(stunRetryInterval)
	}
}

// queryMultipleServers queries multiple STUN servers and collects results
func (s *EnhancedSTUNService) queryMultipleServers(ctx context.Context) []*ServerResult {
	var results []*ServerResult
	
	servers := s.cfg.Options().StunServers()
	if len(servers) == 0 {
		// Use default servers if none configured
		servers = []string{
			"stun.syncthing.net:3478",
			"stun.callwithus.com:3478",
			"stun.counterpath.com:3478",
			"stun.counterpath.net:3478",
			"stun.e-fellows.net:3478",
			"stun.gmx.de:3478",
			"stun.gmx.net:3478",
			"stun.hosteurope.de:3478",
			"stun.ideasip.com:3478",
			"stun.imesh.com:3478",
			"stun.internetcalls.com:3478",
			"stun.ipns.com:3478",
			"stun.ipphone.com:3478",
			"stun.ivao.aero:3478",
			"stun.jabber.dk:3478",
			"stun.jabber.org:3478",
			"stun.jappix.com:3478",
			"stun.l.google.com:19302",
			"stun.labs.net:3478",
			"stun.linphone.org:3478",
			"stun.liveo.fr:3478",
			"stun.miwifi.com:3478",
			"stun.modem-help.net:3478",
			"stun.myvoiptraffic.com:3478",
			"stun.netappel.com:3478",
			"stun.noc.ams-ix.net:3478",
			"stun.ooma.com:3478",
			"stun.ozekiphone.com:3478",
			"stun.patlive.com:3478",
			"stun.personal-voip.de:3478",
			"stun.pjsip.org:3478",
			"stun.poivy.com:3478",
			"stun.qvod.com:3478",
			"stun.rackco.com:3478",
			"stun.rb-net.com:3478",
			"stun.rixtelecom.se:3478",
			"stun.samsungsmartcam.com:3478",
			"stun.schlund.de:3478",
			"stun.services.mozilla.com:3478",
			"stun.sip.us:3478",
			"stun.sipdiscount.com:3478",
			"stun.sipgate.net:3478",
			"stun.sipgate.net:10000",
			"stun.siplogin.de:3478",
			"stun.sipnet.net:3478",
			"stun.sipnet.ru:3478",
			"stun.solcon.nl:3478",
			"stun.solnet.ch:3478",
			"stun.sonetel.com:3478",
			"stun.sonetel.net:3478",
			"stun.stunprotocol.org:3478",
			"stun.symform.com:3478",
			"stun.t-online.de:3478",
			"stun.tel.lu:3478",
			"stun.telbo.com:3478",
			"stun.telefacil.com:3478",
			"stun.tng.de:3478",
			"stun.twt.it:3478",
			"stun.uls.co.za:3478",
			"stun.voip.aebc.com:3478",
			"stun.voip.blackberry.com:3478",
			"stun.voip.eutelia.it:3478",
			"stun.voiparound.com:3478",
			"stun.voipbuster.com:3478",
			"stun.voipbusterpro.com:3478",
			"stun.voipcheap.co.uk:3478",
			"stun.voipcheap.com:3478",
			"stun.voipfibre.com:3478",
			"stun.voipgain.com:3478",
			"stun.voipgate.com:3478",
			"stun.voipinfocenter.com:3478",
			"stun.voipplanet.nl:3478",
			"stun.voippro.com:3478",
			"stun.voipraider.com:3478",
			"stun.voipstunt.com:3478",
			"stun.voipwise.com:3478",
			"stun.voipzoom.com:3478",
			"stun.vopium.com:3478",
			"stun.voxgratia.org:3478",
			"stun.voxox.com:3478",
			"stun.whoi.edu:3478",
			"stun.xten.com:3478",
			"stun.zadarma.com:3478",
			"stun.zadv.com:3478",
			"stun.zoiper.com:3478",
		}
	}
	
	// Query up to 5 servers concurrently for better performance
	concurrentQueries := 5
	if len(servers) < concurrentQueries {
		concurrentQueries = len(servers)
	}
	
	// Use semaphore to limit concurrent queries
	semaphore := make(chan struct{}, concurrentQueries)
	resultChan := make(chan *ServerResult, len(servers))
	
	var wg sync.WaitGroup
	
	for _, server := range servers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			result := s.querySingleServer(ctx, server)
			resultChan <- result
		}(server)
	}
	
	// Close result channel when all queries are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results with timeout
	timeout := time.After(30 * time.Second)
	
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				// Channel closed, we're done
				return results
			}
			results = append(results, result)
			
		case <-timeout:
			slog.DebugContext(ctx, "STUN query timeout reached")
			return results
		case <-ctx.Done():
			return results
		}
	}
}

// querySingleServer queries a single STUN server
func (s *EnhancedSTUNService) querySingleServer(ctx context.Context, serverAddr string) *ServerResult {
	result := &ServerResult{
		ServerAddr: serverAddr,
		Timestamp:  time.Now(),
	}
	
	// Resolve the address
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve address %s: %w", serverAddr, err)
		return result
	}
	
	s.client.SetServerAddr(udpAddr.String())
	
	// Perform STUN discovery
	var natType stun.NATType
	var extAddr *stun.Host
	err = svcutil.CallWithContext(ctx, func() error {
		natType, extAddr, err = s.client.Discover()
		return err
	})
	
	if err != nil {
		result.Error = fmt.Errorf("STUN discovery failed for %s: %w", serverAddr, err)
		return result
	}
	
	result.NATType = NATType(natType)
	result.ExternalAddr = extAddr
	result.Success = true
	
	// Store result
	s.mut.Lock()
	s.serverResults[serverAddr] = result
	s.mut.Unlock()
	
	return result
}

// analyzeResults analyzes multiple STUN server results to determine the most accurate NAT type
func (s *EnhancedSTUNService) analyzeResults(results []*ServerResult) (NATType, *Host, float64) {
	if len(results) == 0 {
		return NATUnknown, nil, 0.0
	}
	
	// Count successful results by NAT type
	typeCount := make(map[NATType]int)
	successfulResults := make([]*ServerResult, 0)
	
	for _, result := range results {
		if result.Success {
			successfulResults = append(successfulResults, result)
			typeCount[result.NATType]++
		}
	}
	
	if len(successfulResults) == 0 {
		return NATUnknown, nil, 0.0
	}
	
	// Find the most common NAT type
	var mostCommonType NATType
	maxCount := 0
	for natType, count := range typeCount {
		if count > maxCount {
			mostCommonType = natType
			maxCount = count
		}
	}
	
	// Calculate confidence based on consensus
	confidence := float64(maxCount) / float64(len(successfulResults))
	
	// Select the external address from the most common NAT type results
	var selectedAddr *Host
	for _, result := range successfulResults {
		if result.NATType == mostCommonType && result.ExternalAddr != nil {
			selectedAddr = result.ExternalAddr
			break
		}
	}
	
	// If no address found for the most common type, use the first successful result
	if selectedAddr == nil && len(successfulResults) > 0 {
		selectedAddr = successfulResults[0].ExternalAddr
	}
	
	return mostCommonType, selectedAddr, confidence
}

// storeDetectionResult stores the detection result in history
func (s *EnhancedSTUNService) storeDetectionResult(natType NATType, confidence float64) {
	s.mut.Lock()
	defer s.mut.Unlock()
	
	record := DetectionRecord{
		Timestamp:  time.Now(),
		NATType:    natType,
		Confidence: confidence,
	}
	
	s.detectionHistory = append(s.detectionHistory, record)
	
	// Keep only the last 100 records
	if len(s.detectionHistory) > 100 {
		s.detectionHistory = s.detectionHistory[1:]
	}
}

// GetDetectionHistory returns the NAT detection history
func (s *EnhancedSTUNService) GetDetectionHistory() []DetectionRecord {
	s.mut.RLock()
	defer s.mut.RUnlock()
	
	// Return a copy of the history
	history := make([]DetectionRecord, len(s.detectionHistory))
	copy(history, s.detectionHistory)
	return history
}

// GetServerResults returns the latest results from all STUN servers
func (s *EnhancedSTUNService) GetServerResults() map[string]*ServerResult {
	s.mut.RLock()
	defer s.mut.RUnlock()
	
	// Return a copy of the results
	results := make(map[string]*ServerResult)
	for addr, result := range s.serverResults {
		results[addr] = result
	}
	return results
}

// isNATTypePunchable checks if a NAT type is punchable
func (s *EnhancedSTUNService) isNATTypePunchable(natType NATType) bool {
	return natType == NATNone || natType == NATPortRestricted || natType == NATRestricted || natType == NATFull || natType == NATSymmetricUDPFirewall
}

// setNATType updates the NAT type and notifies subscribers
func (s *EnhancedSTUNService) setNATType(natType NATType) {
	if natType != s.natType {
		slog.DebugContext(context.Background(), "Notifying subscriber of NAT type change", 
			"subscriber", s.subscriber, "natType", natType)
		s.subscriber.OnNATTypeChanged(natType)
	}
	s.natType = natType
}

// setExternalAddress updates the external address and notifies subscribers
func (s *EnhancedSTUNService) setExternalAddress(addr *Host, via string) {
	if areDifferent(s.addr, addr) {
		slog.DebugContext(context.Background(), "Notifying subscriber of address change", 
			"subscriber", s.subscriber, "address", addr, "via", via)
		s.subscriber.OnExternalAddressChanged(addr, via)
	}
	s.addr = addr
}

// String returns a string representation of the service
func (s *EnhancedSTUNService) String() string {
	return s.name
}

// stunKeepAlive performs STUN keepalive (same as original implementation)
func (s *EnhancedSTUNService) stunKeepAlive(ctx context.Context, addr string, extAddr *Host) error {
	var err error
	nextSleep := time.Duration(s.cfg.Options().StunKeepaliveStartS) * time.Second

	slog.DebugContext(ctx, "Starting STUN keepalive", "service", s, "addr", addr, "nextSleep", nextSleep)

	var ourLastWrite time.Time
	for {
		if areDifferent(s.addr, extAddr) {
			// If the port has changed (addresses are not equal but the hosts are equal),
			// we're probably spending too much time between keepalives, reduce the sleep.
			if s.addr != nil && extAddr != nil && s.addr.IP() == extAddr.IP() {
				nextSleep /= 2
				slog.DebugContext(ctx, "STUN port change detected", 
					"service", s, "oldAddr", s.addr.TransportAddr(), 
					"newAddr", extAddr.TransportAddr(), "nextSleep", nextSleep)
			}

			s.setExternalAddress(extAddr, addr)

			// The stun server is probably stuffed, we've gone beyond min timeout, yet the address keeps changing.
			minSleep := time.Duration(s.cfg.Options().StunKeepaliveMinS) * time.Second
			if nextSleep < minSleep {
				slog.DebugContext(ctx, "Keepalive aborting, sleep below min", 
					"service", s, "nextSleep", nextSleep, "minSleep", minSleep)
				return fmt.Errorf("unreasonably low keepalive: %v", minSleep)
			}
		}

		// Adjust the keepalives to fire only nextSleep after last write.
		lastWrite := ourLastWrite
		if quicLastWrite := s.lastWriter.LastWrite(); quicLastWrite.After(lastWrite) {
			lastWrite = quicLastWrite
		}
		minSleep := time.Duration(s.cfg.Options().StunKeepaliveMinS) * time.Second
		if nextSleep < minSleep {
			nextSleep = minSleep
		}
	tryLater:
		sleepFor := nextSleep

		timeUntilNextKeepalive := time.Until(lastWrite.Add(sleepFor))
		if timeUntilNextKeepalive > 0 {
			sleepFor = timeUntilNextKeepalive
		}

		slog.DebugContext(ctx, "STUN sleeping", "service", s, "sleepFor", sleepFor)

		select {
		case <-time.After(sleepFor):
		case <-ctx.Done():
			slog.DebugContext(ctx, "Stopping, aborting STUN", "service", s)
			return ctx.Err()
		}

		if s.cfg.Options().IsStunDisabled() {
			// Disabled, give up
			slog.DebugContext(ctx, "STUN disabled, aborting", "service", s)
			return fmt.Errorf("disabled")
		}

		// Check if any writes happened while we were sleeping, if they did, sleep again
		lastWrite = s.lastWriter.LastWrite()
		if gap := time.Since(lastWrite); gap < nextSleep {
			slog.DebugContext(ctx, "STUN last write gap less than next sleep", 
				"service", s, "gap", gap, "nextSleep", nextSleep)
			goto tryLater
		}

		slog.DebugContext(ctx, "STUN keepalive", "service", s)

		extAddr, err = s.client.Keepalive()
		if err != nil {
			slog.DebugContext(ctx, "STUN keepalive failed", "service", s, "addr", addr, "error", err, "extAddr", extAddr)
			return err
		}
		ourLastWrite = time.Now()
	}
}