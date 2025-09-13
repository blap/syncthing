// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package beacon

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/syncthing/syncthing/lib/svcutil"
)

type recv struct {
	data []byte
	src  net.Addr
}

type Interface interface {
	suture.Service
	fmt.Stringer
	Send(data []byte)
	Recv() ([]byte, net.Addr)
	Error() error
}

type cast struct {
	*suture.Supervisor
	name    string
	reader  svcutil.ServiceWithError
	writer  svcutil.ServiceWithError
	outbox  chan recv
	inbox   chan []byte
	stopped chan struct{}
	err     error
	errMut  sync.Mutex
}

// newCast creates a base object for multi- or broadcasting. Afterwards the
// caller needs to set reader and writer with the addReader and addWriter
// methods to get a functional implementation of Interface.
func newCast(name string) *cast {
	// Only log restarts in debug mode.
	spec := svcutil.SpecWithDebugLogger()
	// Don't retry too frenetically: an error to open a socket or
	// whatever is usually something that is either permanent or takes
	// a while to get solved...
	spec.FailureThreshold = 2
	spec.FailureBackoff = 60 * time.Second
	c := &cast{
		Supervisor: suture.New(name, spec),
		name:       name,
		inbox:      make(chan []byte),
		outbox:     make(chan recv, 16),
		stopped:    make(chan struct{}),
	}
	svcutil.OnSupervisorDone(c.Supervisor, func() { close(c.stopped) })
	return c
}

func (c *cast) addReader(svc func(context.Context) error) {
	c.reader = svcutil.AsService(svc, fmt.Sprintf("%s/reader", c.name))
	c.Add(c.reader)
}

func (c *cast) addWriter(svc func(context.Context) error) {
	c.writer = svcutil.AsService(svc, fmt.Sprintf("%s/writer", c.name))
	c.Add(c.writer)
}

func (c *cast) Send(data []byte) {
	select {
	case c.inbox <- data:
	case <-c.stopped:
	}
}

func (c *cast) Recv() ([]byte, net.Addr) {
	select {
	case r := <-c.outbox:
		return r.data, r.src
	case <-c.stopped:
		return nil, nil
	}
}

func (c *cast) Error() error {
	c.errMut.Lock()
	defer c.errMut.Unlock()
	return c.err
}

func (c *cast) String() string {
	return c.name
}
