// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

func TestDebugHealthScore(t *testing.T) {
	cfg := config.Wrap("/tmp/test-config.xml", config.New(protocol.EmptyDeviceID), protocol.EmptyDeviceID, nil)
	hm := NewHealthMonitorWithConfig(cfg, "device1")

	// Test with very bad network conditions
	hm.RecordLatency(600 * time.Millisecond)
	t.Logf("After bad latency (600ms): Health score = %f, Interval = %v", hm.GetHealthScore(), hm.GetInterval())

	hm.RecordPacketLoss(20.0)
	t.Logf("After packet loss (20%%): Health score = %f, Interval = %v", hm.GetHealthScore(), hm.GetInterval())

	// Test with multiple bad measurements
	for i := 0; i < 5; i++ {
		hm.RecordLatency(600 * time.Millisecond)
		hm.RecordPacketLoss(20.0)
		t.Logf("After %d bad measurements: Health score = %f, Interval = %v", i+1, hm.GetHealthScore(), hm.GetInterval())
	}
}
