// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/thejerf/suture/v4"
	"google.golang.org/protobuf/proto"

	"github.com/syncthing/syncthing/internal/gen/discoproto"
	"github.com/syncthing/syncthing/internal/slogutil"
	"github.com/syncthing/syncthing/lib/beacon"
	"github.com/syncthing/syncthing/lib/build"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/rand"
	"github.com/syncthing/syncthing/lib/svcutil"
)

type localClient struct {
	*suture.Supervisor
	myID     protocol.DeviceID
	addrList AddressLister
	name     string
	evLogger events.Logger

	beacon          beacon.Interface
	localBcastStart time.Time
	localBcastTick  <-chan time.Time
	forcedBcastTick chan time.Time

	*cache
	
	// For adaptive broadcast intervals
	broadcastInterval time.Duration
	discoveryStats    *discoveryStatistics
}

// discoveryStatistics tracks discovery success rates for adaptive intervals
type discoveryStatistics struct {
	successCount int
	totalCount   int
	lastUpdate   time.Time
}

const (
	BroadcastInterval = 30 * time.Second
	CacheLifeTime     = 3 * BroadcastInterval
	Magic             = uint32(0x2EA7D90B) // same as in BEP
	v13Magic          = uint32(0x7D79BC40) // previous version
	// Added for v2 compatibility
	v2Magic           = uint32(0x2EA7D90C) // v2 version
	ProtocolVersion   = uint32(2)          // Current protocol version
	
	// Adaptive interval constants
	MinBroadcastInterval = 10 * time.Second
	MaxBroadcastInterval = 60 * time.Second
	AdaptationWindow     = 5 * time.Minute
)

// Feature flags for extended capabilities
const (
	FeatureMultipleConnections = 1 << iota
	FeatureEd25519Keys
	FeatureExtendedAttributes
)

