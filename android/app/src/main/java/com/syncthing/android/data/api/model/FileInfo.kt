package com.syncthing.android.data.api.model

data class FileInfo(
    val name: String,
    val type: String, // "file" or "directory"
    val size: Long,
    val modTime: String, // ISO 8601 timestamp
    val version: VersionVector,
    val sequence: Long,
    val permissions: String?,
    val noPermissions: Boolean,
    val invalid: Boolean,
    val blockSize: Int
)

data class VersionVector(
    val counters: List<Counter>
)

data class Counter(
    val id: Int,
    val value: Long
)