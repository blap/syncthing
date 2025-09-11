package com.syncthing.android.data.api.model

import com.syncthing.android.data.api.model.FileInfo

data class DatabaseBrowseResult(
    val files: List<FileInfo>
)