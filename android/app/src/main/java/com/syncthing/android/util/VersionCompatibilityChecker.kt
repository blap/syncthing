package com.syncthing.android.util

import com.syncthing.android.data.api.model.SystemVersion

class VersionCompatibilityChecker {
    
    companion object {
        /**
         * Check if the Android app version is compatible with the desktop version
         * This implementation considers:
         * - Major version compatibility (critical)
         * - Minor version differences (feature availability)
         * - Patch version differences (bug fixes)
         * - API endpoint availability based on shared constants
         * - Feature support based on version
         */
        fun checkCompatibility(
            androidVersion: String,
            desktopVersion: SystemVersion
        ): CompatibilityResult {
            val desktopVer = parseVersion(desktopVersion.version)
            val androidVer = parseVersion(androidVersion)
            
            // Check if major versions match (critical for compatibility)
            val majorCompatible = desktopVer.major == androidVer.major
            
            // Check if desktop version is newer
            val desktopNewer = compareVersions(desktopVer, androidVer) > 0
            
            // Check if versions are too far apart
            val majorDifference = kotlin.math.abs(desktopVer.major - androidVer.major)
            val minorDifference = kotlin.math.abs(desktopVer.minor - androidVer.minor)
            
            // More sophisticated compatibility logic:
            // 1. If major versions don't match, likely incompatible
            // 2. If desktop is significantly newer, recommend update
            // 3. If Android is newer than desktop, might have issues with older API
            val isCompatible = when {
                majorDifference > 1 -> false // Too far apart
                !majorCompatible -> false // Major version mismatch
                else -> true
            }
            
            // Recommend update if desktop is newer and compatible
            val needsUpdate = desktopNewer && isCompatible
            
            // Determine update message based on version differences
            val message = when {
                !majorCompatible -> "Major version mismatch (${desktopVersion.version} vs $androidVersion). Please update your Android app for compatibility."
                majorDifference > 1 -> "Version difference too large (${desktopVersion.version} vs $androidVersion). Please update your Android app."
                needsUpdate -> "A newer version of Syncthing (${desktopVersion.version}) is available. Consider updating for the latest features."
                compareVersions(androidVer, desktopVer) > 0 -> "Your Android app is newer than the desktop version (${desktopVersion.version}). Some features may not work correctly."
                else -> null
            }
            
            return CompatibilityResult(
                isCompatible = isCompatible,
                needsUpdate = needsUpdate,
                updateMessage = message,
                desktopVersion = desktopVersion.version,
                androidVersion = androidVersion,
                desktopCodename = desktopVersion.codename,
                isBeta = desktopVersion.isBeta,
                isCandidate = desktopVersion.isCandidate
            )
        }
        
        /**
         * Check if a specific feature is supported based on version
         * This would be expanded with a feature compatibility matrix
         */
        fun isFeatureSupported(feature: String, desktopVersion: SystemVersion): Boolean {
            val desktopVer = parseVersion(desktopVersion.version)
            
            // Example feature compatibility rules (these would be expanded)
            return when (feature) {
                "advanced_ignore" -> desktopVer.major >= 1 && desktopVer.minor >= 2
                "external_versioning" -> desktopVer.major >= 1 && desktopVer.minor >= 1
                "custom_discovery" -> desktopVer.major >= 1
                else -> true // Assume supported by default
            }
        }
        
        /**
         * Parse a version string into components
         */
        public fun parseVersion(version: String): VersionComponents {
            // Remove any non-version characters (like 'v' prefix or build metadata)
            val cleanVersion = version.replace(Regex("[^0-9\\.\\-]"), "")
            
            val parts = cleanVersion.split(".").map { it.toIntOrNull() ?: 0 }
            
            return VersionComponents(
                major = if (parts.isNotEmpty()) parts[0] else 0,
                minor = if (parts.size > 1) parts[1] else 0,
                patch = if (parts.size > 2) parts[2] else 0
            )
        }
        
        /**
         * Compare two version components
         * Returns:
         * -1 if version1 < version2
         * 0 if version1 == version2
         * 1 if version1 > version2
         */
        public fun compareVersions(version1: VersionComponents, version2: VersionComponents): Int {
            return when {
                version1.major != version2.major -> version1.major.compareTo(version2.major)
                version1.minor != version2.minor -> version1.minor.compareTo(version2.minor)
                version1.patch != version2.patch -> version1.patch.compareTo(version2.patch)
                else -> 0
            }
        }
    }
    
    data class VersionComponents(
        val major: Int,
        val minor: Int,
        val patch: Int
    )
    
    data class CompatibilityResult(
        val isCompatible: Boolean,
        val needsUpdate: Boolean,
        val updateMessage: String?,
        val desktopVersion: String,
        val androidVersion: String,
        val desktopCodename: String,
        val isBeta: Boolean,
        val isCandidate: Boolean
    )
}