package com.syncthing.android.data.api.model

data class LocalChangedResult(
    val files: List<FileInfo>,
    val page: Int,
    val perpage: Int,
    val total: Int
)