package com.syncthing.android.util

import com.syncthing.android.data.api.model.SystemVersion

class VersionCompatibilityChecker {
    
    companion object {
        /**
         * Check if the Android app version is compatible with the desktop version
         * This is a simplified implementation - in a real app, you might want to check:
         * - Major version compatibility
         * - API endpoint availability
         * - Feature support based on version
         */
        fun checkCompatibility(
            androidVersion: String,
            desktopVersion: SystemVersion
        ): CompatibilityResult {
            val desktopVer = parseVersion(desktopVersion.version)
            val androidVer = parseVersion(androidVersion)
            
            // Check if major versions match
            val majorCompatible = desktopVer.major == androidVer.major
            
            // Check if desktop version is newer
            val desktopNewer = compareVersions(desktopVer, androidVer) > 0
            
            // For this implementation, we'll consider compatible if major versions match
            // and the Android app is not significantly behind
            val isCompatible = majorCompatible
            
            // Recommend update if desktop is newer
            val needsUpdate = desktopNewer
            
            val message = when {
                !majorCompatible -> "Major version mismatch. Please update your Android app."
                needsUpdate -> "A newer version of Syncthing (${desktopVersion.version}) is available."
                else -> null
            }
            
            return CompatibilityResult(
                isCompatible = isCompatible,
                needsUpdate = needsUpdate,
                updateMessage = message,
                desktopVersion = desktopVersion.version,
                androidVersion = androidVersion
            )
        }
        
        /**
         * Parse a version string into components
         */
        private fun parseVersion(version: String): VersionComponents {
            // Remove any non-version characters (like 'v' prefix)
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
        private fun compareVersions(version1: VersionComponents, version2: VersionComponents): Int {
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
        val androidVersion: String
    )
}