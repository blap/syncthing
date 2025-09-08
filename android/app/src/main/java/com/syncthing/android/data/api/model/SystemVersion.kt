package com.syncthing.android.data.api.model

data class SystemVersion(
    val version: String,
    val codename: String,
    val longVersion: String,
    val extra: String,
    val os: String,
    val arch: String,
    val isBeta: Boolean,
    val isCandidate: Boolean,
    val isRelease: Boolean,
    val date: String,
    val tags: List<String>,
    val stamp: String,
    val user: String,
    val container: String
)