func NewLocal(id protocol.DeviceID, addr string, addrList AddressLister, evLogger events.Logger) (FinderService, error) {
	c := &localClient{
		Supervisor:        suture.New("local", svcutil.SpecWithDebugLogger()),
		myID:              id,
		addrList:          addrList,
		evLogger:          evLogger,
		broadcastInterval: BroadcastInterval,
		discoveryStats:    &discoveryStatistics{},
		localBcastTick:    time.NewTicker(BroadcastInterval).C,
		forcedBcastTick:   make(chan time.Time),
		localBcastStart:   time.Now(),
		cache:             newCache(),
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	if host == "" {
		// A broadcast client
		c.name = "IPv4 local"
		bcPort, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
		c.beacon = beacon.NewBroadcast(bcPort)
	} else {
		// A multicast client
		c.name = "IPv6 local"
		c.beacon = beacon.NewMulticast(addr)
	}
	c.Add(c.beacon)
	c.Add(svcutil.AsService(c.recvAnnouncements, fmt.Sprintf("%s/recv", c)))

	c.Add(svcutil.AsService(c.sendLocalAnnouncements, fmt.Sprintf("%s/sendLocal", c)))

	return c, nil
}

// Lookup returns a list of addresses the device is available at.
func (c *localClient) Lookup(_ context.Context, device protocol.DeviceID) (addresses []string, err error) {
	if cache, ok := c.Get(device); ok {
		if time.Since(cache.when) < CacheLifeTime {
			addresses = cache.Addresses
		}
	}

	return addresses, err
}

func (c *localClient) String() string {
	return c.name
}

func (c *localClient) Error() error {
	return c.beacon.Error()
}

// announcementPkt appends the local discovery packet to send to msg. Returns
// true if the packet should be sent, false if there is nothing useful to
// send.
func (c *localClient) announcementPkt(instanceID int64, msg []byte) ([]byte, bool) {
	addrs := c.addrList.AllAddresses()

	// remove all addresses which are not dialable
	addrs = filterUndialableLocal(addrs)

	// do not leak relay tokens to discovery
	addrs = sanitizeRelayAddresses(addrs)

	if len(addrs) == 0 {
		// Nothing to announce
		return msg, false
	}

	// Get build information for client identification
	clientName := "syncthing"
	clientVersion := build.Version
	if build.Extra != "" {
		clientVersion += "+" + build.Extra
	}

	pkt := &discoproto.Announce{
		Id:            c.myID[:],
		Addresses:     addrs,
		InstanceId:    instanceID,
		Version:       ProtocolVersion,
		Features:      c.getSupportedFeatures(),
		ClientName:    clientName,
		ClientVersion: clientVersion,
	}
	bs, _ := proto.Marshal(pkt)

	if pktLen := 4 + len(bs); cap(msg) < pktLen {
		msg = make([]byte, 0, pktLen)
	}
	msg = msg[:4]
	binary.BigEndian.PutUint32(msg, Magic)
	msg = append(msg, bs...)

	return msg, true
}

// getSupportedFeatures returns a bitmask of features supported by this client
func (c *localClient) getSupportedFeatures() uint64 {
	var features uint64
	
	// Check if we support multiple connections (v2 feature)
	features |= FeatureMultipleConnections
	
	// Check if we support Ed25519 keys (v2 feature)
	features |= FeatureEd25519Keys
	
	// Check if we support extended attributes
	features |= FeatureExtendedAttributes
	
	return features
}

func (c *localClient) sendLocalAnnouncements(ctx context.Context) error {
	var msg []byte
	var ok bool
	instanceID := rand.Int63()
	
	// Use adaptive ticker
	ticker := time.NewTicker(c.broadcastInterval)
	defer ticker.Stop()
	
	for {
		if msg, ok = c.announcementPkt(instanceID, msg[:0]); ok {
			c.beacon.Send(msg)
		}

		select {
		case <-ticker.C:
			// Adapt interval based on discovery success rate
			c.adaptBroadcastInterval()
			ticker.Reset(c.broadcastInterval)
		case <-c.forcedBcastTick:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// adaptBroadcastInterval adjusts the broadcast interval based on discovery success rates
func (c *localClient) adaptBroadcastInterval() {
	stats := c.discoveryStats
	
	// Only adapt if we have enough data
	if stats.totalCount < 5 {
		return
	}
	
	// Reset statistics periodically
	if time.Since(stats.lastUpdate) > AdaptationWindow {
		stats.successCount = 0
		stats.totalCount = 0
		stats.lastUpdate = time.Now()
		return
	}
	
	// Calculate success rate
	successRate := float64(stats.successCount) / float64(stats.totalCount)
	
	// Adjust interval based on success rate
	if successRate > 0.8 {
		// High success rate, we can broadcast less frequently
		c.broadcastInterval = min(c.broadcastInterval*11/10, MaxBroadcastInterval)
	} else if successRate < 0.3 {
		// Low success rate, we should broadcast more frequently
		c.broadcastInterval = max(c.broadcastInterval*9/10, MinBroadcastInterval)
	}
	
	slog.Debug("Adaptive broadcast interval", "interval", c.broadcastInterval, "successRate", successRate)
}

// min returns the smaller of two time.Duration values
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// minInt returns the smaller of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two time.Duration values
func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (c *localClient) recvAnnouncements(ctx context.Context) error {
	b := c.beacon
	warnedAbout := make(map[string]bool)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		buf, addr := b.Recv()
		if addr == nil {
			continue
		}
		if len(buf) < 4 {
			slog.DebugContext(ctx, "received short packet", "address", addr.String())
			continue
		}

		magic := binary.BigEndian.Uint32(buf)
		switch magic {
		case Magic:
			// Current version - all good
			c.handleAnnouncement(ctx, buf, addr, ProtocolVersion)

		case v2Magic:
			// v2 version with extended protocol
			c.handleAnnouncement(ctx, buf, addr, 2)

		case v13Magic:
			// Old version
			if !warnedAbout[addr.String()] {
				slog.ErrorContext(ctx, "Incompatible (v0.13) local discovery packet - upgrade that device to connect", slogutil.Address(addr))
				warnedAbout[addr.String()] = true
			}
			continue

		default:
			slog.DebugContext(ctx, "Incorrect magic", "magic", magic, "address", addr)
			// Log additional information for debugging
			slog.DebugContext(ctx, "Raw packet data", "data", hex.EncodeToString(buf[:minInt(len(buf), 16)]))
			continue
		}
	}
}

// handleAnnouncement processes a received announcement packet
func (c *localClient) handleAnnouncement(ctx context.Context, buf []byte, addr net.Addr, version uint32) {
	var pkt discoproto.Announce
	err := proto.Unmarshal(buf[4:], &pkt)
	if err != nil && !errors.Is(err, io.EOF) {
		slog.DebugContext(ctx, "Failed to unmarshal local announcement", "address", addr, slogutil.Error(err), "packet", hex.Dump(buf[4:]))
		return
	}

	id, _ := protocol.DeviceIDFromBytes(pkt.Id)
	
	// Enhanced logging with more device information
	clientInfo := pkt.ClientName + " " + pkt.ClientVersion
	if clientInfo == " " {
		clientInfo = "unknown"
	}
	
	slog.DebugContext(ctx, "Received local announcement", 
		"address", addr, 
		"device", id, 
		"version", pkt.Version, 
		"client", clientInfo,
		"features", c.getFeatureNames(pkt.Features))

	// Check version compatibility
	if !c.isVersionCompatible(pkt.Version) {
		slog.WarnContext(ctx, "Version incompatibility detected", 
			"device", id, 
			"remoteVersion", pkt.Version, 
			"localVersion", ProtocolVersion,
			"remoteClient", clientInfo)
		
		// Provide specific guidance based on version difference
		if pkt.Version < ProtocolVersion {
			slog.WarnContext(ctx, "Remote device is using an older version - consider upgrading for better compatibility", 
				"device", id)
		} else if pkt.Version > ProtocolVersion {
			slog.WarnContext(ctx, "Remote device is using a newer version - consider upgrading", 
				"device", id)
		}
		// Continue processing but log the incompatibility
	}

	var newDevice bool
	if !bytes.Equal(pkt.Id, c.myID[:]) {
		newDevice = c.registerDevice(addr, &pkt)
	}

	if newDevice {
		// Force a transmit to announce ourselves, if we are ready to do
		// so right away.
		select {
		case c.forcedBcastTick <- time.Now():
		default:
		}
	}
}

// isVersionCompatible checks if the remote protocol version is compatible with ours
func (c *localClient) isVersionCompatible(remoteVersion uint32) bool {
	// For now, we consider versions compatible if they're both >= 1
	// In the future, we might want more sophisticated compatibility checking
	return remoteVersion >= 1 && remoteVersion <= ProtocolVersion+1
}

func (c *localClient) registerDevice(src net.Addr, device *discoproto.Announce) bool {
	// Remember whether we already had a valid cache entry for this device.
	// If the instance ID has changed the remote device has restarted since
	// we last heard from it, so we should treat it as a new device.

	id, err := protocol.DeviceIDFromBytes(device.Id)
	if err != nil {
		slog.Debug("Failed to parse device ID", "deviceIdBytes", device.Id, "error", err)
		return false
	}

	ce, existsAlready := c.Get(id)
	isNewDevice := !existsAlready || time.Since(ce.when) > CacheLifeTime || ce.instanceID != device.InstanceId

	slog.Debug("Device discovery status", 
		"device", id,
		"existsAlready", existsAlready,
		"cacheExpired", time.Since(ce.when) > CacheLifeTime,
		"instanceIdChanged", ce.instanceID != device.InstanceId,
		"isNewDevice", isNewDevice)

	// Update discovery statistics
	c.discoveryStats.totalCount++
	if isNewDevice {
		c.discoveryStats.successCount++
	}
	c.discoveryStats.lastUpdate = time.Now()

	// Any empty or unspecified addresses should be set to the source address
	// of the announcement. We also skip any addresses we can't parse.

	slog.Debug("Registering addresses for device", "device", id, "numAddresses", len(device.Addresses))
	var validAddresses []string
	for i, addr := range device.Addresses {
		slog.Debug("Processing address", "device", id, "addressIndex", i, "address", addr)
		u, err := url.Parse(addr)
		if err != nil {
			slog.Debug("Failed to parse URL", "device", id, "address", addr, "error", err)
			continue
		}

		tcpAddr, err := net.ResolveTCPAddr("tcp", u.Host)
		if err != nil {
			slog.Debug("Failed to resolve TCP address", "device", id, "host", u.Host, "error", err)
			continue
		}

		if len(tcpAddr.IP) == 0 || tcpAddr.IP.IsUnspecified() {
			slog.Debug("Processing unspecified IP address", "device", id, "originalAddress", addr)
			srcAddr, err := net.ResolveTCPAddr("tcp", src.String())
			if err != nil {
				slog.Debug("Failed to resolve source address", "device", id, "source", src.String(), "error", err)
				continue
			}

			// Do not use IPv6 source address if requested scheme is tcp4
			if u.Scheme == "tcp4" && srcAddr.IP.To4() == nil {
				slog.Debug("Skipping IPv6 source address for tcp4 scheme", "device", id)
				continue
			}

			// Do not use IPv4 source address if requested scheme is tcp6
			if u.Scheme == "tcp6" && srcAddr.IP.To4() != nil {
				slog.Debug("Skipping IPv4 source address for tcp6 scheme", "device", id)
				continue
			}

			host, _, err := net.SplitHostPort(src.String())
			if err != nil {
				slog.Debug("Failed to split host port", "device", id, "source", src.String(), "error", err)
				continue
			}
			u.Host = net.JoinHostPort(host, strconv.Itoa(tcpAddr.Port))
			slog.Debug("Reconstructed URL", "device", id, "reconstructedURL", u.String())
			validAddresses = append(validAddresses, u.String())
			slog.Debug("Replaced address", "device", id, "original", addr, "replacedWith", u.String())
		} else {
			validAddresses = append(validAddresses, addr)
			slog.Debug("Accepted address verbatim", "device", id, "address", addr)
		}
	}

	slog.Debug("Updating device cache", "device", id, "numValidAddresses", len(validAddresses), "addresses", validAddresses)
	c.Set(id, CacheEntry{
		Addresses:  validAddresses,
		when:       time.Now(),
		found:      true,
		instanceID: device.InstanceId,
	})

	// Log additional information if available
	if device.Version > 0 || device.ClientName != "" {
		deviceInfo := map[string]interface{}{
			"device":  id.String(),
			"addrs":   validAddresses,
			"version": device.Version,
			"client":  device.ClientName + " " + device.ClientVersion,
		}
		
		// Add feature information if available
		if device.Features > 0 {
			deviceInfo["features"] = c.getFeatureNames(device.Features)
		}
		
		c.evLogger.Log(events.DeviceDiscovered, deviceInfo)
	} else {
		// Fall back to original logging format for compatibility
		c.evLogger.Log(events.DeviceDiscovered, map[string]interface{}{
			"device": id.String(),
			"addrs":  validAddresses,
		})
	}

	return isNewDevice
}

// getFeatureNames returns a slice of feature names for the given feature bitmask
func (c *localClient) getFeatureNames(features uint64) []string {
	var names []string
	if features&FeatureMultipleConnections != 0 {
		names = append(names, "multiple-connections")
	}
	if features&FeatureEd25519Keys != 0 {
		names = append(names, "ed25519-keys")
	}
	if features&FeatureExtendedAttributes != 0 {
		names = append(names, "extended-attributes")
	}
	return names
}

// filterUndialableLocal returns the list of addresses after removing any
// localhost, multicast, broadcast or port-zero addresses.
func filterUndialableLocal(addrs []string) []string {
	filtered := addrs[:0]
	for _, addr := range addrs {
		u, err := url.Parse(addr)
		if err != nil {
			continue
		}

		tcpAddr, err := net.ResolveTCPAddr("tcp", u.Host)
		if err != nil {
			continue
		}

		switch {
		case len(tcpAddr.IP) == 0:
		case tcpAddr.Port == 0:
		case tcpAddr.IP.IsGlobalUnicast(), tcpAddr.IP.IsLinkLocalUnicast(), tcpAddr.IP.IsUnspecified():
			filtered = append(filtered, addr)
		}
	}
	return filtered
}

func sanitizeRelayAddresses(addrs []string) []string {
	filtered := addrs[:0]
	allowlist := []string{"id"}

	for _, addr := range addrs {
		u, err := url.Parse(addr)
		if err != nil {
			continue
		}

		if u.Scheme == "relay" {
			s := url.Values{}
			q := u.Query()

			for _, w := range allowlist {
				if q.Has(w) {
					s.Add(w, q.Get(w))
				}
			}

			u.RawQuery = s.Encode()
			addr = u.String()
		}

		filtered = append(filtered, addr)
	}
	return filtered
}
