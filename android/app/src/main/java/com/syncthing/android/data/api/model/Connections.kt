package com.syncthing.android.data.api.model

data class Connections(
    val total: ConnectionInfo,
    val connections: Map<String, ConnectionInfo>
)

data class ConnectionInfo(
    val at: String, // ISO 8601 timestamp
    val inBytesTotal: Long,
    val outBytesTotal: Long,
    val connected: Boolean,
    val paused: Boolean,
    val clientVersion: String?
)