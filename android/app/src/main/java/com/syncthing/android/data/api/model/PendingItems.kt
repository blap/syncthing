package com.syncthing.android.data.api.model

data class PendingDevice(
    val deviceID: String,
    val name: String,
    val time: String // ISO 8601 timestamp
)

data class PendingFolder(
    val id: String,
    val label: String,
    val time: String // ISO 8601 timestamp
)

data class PendingDevicesResponse(
    val devices: List<PendingDevice>
)

data class PendingFoldersResponse(
    val folders: List<PendingFolder>
)