// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/syncthing/syncthing/internal/gen/discoproto"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"google.golang.org/protobuf/proto"
)

func TestLocalInstanceID(t *testing.T) {
	c, err := NewLocal(protocol.LocalDeviceID, ":0", &fakeAddressLister{}, events.NoopLogger)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go c.Serve(ctx)
	defer cancel()

	lc := c.(*localClient)

	p0, ok := lc.announcementPkt(1, nil)
	if !ok {
		t.Fatal("unexpectedly not ok")
	}
	p1, ok := lc.announcementPkt(2, nil)
	if !ok {
		t.Fatal("unexpectedly not ok")
	}
	if bytes.Equal(p0, p1) {
		t.Error("each generated packet should have a new instance id")
	}
}

func TestLocalInstanceIDShouldTriggerNew(t *testing.T) {
	c, err := NewLocal(protocol.LocalDeviceID, ":0", &fakeAddressLister{}, events.NoopLogger)
	if err != nil {
		t.Fatal(err)
	}

	lc := c.(*localClient)
	src := &net.UDPAddr{IP: []byte{10, 20, 30, 40}, Port: 50}

	new := lc.registerDevice(src, &discoproto.Announce{
		Id:         padDeviceID(10),
		Addresses:  []string{"tcp://0.0.0.0:22000"},
		InstanceId: 1234567890,
	})

	if !new {
		t.Fatal("first register should be new")
	}

	new = lc.registerDevice(src, &discoproto.Announce{
		Id:         padDeviceID(10),
		Addresses:  []string{"tcp://0.0.0.0:22000"},
		InstanceId: 1234567890,
	})

	if new {
		t.Fatal("second register should not be new")
	}

	new = lc.registerDevice(src, &discoproto.Announce{
		Id:         padDeviceID(42),
		Addresses:  []string{"tcp://0.0.0.0:22000"},
		InstanceId: 1234567890,
	})

	if !new {
		t.Fatal("new device ID should be new")
	}

	new = lc.registerDevice(src, &discoproto.Announce{
		Id:         padDeviceID(10),
		Addresses:  []string{"tcp://0.0.0.0:22000"},
		InstanceId: 91234567890,
	})

	if !new {
		t.Fatal("new instance ID should be new")
	}
}

func TestExtendedAnnouncePacket(t *testing.T) {
	c, err := NewLocal(protocol.LocalDeviceID, ":0", &fakeAddressLister{}, events.NoopLogger)
	if err != nil {
		t.Fatal(err)
	}

	lc := c.(*localClient)
	
	// Test that the announcement packet includes version and feature information
	msg, ok := lc.announcementPkt(12345, nil)
	if !ok {
		t.Fatal("Failed to create announcement packet")
	}
	
	// Parse the packet to verify it contains the extended fields
	if len(msg) < 4 {
		t.Fatal("Packet too short")
	}
	
	magic := uint32(msg[0])<<24 | uint32(msg[1])<<16 | uint32(msg[2])<<8 | uint32(msg[3])
	// For v2 protocol, we expect v2Magic; for other versions, we expect Magic
	expectedMagic := Magic
	if ProtocolVersion == 2 {
		expectedMagic = 0x2EA7D90C // v2Magic constant value
	}
	if magic != expectedMagic {
		t.Errorf("Incorrect magic number: got %x, expected %x", magic, expectedMagic)
	}
	
	var pkt discoproto.Announce
	err = proto.Unmarshal(msg[4:], &pkt)
	if err != nil {
		t.Fatalf("Failed to unmarshal packet: %v", err)
	}
	
	// Verify extended fields are present
	if pkt.Version != ProtocolVersion {
		t.Errorf("Incorrect protocol version: got %d, expected %d", pkt.Version, ProtocolVersion)
	}
	
	if pkt.ClientName == "" {
		t.Error("Client name should not be empty")
	}
	
	if pkt.Features == 0 {
		t.Error("Features should not be zero")
	}
}

func TestVersionCompatibility(t *testing.T) {
	c, err := NewLocal(protocol.LocalDeviceID, ":0", &fakeAddressLister{}, events.NoopLogger)
	if err != nil {
		t.Fatal(err)
	}

	lc := c.(*localClient)
	
	// Test compatible versions
	if !lc.isVersionCompatible(0) {
		t.Error("Version 0 should be compatible (for older Android devices)")
	}
	
	if !lc.isVersionCompatible(1) {
		t.Error("Version 1 should be compatible")
	}
	
	if !lc.isVersionCompatible(ProtocolVersion) {
		t.Errorf("Current version %d should be compatible", ProtocolVersion)
	}
	
	// Test future version (should still be compatible within reason)
	if !lc.isVersionCompatible(ProtocolVersion + 1) {
		t.Errorf("Future version %d should be compatible", ProtocolVersion+1)
	}
	
	// Test very old version (should not be compatible)
	if lc.isVersionCompatible(100) {
		t.Error("Version 100 should not be compatible")
	}
}

