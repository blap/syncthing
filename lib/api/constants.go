// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package api provides constants shared between desktop and mobile versions
package api

const (
	// System endpoints
	SystemStatusEndpoint   = "/rest/system/status"
	SystemConfigEndpoint   = "/rest/system/config"
	SystemConnectionsEndpoint = "/rest/system/connections"
	SystemShutdownEndpoint = "/rest/system/shutdown"
	SystemRestartEndpoint  = "/rest/system/restart"
	SystemVersionEndpoint  = "/rest/system/version"
	
	// Database endpoints
	DBStatusEndpoint       = "/rest/db/status"
	DBBrowseEndpoint       = "/rest/db/browse"
	DBNeedEndpoint         = "/rest/db/need"
	
	// Statistics endpoints
	StatsDeviceEndpoint    = "/rest/stats/device"
	StatsFolderEndpoint    = "/rest/stats/folder"
	
	// Configuration endpoints
	ConfigFoldersEndpoint  = "/rest/config/folders"
	ConfigDevicesEndpoint  = "/rest/config/devices"
	ConfigOptionsEndpoint  = "/rest/config/options"
	
	// Events endpoint
	EventsEndpoint         = "/rest/events"
	
	// Default ports
	DefaultGuiPort         = 8384
	DefaultSyncPort        = 22000
	DefaultDiscoveryPort   = 21027
	
	// Headers
	ApiKeyHeader           = "X-API-Key"
	ContentTypeHeader      = "Content-Type"
	JsonContentType        = "application/json"
	
	// Connection states
	ConnectionStateConnected    = "connected"
	ConnectionStateDisconnected = "disconnected"
	ConnectionStatePaused      = "paused"
	
	// API Versioning
	APIVersion             = "1.0.0"
	APIVersionHeader       = "X-API-Version"
)