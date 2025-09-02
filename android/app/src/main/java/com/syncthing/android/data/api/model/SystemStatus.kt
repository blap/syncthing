package com.syncthing.android.data.api.model

data class SystemStatus(
    val alloc: Long,
    val cpuPercent: Double,
    val discoveryEnabled: Boolean,
    val discoveryErrors: Map<String, String>,
    val discoveryMethods: Int,
    val goroutines: Int,
    val myID: String,
    val pathSeparator: String,
    val startTime: String,
    val sys: Long,
    val tilde: String,
    val uptime: Int
)