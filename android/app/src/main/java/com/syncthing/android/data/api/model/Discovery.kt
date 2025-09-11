package com.syncthing.android.data.api.model

data class DiscoveryEntry(
    val addresses: List<String>,
    val direct: Boolean,
    val error: String?,
    val when: String // ISO 8601 timestamp
)

data class DiscoveryResponse(
    val discoveries: Map<String, DiscoveryEntry>
)