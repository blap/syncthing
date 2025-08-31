// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections_test

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections"
	"github.com/syncthing/syncthing/lib/protocol"
)

func TestDebugStableNetwork(t *testing.T) {
	cfg := config.Wrap("/tmp/test-config.xml", config.New(protocol.EmptyDeviceID), protocol.EmptyDeviceID, nil)
	hm := connections.NewHealthMonitor(cfg, "device1")
	
	// Test with good network conditions
	hm.RecordLatency(20 * time.Millisecond)
	t.Logf("After good latency (20ms): Health score = %f, Interval = %v", hm.GetHealthScore(), hm.GetInterval())
	
	hm.RecordPacketLoss(0.0)
	t.Logf("After no packet loss: Health score = %f, Interval = %v", hm.GetHealthScore(), hm.GetInterval())
	
	// Test with multiple good measurements
	for i := 0; i < 5; i++ {
		hm.RecordLatency(20 * time.Millisecond)
		hm.RecordPacketLoss(0.0)
		t.Logf("After %d good measurements: Health score = %f, Interval = %v", i+1, hm.GetHealthScore(), hm.GetInterval())
	}
}