package com.syncthing.android.data.api.model

data class Ignores(
    val ignore: List<String>,
    val expanded: List<String>,
    val error: String?
)