package com.syncthing.android.util

/**
 * Shared constants between desktop and Android versions of Syncthing
 * These are automatically generated from lib/api/constants.go
 * DO NOT EDIT MANUALLY - Run 'go run script/generate-android-constants.go' to regenerate
 */
object ApiConstants {
    const val SYSTEM_STATUS_ENDPOINT = "/rest/system/status"
    const val SYSTEM_CONFIG_ENDPOINT = "/rest/system/config"
    const val SYSTEM_CONNECTIONS_ENDPOINT = "/rest/system/connections"
    const val SYSTEM_SHUTDOWN_ENDPOINT = "/rest/system/shutdown"
    const val SYSTEM_RESTART_ENDPOINT = "/rest/system/restart"
    const val SYSTEM_VERSION_ENDPOINT = "/rest/system/version"
    const val DB_STATUS_ENDPOINT = "/rest/db/status"
    const val DB_BROWSE_ENDPOINT = "/rest/db/browse"
    const val DB_NEED_ENDPOINT = "/rest/db/need"
    const val STATS_DEVICE_ENDPOINT = "/rest/stats/device"
    const val STATS_FOLDER_ENDPOINT = "/rest/stats/folder"
    const val CONFIG_FOLDERS_ENDPOINT = "/rest/config/folders"
    const val CONFIG_DEVICES_ENDPOINT = "/rest/config/devices"
    const val CONFIG_OPTIONS_ENDPOINT = "/rest/config/options"
    const val EVENTS_ENDPOINT = "/rest/events"
    const val DEFAULT_GUI_PORT = 8384
    const val DEFAULT_SYNC_PORT = 22000
    const val DEFAULT_DISCOVERY_PORT = 21027
    const val API_KEY_HEADER = "X-API-Key"
    const val CONTENT_TYPE_HEADER = "Content-Type"
    const val JSON_CONTENT_TYPE = "application/json"
    const val CONNECTION_STATE_CONNECTED = "connected"
    const val CONNECTION_STATE_DISCONNECTED = "disconnected"
    const val CONNECTION_STATE_PAUSED = "paused"
    const val API_VERSION = "1.0.0"
    const val API_VERSION_HEADER = "X-API-Version"
}
