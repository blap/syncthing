// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"strings"

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
	if strings.Contains(h.ClientVersion, "v2.") || strings.Contains(h.ClientVersion, "2.") {
		slog.Debug("Using v2 magic for Hello message", "clientVersion", h.ClientVersion)
		return HelloMessageV2Magic
	}
	// For now, we'll use the default magic, but this could be extended
	// to support version-specific magic numbers
	slog.Debug("Using default magic for Hello message", "clientVersion", h.ClientVersion)
	return HelloMessageMagic
}

func ExchangeHello(c io.ReadWriter, h Hello) (Hello, error) {
	if h.Timestamp == 0 {
		panic("bug: missing timestamp in outgoing hello")
	}
	
	slog.Debug("Exchanging Hello messages", 
		"outgoingClientName", h.ClientName,
		"outgoingClientVersion", h.ClientVersion,
		"outgoingMagic", h.Magic())
	
	if err := writeHello(c, h); err != nil {
		slog.Debug("Failed to write Hello message", "error", err)
		return Hello{}, err
	}
	
	incoming, err := readHello(c)
	if err != nil {
		slog.Debug("Failed to read Hello message", "error", err)
		return Hello{}, err
	}
	
	slog.Debug("Successfully exchanged Hello messages",
		"incomingClientName", incoming.ClientName,
		"incomingClientVersion", incoming.ClientVersion,
		"incomingNumConnections", incoming.NumConnections)
	
	return incoming, nil
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
			"magicUsed", magic)
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