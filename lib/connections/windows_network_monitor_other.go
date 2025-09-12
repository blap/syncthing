// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows

package connections

// newWindowsNetworkMonitor is a placeholder function for non-Windows platforms
// It returns nil since Windows network monitoring is not applicable on non-Windows platforms
func newWindowsNetworkMonitor(svc Service) interface {
	Start()
	Stop()
} {
	return nil
}