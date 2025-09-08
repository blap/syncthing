package com.syncthing.android.data.api.model

data class FolderBrowseResult(
    val files: List<BrowsedFile>,
    val directories: List<BrowsedDirectory>
)

data class BrowsedFile(
    val name: String,
    val size: Long,
    val modified: String, // ISO 8601 timestamp
    val type: String, // "file"
    val permissions: String? = null
)

data class BrowsedDirectory(
    val name: String,
    val modified: String, // ISO 8601 timestamp
    val type: String, // "dir"
    val permissions: String? = null
)