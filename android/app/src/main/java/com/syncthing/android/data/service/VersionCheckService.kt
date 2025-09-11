package com.syncthing.android.data.service

import android.content.Context
import android.content.pm.PackageManager
import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.api.model.SystemVersion
import com.syncthing.android.util.VersionCompatibilityChecker
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory

class VersionCheckService(private val context: Context) {
    
    companion object {
        private const val DESKTOP_API_URL = "http://localhost:8384" // Default Syncthing GUI port
    }
    
    data class VersionCompatibilityResult(
        val isCompatible: Boolean,
        val desktopVersion: String,
        val androidVersion: String,
        val needsUpdate: Boolean,
        val updateMessage: String?,
        val desktopCodename: String,
        val isBeta: Boolean,
        val isCandidate: Boolean
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
            updateMessage = result.updateMessage,
            desktopCodename = result.desktopCodename,
            isBeta = result.isBeta,
            isCandidate = result.isCandidate
        )
    }
    
    /**
     * Fetch the desktop Syncthing version through the REST API using Retrofit
     */
    suspend fun fetchDesktopVersion(apiKey: String): SystemVersion {
        // Create Retrofit instance
        val retrofit = Retrofit.Builder()
            .baseUrl(DESKTOP_API_URL)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
        
        // Create API service
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        
        // Make the API call
        return apiService.getSystemVersion(apiKey)
    }
    
    /**
     * Get the current Android app version from package manager
     */
    private fun getAppVersion(): String {
        return try {
            val packageInfo = context.packageManager.getPackageInfo(context.packageName, 0)
            packageInfo.versionName ?: "1.0.0" // Default fallback
        } catch (e: PackageManager.NameNotFoundException) {
            "1.0.0" // Default fallback
        }
    }
    
    /**
     * Check if a specific feature is supported based on desktop version
     */
    fun isFeatureSupported(feature: String, desktopVersion: SystemVersion): Boolean {
        return VersionCompatibilityChecker.isFeatureSupported(feature, desktopVersion)
    }
}