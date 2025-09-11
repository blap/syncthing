package com.syncthing.android.data.api.model

data class FileVersion(
    val versionTime: String, // ISO 8601 timestamp
    val size: Long,
    val modTime: String, // ISO 8601 timestamp
    val deleted: Boolean
)

data class FolderVersions(
    val versions: Map<String, List<FileVersion>>
)