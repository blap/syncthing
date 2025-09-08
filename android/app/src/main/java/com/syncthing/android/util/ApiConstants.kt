package com.syncthing.android.util

/**
 * Shared constants between desktop and Android versions of Syncthing
 * These should be kept in sync with lib/api/constants.go in the desktop version
 */
object ApiConstants {
    // System endpoints
    const val SYSTEM_STATUS_ENDPOINT = "/rest/system/status"
    const val SYSTEM_CONFIG_ENDPOINT = "/rest/system/config"
    const val SYSTEM_CONNECTIONS_ENDPOINT = "/rest/system/connections"
    const val SYSTEM_SHUTDOWN_ENDPOINT = "/rest/system/shutdown"
    const val SYSTEM_RESTART_ENDPOINT = "/rest/system/restart"
    const val SYSTEM_VERSION_ENDPOINT = "/rest/system/version"
    const val SYSTEM_UPGRADE_ENDPOINT = "/rest/system/upgrade"
    
    // Database endpoints
    const val DB_STATUS_ENDPOINT = "/rest/db/status"
    const val DB_BROWSE_ENDPOINT = "/rest/db/browse"
    const val DB_NEED_ENDPOINT = "/rest/db/need"
    
    // Statistics endpoints
    const val STATS_DEVICE_ENDPOINT = "/rest/stats/device"
    const val STATS_FOLDER_ENDPOINT = "/rest/stats/folder"
    
    // Configuration endpoints
    const val CONFIG_FOLDERS_ENDPOINT = "/rest/config/folders"
    const val CONFIG_DEVICES_ENDPOINT = "/rest/config/devices"
    const val CONFIG_OPTIONS_ENDPOINT = "/rest/config/options"
    
    // Events endpoint
    const val EVENTS_ENDPOINT = "/rest/events"
    
    // Default ports
    const val DEFAULT_GUI_PORT = 8384
    const val DEFAULT_SYNC_PORT = 22000
    const val DEFAULT_DISCOVERY_PORT = 21027
    
    // Headers
    const val API_KEY_HEADER = "X-API-Key"
    const val CONTENT_TYPE_HEADER = "Content-Type"
    const val JSON_CONTENT_TYPE = "application/json"
    
    // Connection states
    const val CONNECTION_STATE_CONNECTED = "connected"
    const val CONNECTION_STATE_DISCONNECTED = "disconnected"
    const val CONNECTION_STATE_PAUSED = "paused"
}