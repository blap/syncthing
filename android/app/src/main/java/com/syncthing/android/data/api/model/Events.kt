package com.syncthing.android.data.api.model

data class Event(
    val id: Int,
    val `type`: String,
    val time: String, // ISO 8601 timestamp
    val data: Map<String, Any>
)