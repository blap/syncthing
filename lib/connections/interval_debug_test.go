// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"testing"
)

func TestDebugIntervals(t *testing.T) {
	cfg := createTestConfig()
	hm := NewHealthMonitor(cfg, "device1")

	t.Logf("Health Score -> Interval mapping:")
	testScores := []float64{100.0, 90.0, 80.0, 70.0, 60.0, 50.0, 40.0, 30.0, 20.0, 10.0, 0.0}

	for _, score := range testScores {
		hm.SetHealthScore(score)
		interval := hm.GetInterval()
		t.Logf("  %6.1f -> %v", score, interval)
	}
}
