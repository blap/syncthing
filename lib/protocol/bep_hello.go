// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"strconv"
	"regexp"

	"google.golang.org/protobuf/proto"

	"github.com/syncthing/syncthing/internal/gen/bep"
)

const (
	HelloMessageMagic   uint32 = 0x2EA7D90B
	HelloMessageV2Magic uint32 = 0x2EA7D90C // v2
	Version13HelloMagic uint32 = 0x9F79BC40 // old
)

var (
	// ErrTooOldVersion is returned by ExchangeHello when the other side
	// speaks an older, incompatible version of the protocol.
	ErrTooOldVersion = errors.New("the remote device speaks an older version of the protocol not compatible with this version")
	// ErrUnknownMagic is returned by ExchangeHello when the other side
	// speaks something entirely unknown.
	ErrUnknownMagic = errors.New("the remote device speaks an unknown (newer?) version of the protocol")
)

type Hello struct {
	DeviceName     string
	ClientName     string
	ClientVersion  string
	NumConnections int
	Timestamp      int64
}

func (h *Hello) toWire() *bep.Hello {
	return &bep.Hello{
		DeviceName:     h.DeviceName,
		ClientName:     h.ClientName,
		ClientVersion:  h.ClientVersion,
		NumConnections: int32(h.NumConnections),
		Timestamp:      h.Timestamp,
	}
}

func helloFromWire(w *bep.Hello) Hello {
	return Hello{
		DeviceName:     w.DeviceName,
		ClientName:     w.ClientName,
		ClientVersion:  w.ClientVersion,
		NumConnections: int(w.NumConnections),
		Timestamp:      w.Timestamp,
	}
}

func (h Hello) Magic() uint32 {
	// Check if this is a v2 client based on version string
	// v2 clients will have version strings that indicate v2 compatibility
	clientVersion := h.ClientVersion
	
	// Enhanced logging to capture detailed version information
	slog.Debug("Determining magic number for Hello message", 
		"clientVersion", clientVersion,
		"containsV2Dot", strings.Contains(strings.ToLower(clientVersion), "v2."),
		"contains2Dot", strings.Contains(strings.ToLower(clientVersion), "2."))
		
	// More robust version detection logic
	if isV2Client(clientVersion) {
		slog.Debug("Using v2 magic for Hello message", 
			"clientVersion", clientVersion)
		return HelloMessageV2Magic
	}
	
	// For now, we'll use the default magic, but this could be extended
	// to support version-specific magic numbers
	slog.Debug("Using default magic for Hello message", "clientVersion", clientVersion)
	return HelloMessageMagic
}

func ExchangeHello(c io.ReadWriter, h Hello) (Hello, error) {
	if h.Timestamp == 0 {
		panic("bug: missing timestamp in outgoing hello")
	}
	
	outgoingMagic := h.Magic()
	slog.Debug("Exchanging Hello messages", 
		"outgoingClientName", h.ClientName,
		"outgoingClientVersion", h.ClientVersion,
		"outgoingMagic", outgoingMagic,
		"outgoingMagicHex", fmt.Sprintf("0x%08X", outgoingMagic))
	
	if err := writeHello(c, h); err != nil {
		slog.Debug("Failed to write Hello message", "error", err)
		return Hello{}, err
	}
	
	incoming, err := readHello(c)
	if err != nil {
		slog.Debug("Failed to read Hello message", "error", err)
		// Add more context about what we sent
		slog.Debug("Context: failed while reading after sending Hello",
			"sentMagic", outgoingMagic,
			"sentClientName", h.ClientName,
			"sentClientVersion", h.ClientVersion)
		return Hello{}, err
	}
	
	incomingMagic := incoming.Magic()
	slog.Debug("Successfully exchanged Hello messages",
		"incomingClientName", incoming.ClientName,
		"incomingClientVersion", incoming.ClientVersion,
		"incomingNumConnections", incoming.NumConnections,
		"incomingMagic", incomingMagic,
		"incomingMagicHex", fmt.Sprintf("0x%08X", incomingMagic),
		"magicMatch", outgoingMagic == incomingMagic)
	
	// Use enhanced feature negotiation for mixed-version environments
	bestProtocol, features, negotiationErr := NegotiateFeaturesForMixedVersions(h, incoming)
	if negotiationErr != nil {
		slog.Warn("Enhanced feature negotiation issue, falling back to basic negotiation", "error", negotiationErr)
		// Fall back to previous method
		bestProtocol, negotiationErr = negotiateV2Protocol(h, incoming)
		if negotiationErr != nil {
			slog.Warn("Protocol negotiation issue", "error", negotiationErr)
			bestProtocol = NegotiateBestProtocol(h, incoming)
		}
		features = DetectV2Features(h, incoming)
	} else {
		slog.Debug("Successfully negotiated features for mixed versions", 
			"protocol", bestProtocol,
			"multipath", features.MultipathConnections,
			"compression", features.EnhancedCompression,
			"indexing", features.ImprovedIndexing)
	}
	
	slog.Debug("Negotiated best protocol", "protocol", bestProtocol)
	
	return incoming, nil
}

