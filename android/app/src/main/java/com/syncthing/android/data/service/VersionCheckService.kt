package com.syncthing.android.data.service

import android.content.Context
import android.content.pm.PackageManager
import com.syncthing.android.data.api.model.SystemVersion
import com.syncthing.android.util.VersionCompatibilityChecker

class VersionCheckService(private val context: Context) {
    
    data class VersionCompatibilityResult(
        val isCompatible: Boolean,
        val desktopVersion: String,
        val androidVersion: String,
        val needsUpdate: Boolean,
        val updateMessage: String?
    )
    
    /**
     * Check version compatibility between Android app and desktop Syncthing
     */
    suspend fun checkVersionCompatibility(
        desktopVersion: SystemVersion
    ): VersionCompatibilityResult {
        val androidVersion = getAppVersion()
        
        // Use the more sophisticated compatibility checker
        val result = VersionCompatibilityChecker.checkCompatibility(androidVersion, desktopVersion)
        
        return VersionCompatibilityResult(
            isCompatible = result.isCompatible,
            desktopVersion = result.desktopVersion,
            androidVersion = result.androidVersion,
            needsUpdate = result.needsUpdate,
            updateMessage = result.updateMessage
        )
    }
    
    /**
     * Get the current Android app version
     */
    private fun getAppVersion(): String {
        return try {
            val packageInfo = context.packageManager.getPackageInfo(context.packageName, 0)
            packageInfo.versionName ?: "unknown"
        } catch (e: PackageManager.NameNotFoundException) {
            "unknown"
        }
    }
}