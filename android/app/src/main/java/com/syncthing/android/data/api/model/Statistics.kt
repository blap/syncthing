package com.syncthing.android.data.api.model

data class DeviceStatistics(
    val lastSeen: String, // ISO 8601 timestamp
    val lastConnection: String, // ISO 8601 timestamp
    val inBytesTotal: Long,
    val outBytesTotal: Long
)

data class FolderStatistics(
    val lastFile: LastFile?,
    val inBytesTotal: Long,
    val outBytesTotal: Long,
    val state: String,
    val stateChanged: String, // ISO 8601 timestamp
    val pullErrors: Int,
    val needFiles: Int,
    val needDirectories: Int,
    val needSymlinks: Int,
    val needDeletes: Int,
    val needBytes: Long
)

data class LastFile(
    val at: String, // ISO 8601 timestamp
    val filename: String,
    val action: String
)