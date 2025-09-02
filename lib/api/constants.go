// Package api provides constants shared between desktop and mobile versions
package api

const (
	// System endpoints
	SystemStatusEndpoint   = "/rest/system/status"
	SystemConfigEndpoint   = "/rest/system/config"
	SystemConnectionsEndpoint = "/rest/system/connections"
	SystemShutdownEndpoint = "/rest/system/shutdown"
	SystemRestartEndpoint  = "/rest/system/restart"
	
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
)