// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/dialer"
	"github.com/syncthing/syncthing/lib/protocol"
)

func init() {
	dialers["tcp"] = &tcpDialerFactory{}
}

type tcpDialer struct {
	commonDialer
	registry *registry.Registry
}

func (d *tcpDialer) Dial(ctx context.Context, _ protocol.DeviceID, uri *url.URL) (internalConn, error) {
	uri = fixupPort(uri, config.DefaultTCPPort)

	tcaddr, err := net.ResolveTCPAddr(uri.Scheme, uri.Host)
	if err != nil {
		return internalConn{}, err
	}

	conn, err := dialer.DialContextReusePortFunc(d.registry)(ctx, uri.Scheme, tcaddr.String())
	if err != nil {
		return internalConn{}, err
	}

	var tc *tls.Conn
	if tc, err = d.setupTLS(conn, tcaddr); err != nil {
		conn.Close()
		return internalConn{}, err
	}

	priority := d.wanPriority
	isLocal := d.lanChecker.isLANHost(uri.Host)
	if isLocal {
		priority = d.lanPriority
	}

	return newInternalConn(tc, connTypeTCPClient, isLocal, priority), nil
}

func (d *tcpDialer) setupTLS(conn net.Conn, _ *net.TCPAddr) (*tls.Conn, error) {
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	tc := tls.Client(conn, d.tlsCfg)
	err := tlsTimedHandshake(tc)
	_ = conn.SetDeadline(time.Time{})
	return tc, err
}

type tcpDialerFactory struct{}

func (tcpDialerFactory) New(opts config.OptionsConfiguration, tlsCfg *tls.Config, registry *registry.Registry, lanChecker *lanChecker) genericDialer {
	return &tcpDialer{
		commonDialer: commonDialer{
			reconnectInterval: time.Duration(opts.ReconnectIntervalS) * time.Second,
			tlsCfg:            tlsCfg,
			lanChecker:        lanChecker,
			lanPriority:       opts.ConnectionPriorityTCPLAN,
			wanPriority:       opts.ConnectionPriorityTCPWAN,
		},
		registry: registry,
	}
}

func (tcpDialerFactory) Priority(host string, lanChecker *lanChecker) int {
	if lanChecker.isLANHost(host) {
		return lanChecker.cfg.Options().ConnectionPriorityTCPLAN
	}
	return lanChecker.cfg.Options().ConnectionPriorityTCPWAN
}

func (tcpDialerFactory) AlwaysWAN() bool {
	return false
}

func (tcpDialerFactory) Valid(config.Configuration) error {
	// Always valid
	return nil
}

func (tcpDialerFactory) String() string {
	return "tcp"
}