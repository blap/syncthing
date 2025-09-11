package com.syncthing.android.data.api.model

// This will be represented as a Map<String, String> in the API interface
data class PathInfo(
    val key: String,
    val path: String
)