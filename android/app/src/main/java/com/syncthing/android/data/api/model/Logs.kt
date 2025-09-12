package com.syncthing.android.data.api.model

data class LogEntries(
    val timestamp: String, // ISO 8601 timestamp
    val message: String,
    val level: String
)

data class LogResponse(
    val entries: List<LogEntries>
)