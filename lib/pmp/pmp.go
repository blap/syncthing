// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package pmp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	natpmp "github.com/jackpal/go-nat-pmp"

	"github.com/syncthing/syncthing/lib/nat"
	"github.com/syncthing/syncthing/lib/netutil"
	"github.com/syncthing/syncthing/lib/osutil"
	"github.com/syncthing/syncthing/lib/svcutil"
)

var (
	// Add retry configuration
	maxRetries = 3
	retryDelay = time.Second
)

func init() {
	nat.Register(Discover)
}

func Discover(ctx context.Context, renewal, timeout time.Duration) []nat.Device {
	var ip net.IP
	err := svcutil.CallWithContext(ctx, func() error {
		var err error
		ip, err = netutil.Gateway()
		return err
	})
	if err != nil {
		slog.Debug("Failed to discover gateway", "error", err)
		return nil
	}
	if ip == nil || ip.IsUnspecified() {
		return nil
	}

	slog.Debug("Discovered gateway at", "ip", ip)

	c := natpmp.NewClientWithTimeout(ip, timeout)
	// Try contacting the gateway, if it does not respond, assume it does not
	// speak NAT-PMP.

	// Add retry mechanism for external address retrieval
	var ierr error
	for i := 0; i < maxRetries; i++ {
		ierr = svcutil.CallWithContext(ctx, func() error {
			_, err := c.GetExternalAddress()
			return err
		})
		if ierr == nil {
			break
		}

		// Log retry attempt
		slog.Debug("Failed to get external address from NAT-PMP gateway, retrying",
			"attempt", i+1,
			"maxRetries", maxRetries,
			"error", ierr)

		// Wait before retrying (exponential backoff)
		select {
		case <-time.After(retryDelay * time.Duration(1<<uint(i))):
		case <-ctx.Done():
			return nil
		}
	}

	if ierr != nil {
		if errors.Is(ierr, context.Canceled) {
			return nil
		}
		if strings.Contains(ierr.Error(), "Timed out") {
			slog.Debug("Timeout trying to get external address, assume no NAT-PMP available")
			return nil
		}
		// Log the error but continue - some routers might still support port mapping
		slog.Warn("Failed to get external address from NAT-PMP gateway",
			"error", ierr,
			"gateway", ip)
	}

	var localIP net.IP
	// Port comes from the natpmp package
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(timeoutCtx, "udp", net.JoinHostPort(ip.String(), "5351"))
	if err == nil {
		conn.Close()
		localIP, err = osutil.IPFromAddr(conn.LocalAddr())
		if localIP == nil {
			slog.Debug("Failed to lookup local IP", "error", err)
		}
	}

	return []nat.Device{&wrapper{
		renewal:   renewal,
		localIP:   localIP,
		gatewayIP: ip,
		client:    c,
	}}
}

type wrapper struct {
	renewal   time.Duration
	localIP   net.IP
	gatewayIP net.IP
	client    *natpmp.Client
}

func (w *wrapper) ID() string {
	return fmt.Sprintf("NAT-PMP@%s", w.gatewayIP.String())
}

func (w *wrapper) GetLocalIPv4Address() net.IP {
	return w.localIP
}

func (w *wrapper) AddPortMapping(ctx context.Context, protocol nat.Protocol, internalPort, externalPort int, _ string, duration time.Duration) (int, error) {
	// NAT-PMP says that if duration is 0, the mapping is actually removed
	// Swap the zero with the renewal value, which should make the lease for the
	// exact amount of time between the calls.
	if duration == 0 {
		duration = w.renewal
	}

	// Add retry mechanism for port mapping
	var result *natpmp.AddPortMappingResult
	var err error

	for i := 0; i < maxRetries; i++ {
		err = svcutil.CallWithContext(ctx, func() error {
			var ierr error
			result, ierr = w.client.AddPortMapping(strings.ToLower(string(protocol)), internalPort, externalPort, int(duration/time.Second))
			return ierr
		})
		if err == nil {
			break
		}

		// Log retry attempt
		slog.Debug("Failed to add port mapping via NAT-PMP, retrying",
			"attempt", i+1,
			"maxRetries", maxRetries,
			"protocol", protocol,
			"internalPort", internalPort,
			"externalPort", externalPort,
			"error", err)

		// Check for specific errors that shouldn't be retried
		if strings.Contains(err.Error(), "connection refused") {
			slog.Warn("Connection refused when trying to add NAT-PMP port mapping",
				"gateway", w.gatewayIP,
				"error", err)
			// Don't retry connection refused errors
			break
		}

		// Wait before retrying (exponential backoff)
		select {
		case <-time.After(retryDelay * time.Duration(1<<uint(i))):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

	port := 0
	if result != nil {
		port = int(result.MappedExternalPort)
	}

	if err != nil {
		slog.Warn("Failed to add port mapping via NAT-PMP after retries",
			"gateway", w.gatewayIP,
			"protocol", protocol,
			"internalPort", internalPort,
			"externalPort", externalPort,
			"error", err)
	}

	return port, err
}

func (*wrapper) AddPinhole(_ context.Context, _ nat.Protocol, _ nat.Address, _ time.Duration) ([]net.IP, error) {
	// NAT-PMP doesn't support pinholes.
	return nil, errors.New("adding IPv6 pinholes is unsupported on NAT-PMP")
}

func (*wrapper) SupportsIPVersion(version nat.IPVersion) bool {
	// NAT-PMP gateways should always try to create port mappings and not pinholes
	// since NAT-PMP doesn't support IPv6.
	return version == nat.IPvAny || version == nat.IPv4Only
}

func (w *wrapper) GetExternalIPv4Address(ctx context.Context) (net.IP, error) {
	// Add retry mechanism for external address retrieval
	var result *natpmp.GetExternalAddressResult
	var err error

	for i := 0; i < maxRetries; i++ {
		err = svcutil.CallWithContext(ctx, func() error {
			var ierr error
			result, ierr = w.client.GetExternalAddress()
			return ierr
		})
		if err == nil {
			break
		}

		// Log retry attempt
		slog.Debug("Failed to get external address via NAT-PMP, retrying",
			"attempt", i+1,
			"maxRetries", maxRetries,
			"error", err)

		// Wait before retrying (exponential backoff)
		select {
		case <-time.After(retryDelay * time.Duration(1<<uint(i))):
		case <-ctx.Done():
			return net.IPv4zero, ctx.Err()
		}
	}

	ip := net.IPv4zero
	if result != nil {
		ip = net.IPv4(
			result.ExternalIPAddress[0],
			result.ExternalIPAddress[1],
			result.ExternalIPAddress[2],
			result.ExternalIPAddress[3],
		)
	}

	if err != nil {
		slog.Warn("Failed to get external address via NAT-PMP after retries",
			"gateway", w.gatewayIP,
			"error", err)
	}

	return ip, err
}
