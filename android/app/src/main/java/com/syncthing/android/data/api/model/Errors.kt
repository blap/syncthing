package com.syncthing.android.data.api.model

data class Errors(
    val errors: List<ErrorInfo>
)

data class ErrorInfo(
    val timestamp: String, // ISO 8601 timestamp
    val message: String
)