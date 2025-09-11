package com.syncthing.android.data.api.model

data class UpgradeInfo(
    val running: String,
    val latest: String,
    val newer: Boolean,
    val majorNewer: Boolean
)