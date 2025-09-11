package com.syncthing.android.data.api.model

data class CompletionInfo(
    val completion: Double,
    val globalBytes: Long,
    val needBytes: Long,
    val globalItems: Long,
    val needItems: Long,
    val needDeletes: Long,
    val sequence: Long
)