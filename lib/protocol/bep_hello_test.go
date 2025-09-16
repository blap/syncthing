// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestVersion14Hello(t *testing.T) {
	// Tests that we can send and receive a version 0.14 hello message.

	expected := Hello{
		DeviceName:    "test device",
		ClientName:    "syncthing",
		ClientVersion: "v0.14.5",
	}
	msgBuf, err := proto.Marshal(expected.toWire())
	if err != nil {
		t.Fatal(err)
	}

	hdrBuf := make([]byte, 6)
	binary.BigEndian.PutUint32(hdrBuf, HelloMessageMagic)
	binary.BigEndian.PutUint16(hdrBuf[4:], uint16(len(msgBuf)))

	outBuf := new(bytes.Buffer)
	outBuf.Write(hdrBuf)
	outBuf.Write(msgBuf)

	inBuf := new(bytes.Buffer)

	conn := &readWriter{outBuf, inBuf}

	send := Hello{
		DeviceName:    "this device",
		ClientName:    "other client",
		ClientVersion: "v0.14.6",
		Timestamp:     1234567890,
	}

	res, err := ExchangeHello(conn, send)
	if err != nil {
		t.Fatal(err)
	}

	if res.ClientName != expected.ClientName {
		t.Errorf("incorrect ClientName %q != expected %q", res.ClientName, expected.ClientName)
	}
	if res.ClientVersion != expected.ClientVersion {
		t.Errorf("incorrect ClientVersion %q != expected %q", res.ClientVersion, expected.ClientVersion)
	}
	if res.DeviceName != expected.DeviceName {
		t.Errorf("incorrect DeviceName %q != expected %q", res.DeviceName, expected.DeviceName)
	}
}

func TestOldHelloMsgs(t *testing.T) {
	// Tests that we can correctly identify old/missing/unknown hello
	// messages.

	cases := []struct {
		msg string
		err error
	}{
		{"00010001", ErrTooOldVersion}, // v12
		{"9F79BC40", ErrTooOldVersion}, // v13
		{"12345678", ErrUnknownMagic},
	}

	for _, tc := range cases {
		msg, _ := hex.DecodeString(tc.msg)

		outBuf := new(bytes.Buffer)
		outBuf.Write(msg)

		inBuf := new(bytes.Buffer)

		conn := &readWriter{outBuf, inBuf}

		send := Hello{
			DeviceName:    "this device",
			ClientName:    "other client",
			ClientVersion: "v1.0.0",
			Timestamp:     1234567890,
		}

		_, err := ExchangeHello(conn, send)
		if err != tc.err {
			t.Errorf("unexpected error %v != %v", err, tc.err)
		}
	}
}

func TestVersionStringParsing(t *testing.T) {
	// Tests for the isV2Client function with various version string formats

	testCases := []struct {
		version  string
		expected bool
		desc     string
	}{
		{"v2.0", true, "exact v2.0 version"},
		{"2.0", true, "exact 2.0 version"},
		{"v2.0-beta", true, "v2.0 beta version"},
		{"v2.0-rc.1", true, "v2.0 release candidate"},
		{"syncthing v2.0", true, "v2.0 with prefix"},
		{"v2.1", true, "v2.1 version"},
		{"2.1", true, "2.1 version"},
		{"v1.30", false, "v1.30 version"},
		{"1.30", false, "1.30 version"},
		{"v1.2.3", false, "v1.2.3 version"},
		{"syncthing v1.2.3", false, "v1.2.3 with prefix"},
		{"v2", false, "just v2"},
		{"2", false, "just 2"},
		{"", false, "empty string"},
		{"unknown", false, "unknown version"},
	}

	for _, tc := range testCases {
		result := isV2Client(tc.version)
		if result != tc.expected {
			t.Errorf("isV2Client(%q) = %v, expected %v - %s", tc.version, result, tc.expected, tc.desc)
		}
	}
}

func TestHelloMagicSelection(t *testing.T) {
	// Tests that the correct magic number is selected based on version string

	testCases := []struct {
		version  string
		expected uint32
		desc     string
	}{
		{"v2.0", HelloMessageV2Magic, "v2.0 should use v2 magic"},
		{"2.0", HelloMessageV2Magic, "2.0 should use v2 magic"},
		{"v2.0-beta", HelloMessageV2Magic, "v2.0-beta should use v2 magic"},
		{"v1.30", HelloMessageMagic, "v1.30 should use default magic"},
		{"1.30", HelloMessageMagic, "1.30 should use default magic"},
		{"", HelloMessageMagic, "empty string should use default magic"},
	}

	for _, tc := range testCases {
		hello := Hello{
			ClientVersion: tc.version,
		}
		result := hello.Magic()
		if result != tc.expected {
			t.Errorf("Hello{ClientVersion: %q}.Magic() = 0x%08X, expected 0x%08X - %s", tc.version, result, tc.expected, tc.desc)
		}
	}
}

type readWriter struct {
	r io.Reader
	w io.Writer
}

func (rw *readWriter) Write(data []byte) (int, error) {
	return rw.w.Write(data)
}

func (rw *readWriter) Read(data []byte) (int, error) {
	return rw.r.Read(data)
}

