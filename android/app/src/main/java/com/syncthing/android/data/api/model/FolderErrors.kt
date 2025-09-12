package com.syncthing.android.data.api.model

data class FolderError(
    val path: String,
    val error: String,
    val timestamp: String // ISO 8601 timestamp
)

data class FolderErrorsResponse(
    val errors: List<FolderError>,
    val page: Int,
    val perpage: Int,
    val total: Int
)