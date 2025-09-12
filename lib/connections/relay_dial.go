// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/dialer"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/relay/client"
)

func init() {
	dialers["relay"] = relayDialerFactory{}
}

type relayDialer struct {
	commonDialer
}

func (d *relayDialer) Dial(ctx context.Context, id protocol.DeviceID, uri *url.URL) (internalConn, error) {
	inv, err := client.GetInvitationFromRelay(ctx, uri, id, d.tlsCfg.Certificates, 10*time.Second)
	if err != nil {
		// Record connection failure for health monitoring
		if globalService != nil {
			globalService.healthMonitor.RecordConnectionError(id, uri.Host, err)
		}
		return internalConn{}, err
	}

	conn, err := client.JoinSession(ctx, inv)
	if err != nil {
		// Record connection failure for health monitoring
		if globalService != nil {
			globalService.healthMonitor.RecordConnectionError(id, uri.Host, err)
		}
		return internalConn{}, err
	}

	err = dialer.SetTCPOptions(conn)
	if err != nil {
		conn.Close()
		// Record connection failure for health monitoring
		if globalService != nil {
			globalService.healthMonitor.RecordConnectionError(id, uri.Host, err)
		}
		return internalConn{}, err
	}

	err = dialer.SetTrafficClass(conn, d.trafficClass)
	if err != nil {
		l.Debugln("Dial (BEP/relay): setting traffic class:", err)
	}

	var tc *tls.Conn
	if inv.ServerSocket {
		tc = tls.Server(conn, d.tlsCfg)
	} else {
		tc = tls.Client(conn, d.tlsCfg)
	}

	// Get progressive dial timeout based on connection history
	timeout := getProgressiveDialTimeoutForAddress(uri.Host)
	_ = conn.SetDeadline(time.Now().Add(timeout))
	
	// Use global adaptive timeouts since we don't have access to service instance here
	err = tlsTimedHandshake(tc)
	
	// Record connection success or failure
	if err == nil {
		recordConnectionSuccessForAddress(uri.Host)
		// Record connection success for health monitoring
		if globalService != nil {
			globalService.healthMonitor.RecordConnectionSuccess(id, uri.Host)
		}
	} else {
		recordConnectionFailureForAddress(uri.Host)
		// Record connection failure for health monitoring
		if globalService != nil {
			globalService.healthMonitor.RecordConnectionError(id, uri.Host, err)
		}
	}
	
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		tc.Close()
		return internalConn{}, err
	}

	return newInternalConn(tc, connTypeRelayClient, false, d.wanPriority), nil
}

func (d *relayDialer) Priority(_ string) int {
	return d.wanPriority
}

type relayDialerFactory struct{}

func (relayDialerFactory) New(opts config.OptionsConfiguration, tlsCfg *tls.Config, _ *registry.Registry, _ *lanChecker) genericDialer {
	return &relayDialer{commonDialer{
		trafficClass:      opts.TrafficClass,
		reconnectInterval: time.Duration(opts.RelayReconnectIntervalM) * time.Minute,
		tlsCfg:            tlsCfg,
		wanPriority:       opts.ConnectionPriorityRelay,
		lanPriority:       opts.ConnectionPriorityRelay,
	}}
}

func (relayDialerFactory) AlwaysWAN() bool {
	return true
}

func (relayDialerFactory) Valid(cfg config.Configuration) error {
	if !cfg.Options.RelaysEnabled {
		return errDisabled
	}
	return nil
}

func (relayDialerFactory) String() string {
	return "Relay Dialer"
}