// TestV2DeviceConnection tests the connection between two v2.0 devices
func TestV2DeviceConnection(t *testing.T) {
	// Tests that we can successfully establish a connection between two v2.0 devices

	deviceA := Hello{
		DeviceName:    "Device A",
		ClientName:    "syncthing",
		ClientVersion: "v2.0",
		Timestamp:     1234567890, // Add timestamp
	}

	deviceB := Hello{
		DeviceName:    "Device B",
		ClientName:    "syncthing",
		ClientVersion: "v2.0",
		Timestamp:     1234567891, // Add timestamp
	}

	// Simulate bidirectional connection
	// Device A sends hello to Device B
	msgBufA, err := proto.Marshal(deviceA.toWire())
	if err != nil {
		t.Fatal(err)
	}

	hdrBufA := make([]byte, 6)
	binary.BigEndian.PutUint32(hdrBufA, deviceA.Magic())
	binary.BigEndian.PutUint16(hdrBufA[4:], uint16(len(msgBufA)))

	outBufA := new(bytes.Buffer)
	outBufA.Write(hdrBufA)
	outBufA.Write(msgBufA)

	inBufB := new(bytes.Buffer)
	connAtoB := &readWriter{outBufA, inBufB}

	// Device B receives hello from Device A and responds
	receivedA, err := ExchangeHello(connAtoB, deviceB)
	if err != nil {
		t.Fatalf("Device B failed to receive hello from Device A: %v", err)
	}

	if receivedA.ClientVersion != deviceA.ClientVersion {
		t.Errorf("Device B received incorrect ClientVersion %q != expected %q", receivedA.ClientVersion, deviceA.ClientVersion)
	}

	// Device B sends hello response to Device A
	msgBufB, err := proto.Marshal(deviceB.toWire())
	if err != nil {
		t.Fatal(err)
	}

	hdrBufB := make([]byte, 6)
	binary.BigEndian.PutUint32(hdrBufB, deviceB.Magic())
	binary.BigEndian.PutUint16(hdrBufB[4:], uint16(len(msgBufB)))

	outBufB := new(bytes.Buffer)
	outBufB.Write(hdrBufB)
	outBufB.Write(msgBufB)

	inBufA := new(bytes.Buffer)
	connBtoA := &readWriter{outBufB, inBufA}

	// Device A receives hello response from Device B
	receivedB, err := ExchangeHello(connBtoA, deviceA)
	if err != nil {
		t.Fatalf("Device A failed to receive hello response from Device B: %v", err)
	}

	if receivedB.ClientVersion != deviceB.ClientVersion {
		t.Errorf("Device A received incorrect ClientVersion %q != expected %q", receivedB.ClientVersion, deviceB.ClientVersion)
	}

	// Verify both devices are using the correct magic number
	if deviceA.Magic() != HelloMessageV2Magic {
		t.Errorf("Device A should use v2 magic number, got 0x%08X", deviceA.Magic())
	}

	if deviceB.Magic() != HelloMessageV2Magic {
		t.Errorf("Device B should use v2 magic number, got 0x%08X", deviceB.Magic())
	}
}

// TestV2V1Connection tests the connection between v2.0 and v1.30 devices
func TestV2V1Connection(t *testing.T) {
	// Tests that we can successfully establish a connection between v2.0 and v1.30 devices

	deviceV2 := Hello{
		DeviceName:    "Device V2",
		ClientName:    "syncthing",
		ClientVersion: "v2.0",
		Timestamp:     1234567892, // Add timestamp
	}

	deviceV1 := Hello{
		DeviceName:    "Device V1",
		ClientName:    "syncthing",
		ClientVersion: "v1.30",
		Timestamp:     1234567893, // Add timestamp
	}

	// Device V2 sends hello to Device V1
	msgBufV2, err := proto.Marshal(deviceV2.toWire())
	if err != nil {
		t.Fatal(err)
	}

	hdrBufV2 := make([]byte, 6)
	binary.BigEndian.PutUint32(hdrBufV2, deviceV2.Magic())
	binary.BigEndian.PutUint16(hdrBufV2[4:], uint16(len(msgBufV2)))

	outBufV2 := new(bytes.Buffer)
	outBufV2.Write(hdrBufV2)
	outBufV2.Write(msgBufV2)

	inBufV1 := new(bytes.Buffer)
	connV2toV1 := &readWriter{outBufV2, inBufV1}

	// Device V1 receives hello from Device V2 and responds
	receivedV2, err := ExchangeHello(connV2toV1, deviceV1)
	if err != nil {
		t.Fatalf("Device V1 failed to receive hello from Device V2: %v", err)
	}

	if receivedV2.ClientVersion != deviceV2.ClientVersion {
		t.Errorf("Device V1 received incorrect ClientVersion %q != expected %q", receivedV2.ClientVersion, deviceV2.ClientVersion)
	}

	// Device V1 sends hello response to Device V2
	msgBufV1, err := proto.Marshal(deviceV1.toWire())
	if err != nil {
		t.Fatal(err)
	}

	hdrBufV1 := make([]byte, 6)
	binary.BigEndian.PutUint32(hdrBufV1, deviceV1.Magic())
	binary.BigEndian.PutUint16(hdrBufV1[4:], uint16(len(msgBufV1)))

	outBufV1 := new(bytes.Buffer)
	outBufV1.Write(hdrBufV1)
	outBufV1.Write(msgBufV1)

	inBufV2 := new(bytes.Buffer)
	connV1toV2 := &readWriter{outBufV1, inBufV2}

	// Device V2 receives hello response from Device V1
	receivedV1, err := ExchangeHello(connV1toV2, deviceV2)
	if err != nil {
		t.Fatalf("Device V2 failed to receive hello response from Device V1: %v", err)
	}

	if receivedV1.ClientVersion != deviceV1.ClientVersion {
		t.Errorf("Device V2 received incorrect ClientVersion %q != expected %q", receivedV1.ClientVersion, deviceV1.ClientVersion)
	}

	// Verify magic number selection
	if deviceV2.Magic() != HelloMessageV2Magic {
		t.Errorf("Device V2 should use v2 magic number, got 0x%08X", deviceV2.Magic())
	}

	if deviceV1.Magic() != HelloMessageMagic {
		t.Errorf("Device V1 should use default magic number, got 0x%08X", deviceV1.Magic())
	}
}
