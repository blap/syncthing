package com.syncthing.android.data.api.model

data class DatabaseStatus(
    val id: String,
    val globalBytes: Long,
    val globalFiles: Long,
    val localBytes: Long,
    val localFiles: Long,
    val needBytes: Long,
    val needFiles: Long,
    val ignorePatterns: Boolean,
    val state: String,
    val stateChanged: String, // ISO 8601 timestamp
    val version: Long,
    val sequence: Long,
    val error: String?
)