// NegotiateBestProtocol negotiates the best protocol to use based on client versions
func NegotiateBestProtocol(localHello, remoteHello Hello) string {
	// Use our new compatibility functions
	return NegotiateProtocol("", localHello.ClientVersion, remoteHello.ClientVersion)
}

// IsV2Client determines if a client version string indicates v2 compatibility
// This is the exported version of isV2Client
func IsV2Client(version string) bool {
	return isV2Client(version)
}

// isV2Client determines if a client version string indicates v2 compatibility
func isV2Client(version string) bool {
	// Handle various version string formats that indicate v2 compatibility
	// Common patterns: "v2.0", "2.0", "v2.0-beta", "v2.0-rc.1", etc.
	
	// Defensive check for empty version string
	if version == "" {
		slog.Debug("Empty version string, not a v2 client")
		return false
	}
	
	// Convert to lowercase for case-insensitive comparison and trim whitespace
	versionLower := strings.ToLower(strings.TrimSpace(version))
	
	// Log the version we're checking
	slog.Debug("Checking if client is v2 compatible", "version", versionLower)
	
	// Enhanced semantic version parsing
	major, _, _, err := parseSemVer(versionLower)
	if err == nil {
		if major >= 2 {
			slog.Debug("Client is v2 compatible (semantic version)", "version", versionLower, "major", major)
			return true
		}
		return false
	}
	
	// Check for exact v2 patterns at the beginning of the string
	// This prevents false positives like "syncthing v1.2.3" matching "2."
	if strings.HasPrefix(versionLower, "v2.") || strings.HasPrefix(versionLower, "2.") {
		slog.Debug("Client is v2 compatible (prefix match)", "version", versionLower)
		return true
	}
	
	// Check for v2 patterns anywhere in the string (for cases like "syncthing v2.0")
	v2Patterns := []string{"v2.", "v2-", "v2_"}
	for _, pattern := range v2Patterns {
		if strings.Contains(versionLower, pattern) {
			slog.Debug("Client is v2 compatible (contains pattern)", "version", versionLower, "pattern", pattern)
			return true
		}
	}
	
	// Special case for exact "v2.0" or "2.0" versions
	if versionLower == "v2.0" || versionLower == "2.0" {
		slog.Debug("Client is v2 compatible (exact match)", "version", versionLower)
		return true
	}
	
	// Additional check for semantic versioning patterns like "2.0.0"
	if strings.HasPrefix(versionLower, "v2.0.") || strings.HasPrefix(versionLower, "2.0.") {
		slog.Debug("Client is v2 compatible (semantic version match)", "version", versionLower)
		return true
	}
	
	// If none of the above patterns match, it's not a v2 client
	slog.Debug("Client is not v2 compatible", "version", versionLower)
	return false
}

