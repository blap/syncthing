package com.syncthing.android.data.api.model

data class FolderNeedResult(
    val progress: List<NeedProgressItem>,
    val queued: List<NeedQueuedItem>,
    val rest: List<NeedRestItem>,
    val total: Int,
    val page: Int,
    val perpage: Int
)

data class NeedProgressItem(
    val name: String,
    val type: String, // "file" or "directory"
    val completion: Double,
    val modTime: String // ISO 8601 timestamp
)

data class NeedQueuedItem(
    val name: String,
    val type: String, // "file" or "directory"
    val size: Long,
    val modTime: String // ISO 8601 timestamp
)

data class NeedRestItem(
    val name: String,
    val type: String, // "file" or "directory"
    val size: Long,
    val modTime: String // ISO 8601 timestamp
)