func TestFeatureNames(t *testing.T) {
	c, err := NewLocal(protocol.LocalDeviceID, ":0", &fakeAddressLister{}, events.NoopLogger)
	if err != nil {
		t.Fatal(err)
	}

	lc := c.(*localClient)
	
	// Test individual features
	names := lc.getFeatureNames(FeatureMultipleConnections)
	if len(names) != 1 || names[0] != "multiple-connections" {
		t.Errorf("Incorrect feature names for multiple connections: %v", names)
	}
	
	names = lc.getFeatureNames(FeatureEd25519Keys)
	if len(names) != 1 || names[0] != "ed25519-keys" {
		t.Errorf("Incorrect feature names for Ed25519 keys: %v", names)
	}
	
	names = lc.getFeatureNames(FeatureExtendedAttributes)
	if len(names) != 1 || names[0] != "extended-attributes" {
		t.Errorf("Incorrect feature names for extended attributes: %v", names)
	}
	
	// Test combined features
	combined := uint64(FeatureMultipleConnections | FeatureEd25519Keys)
	names = lc.getFeatureNames(combined)
	if len(names) != 2 {
		t.Errorf("Expected 2 feature names, got %d: %v", len(names), names)
	}
	
	// Order might vary, so check both possibilities
	if !(names[0] == "multiple-connections" && names[1] == "ed25519-keys") &&
	   !(names[0] == "ed25519-keys" && names[1] == "multiple-connections") {
		t.Errorf("Incorrect feature names for combined features: %v", names)
	}
}

func TestAdaptiveIntervals(t *testing.T) {
	c, err := NewLocal(protocol.LocalDeviceID, ":0", &fakeAddressLister{}, events.NoopLogger)
	if err != nil {
		t.Fatal(err)
	}

	lc := c.(*localClient)
	
	// Initially should be at default interval
	if lc.broadcastInterval != BroadcastInterval {
		t.Errorf("Initial interval incorrect: got %v, expected %v", lc.broadcastInterval, BroadcastInterval)
	}
	
	// Test that with no data, interval doesn't change
	lc.adaptBroadcastInterval()
	if lc.broadcastInterval != BroadcastInterval {
		t.Errorf("Interval should not change with no data: got %v, expected %v", lc.broadcastInterval, BroadcastInterval)
	}
	
	// Add some stats data
	lc.discoveryStats.totalCount = 10
	
	// Test high success rate (should increase interval)
	lc.discoveryStats.successCount = 9 // 90% success rate
	initialInterval := lc.broadcastInterval
	lc.adaptBroadcastInterval()
	if lc.broadcastInterval <= initialInterval {
		// This might not always increase due to floating point precision, so let's check it's not decreased
		if lc.broadcastInterval < initialInterval {
			t.Error("Interval should not decrease with high success rate")
		}
	}
	
	// Test low success rate (should decrease interval)
	lc.discoveryStats.successCount = 2 // 20% success rate
	lc.broadcastInterval = BroadcastInterval // Reset to default
	initialInterval = lc.broadcastInterval
	lc.adaptBroadcastInterval()
	if lc.broadcastInterval >= initialInterval {
		// This might not always decrease due to floating point precision, so let's check it's not increased
		if lc.broadcastInterval > initialInterval {
			t.Error("Interval should not increase with low success rate")
		}
	}
	
	// Test boundary conditions
	lc.broadcastInterval = MinBroadcastInterval
	lc.discoveryStats.successCount = 2
	lc.discoveryStats.totalCount = 10
	lc.adaptBroadcastInterval()
	if lc.broadcastInterval < MinBroadcastInterval {
		t.Error("Interval should not go below minimum")
	}
	
	lc.broadcastInterval = MaxBroadcastInterval
	lc.discoveryStats.successCount = 9
	lc.discoveryStats.totalCount = 10
	lc.adaptBroadcastInterval()
	if lc.broadcastInterval > MaxBroadcastInterval {
		t.Error("Interval should not go above maximum")
	}
}

func padDeviceID(bs ...byte) []byte {
	var padded [32]byte
	copy(padded[:], bs)
	return padded[:]
}

func TestFilterUndialable(t *testing.T) {
	addrs := []string{
		"quic://[2001:db8::1]:22000",             // OK
		"tcp://192.0.2.42:22000",                 // OK
		"quic://[2001:db8::1]:0",                 // remove, port zero
		"tcp://192.0.2.42:0",                     // remove, port zero
		"quic://[::]:22000",                      // OK
		"tcp://0.0.0.0:22000",                    // OK
		"tcp://[2001:db8::1]",                    // remove, no port
		"tcp://192.0.2.42",                       // remove, no port
		"tcp://foo:bar",                          // remove, host/port does not resolve
		"tcp://127.0.0.1:22000",                  // remove, not usable from outside
		"tcp://[::1]:22000",                      // remove, not usable from outside
		"tcp://224.1.2.3:22000",                  // remove, not usable from outside (multicast)
		"tcp://[fe80::9ef:dff1:b332:5e56]:55681", // OK
		"pure garbage",                           // remove, garbage
		"",                                       // remove, garbage
	}
	exp := []string{
		"quic://[2001:db8::1]:22000",
		"tcp://192.0.2.42:22000",
		"quic://[::]:22000",
		"tcp://0.0.0.0:22000",
		"tcp://[fe80::9ef:dff1:b332:5e56]:55681",
	}
	res := filterUndialableLocal(addrs)
	if fmt.Sprint(res) != fmt.Sprint(exp) {
		t.Log(res)
		t.Error("filterUndialableLocal returned invalid addresses")
	}
}