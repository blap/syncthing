package com.syncthing.android.util

import com.syncthing.android.util.VersionCompatibilityChecker.VersionComponents

/**
 * Feature compatibility matrix that tracks feature availability across Syncthing versions
 * This helps determine which features are available based on the desktop version
 */
object FeatureCompatibilityMatrix {
    
    /**
     * Data class representing feature availability information
     */
    data class FeatureInfo(
        val featureName: String,
        val androidVersion: String, // Minimum Android version required
        val desktopVersion: String, // Minimum desktop version required
        val description: String
    )
    
    /**
     * Map of features and their compatibility requirements
     */
    private val featureMatrix = mapOf(
        "basic_sync" to FeatureInfo(
            "Basic Synchronization",
            "1.0.0",
            "1.0.0",
            "Core file synchronization functionality"
        ),
        "versioning" to FeatureInfo(
            "File Versioning",
            "1.0.0",
            "1.0.0",
            "Basic file versioning support"
        ),
        "advanced_ignore" to FeatureInfo(
            "Advanced Ignore Patterns",
            "1.2.0",
            "1.2.0",
            "Support for advanced ignore patterns and .stignore syntax"
        ),
        "external_versioning" to FeatureInfo(
            "External Versioning",
            "1.1.0",
            "1.1.0",
            "Support for external versioning scripts"
        ),
        "custom_discovery" to FeatureInfo(
            "Custom Discovery Servers",
            "1.0.0",
            "1.0.0",
            "Support for custom discovery servers"
        ),
        "custom_relay" to FeatureInfo(
            "Custom Relay Servers",
            "1.0.0",
            "1.0.0",
            "Support for custom relay servers"
        ),
        "bandwidth_limits" to FeatureInfo(
            "Bandwidth Limiting",
            "1.1.0",
            "1.1.0",
            "Support for bandwidth rate limiting"
        ),
        "folder_pause" to FeatureInfo(
            "Folder Pausing",
            "1.0.0",
            "1.0.0",
            "Ability to pause individual folders"
        ),
        "device_pause" to FeatureInfo(
            "Device Pausing",
            "1.0.0",
            "1.0.0",
            "Ability to pause individual devices"
        ),
        "rescan_scheduling" to FeatureInfo(
            "Rescan Scheduling",
            "1.1.0",
            "1.1.0",
            "Support for custom folder rescan scheduling"
        )
    )
    
    /**
     * Check if a feature is supported based on versions
     */
    fun isFeatureSupported(feature: String, androidVersion: String, desktopVersion: String): Boolean {
        val featureInfo = featureMatrix[feature] ?: return true // Assume supported if not in matrix
        
        val androidVer = VersionCompatibilityChecker.parseVersion(androidVersion)
        val desktopVer = VersionCompatibilityChecker.parseVersion(desktopVersion)
        val requiredAndroidVer = VersionCompatibilityChecker.parseVersion(featureInfo.androidVersion)
        val requiredDesktopVer = VersionCompatibilityChecker.parseVersion(featureInfo.desktopVersion)
        
        // Feature is supported if both Android and desktop versions meet requirements
        return VersionCompatibilityChecker.compareVersions(androidVer, requiredAndroidVer) >= 0 && 
               VersionCompatibilityChecker.compareVersions(desktopVer, requiredDesktopVer) >= 0
    }
    
    /**
     * Get all features supported by the given versions
     */
    fun getSupportedFeatures(androidVersion: String, desktopVersion: String): List<String> {
        return featureMatrix.filter { entry ->
            isFeatureSupported(entry.key, androidVersion, desktopVersion)
        }.keys.toList()
    }
    
    /**
     * Get all features that are NOT supported by the given versions
     */
    fun getUnsupportedFeatures(androidVersion: String, desktopVersion: String): List<String> {
        return featureMatrix.filter { entry ->
            !isFeatureSupported(entry.key, androidVersion, desktopVersion)
        }.keys.toList()
    }
    
    /**
     * Get detailed information about a specific feature
     */
    fun getFeatureInfo(feature: String): FeatureInfo? {
        return featureMatrix[feature]
    }
    
    /**
     * Get all features with their compatibility information
     */
    fun getAllFeatures(): Map<String, FeatureInfo> {
        return featureMatrix
    }
}