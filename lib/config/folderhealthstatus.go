// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"time"
)

// FolderHealthStatus represents the health status of a folder
type FolderHealthStatus struct {
	FolderID      string        `json:"folderID"`
	Healthy       bool          `json:"healthy"`
	Issues        []string      `json:"issues"`
	LastChecked   time.Time     `json:"lastChecked"`
	CheckTime     time.Time     `json:"checkTime"`
	CheckDuration time.Duration `json:"checkDuration"`
}