// parseSemVer attempts to parse a semantic version string
func parseSemVer(version string) (major, minor, patch int, err error) {
	// Remove leading 'v' if present
	version = strings.TrimPrefix(version, "v")
	
	// Regular expression for semantic versioning
	semVerRegex := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:[-+].*)?$`)
	matches := semVerRegex.FindStringSubmatch(version)
	
	if matches == nil {
		return 0, 0, 0, fmt.Errorf("not a valid semantic version: %s", version)
	}
	
	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, 0, err
	}
	
	minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, 0, err
	}
	
	// Patch version is optional
	if matches[3] != "" {
		patch, err = strconv.Atoi(matches[3])
		if err != nil {
			return 0, 0, 0, err
		}
	}
	
	return major, minor, patch, nil
}

// IsVersionMismatch returns true if the error is a reliable indication of a
// version mismatch that we might want to alert the user about.
func IsVersionMismatch(err error) bool {
	return errors.Is(err, ErrTooOldVersion) || errors.Is(err, ErrUnknownMagic)
}

func readHello(c io.Reader) (Hello, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(c, header); err != nil {
		slog.Debug("Failed to read Hello header", "error", err)
		return Hello{}, err
	}

	magic := binary.BigEndian.Uint32(header)
	slog.Debug("Received Hello with magic", "magic", magic)

	switch magic {
	case HelloMessageMagic, HelloMessageV2Magic:
		// This is a v0.14 or v2 Hello message in proto format
		if _, err := io.ReadFull(c, header[:2]); err != nil {
			slog.Debug("Failed to read Hello message size", "error", err)
			return Hello{}, err
		}
		msgSize := binary.BigEndian.Uint16(header[:2])
		if msgSize > 32767 {
			err := errors.New("hello message too big")
			slog.Debug("Hello message too big", "size", msgSize, "error", err)
			return Hello{}, err
		}
		buf := make([]byte, msgSize)
		if _, err := io.ReadFull(c, buf); err != nil {
			slog.Debug("Failed to read Hello message body", "error", err)
			return Hello{}, err
		}

		var wh bep.Hello
		if err := proto.Unmarshal(buf, &wh); err != nil {
			slog.Debug("Failed to unmarshal Hello message", "error", err)
			return Hello{}, err
		}

		hello := helloFromWire(&wh)
		slog.Debug("Successfully read Hello message", 
			"clientName", hello.ClientName,
			"clientVersion", hello.ClientVersion,
			"magicUsed", magic,
			"willUseMagicForReply", hello.Magic(),
			"willUseMagicHex", fmt.Sprintf("0x%08X", hello.Magic()))
		return hello, nil

	case 0x00010001, 0x00010000, Version13HelloMagic:
		// This is the first word of an older cluster config message or an
		// old magic number. (Version 0, message ID 1, message type 0,
		// compression enabled or disabled)
		slog.Debug("Received old version Hello message", "magic", magic)
		return Hello{}, ErrTooOldVersion
	}

	slog.Debug("Received unknown Hello magic", "magic", magic)
	return Hello{}, ErrUnknownMagic
}

func writeHello(c io.Writer, h Hello) error {
	msg, err := proto.Marshal(h.toWire())
	if err != nil {
		slog.Debug("Failed to marshal Hello message", "error", err)
		return err
	}
	if len(msg) > 32767 {
		// The header length must be a positive signed int16
		panic("bug: attempting to serialize too large hello message")
	}

	header := make([]byte, 6, 6+len(msg))
	magic := h.Magic()
	binary.BigEndian.PutUint32(header[:4], magic)
	binary.BigEndian.PutUint16(header[4:], uint16(len(msg)))

	slog.Debug("Writing Hello message", 
		"magic", magic,
		"clientName", h.ClientName,
		"clientVersion", h.ClientVersion,
		"messageSize", len(msg))

	_, err = c.Write(append(header, msg...))
	if err != nil {
		slog.Debug("Failed to write Hello message to connection", "error", err)
	}
	return err
}