package com.syncthing.android

import com.syncthing.android.util.VersionCompatibilityChecker
import com.syncthing.android.data.api.model.SystemVersion
import org.junit.Test
import org.junit.Assert.*

/**
 * Version compatibility tests
 */
class VersionCompatibilityTest {
    
    /**
     * Test version parsing
     */
    @Test
    fun testVersionParsing() {
        val version1 = VersionCompatibilityChecker.parseVersion("1.2.3")
        assertEquals("Major version should be 1", 1, version1.major)
        assertEquals("Minor version should be 2", 2, version1.minor)
        assertEquals("Patch version should be 3", 3, version1.patch)
        
        val version2 = VersionCompatibilityChecker.parseVersion("2.0")
        assertEquals("Major version should be 2", 2, version2.major)
        assertEquals("Minor version should be 0", 0, version2.minor)
        assertEquals("Patch version should be 0", 0, version2.patch)
    }
    
    /**
     * Test version comparison
     */
    @Test
    fun testVersionComparison() {
        val version1 = VersionCompatibilityChecker.VersionComponents(1, 2, 3)
        val version2 = VersionCompatibilityChecker.VersionComponents(1, 2, 3)
        val version3 = VersionCompatibilityChecker.VersionComponents(1, 2, 4)
        // Removed unused variable version4
        val version5 = VersionCompatibilityChecker.VersionComponents(2, 0, 0)
        
        assertEquals("Versions should be equal", 0, VersionCompatibilityChecker.compareVersions(version1, version2))
        assertEquals("Version1 should be less than version3", -1, VersionCompatibilityChecker.compareVersions(version1, version3))
        assertEquals("Version3 should be greater than version1", 1, VersionCompatibilityChecker.compareVersions(version3, version1))
        assertEquals("Version1 should be less than version5", -1, VersionCompatibilityChecker.compareVersions(version1, version5))
    }
    
    /**
     * Test compatibility checking with same versions
     */
    @Test
    fun testCompatibilityWithSameVersions() {
        val desktopVersion = SystemVersion(
            version = "1.2.3",
            codename = "Fermium",
            longVersion = "1.2.3",
            extra = "",
            os = "linux",
            arch = "amd64",
            isBeta = false,
            isCandidate = false,
            isRelease = true,
            date = "2025-01-01",
            tags = listOf(),
            stamp = "1234567890",
            user = "user",
            container = "docker"
        )
        
        val result = VersionCompatibilityChecker.checkCompatibility("1.2.3", desktopVersion)
        assertTrue("Same versions should be compatible", result.isCompatible)
        assertFalse("Same versions should not need update", result.needsUpdate)
        assertNull("Same versions should not have update message", result.updateMessage)
    }
    
    /**
     * Test compatibility checking with newer desktop version
     */
    @Test
    fun testCompatibilityWithNewerDesktop() {
        val desktopVersion = SystemVersion(
            version = "1.3.0",
            codename = "Gadolinium",
            longVersion = "1.3.0",
            extra = "",
            os = "linux",
            arch = "amd64",
            isBeta = false,
            isCandidate = false,
            isRelease = true,
            date = "2025-06-01",
            tags = listOf(),
            stamp = "1234567890",
            user = "user",
            container = "docker"
        )
        
        val result = VersionCompatibilityChecker.checkCompatibility("1.2.3", desktopVersion)
        assertTrue("Compatible versions should be compatible", result.isCompatible)
        assertTrue("Newer desktop version should recommend update", result.needsUpdate)
        assertNotNull("Update message should not be null", result.updateMessage)
    }
    
    /**
     * Test compatibility checking with major version mismatch
     */
    @Test
    fun testCompatibilityWithMajorMismatch() {
        val desktopVersion = SystemVersion(
            version = "2.0.0",
            codename = "Hydrogen",
            longVersion = "2.0.0",
            extra = "",
            os = "linux",
            arch = "amd64",
            isBeta = false,
            isCandidate = false,
            isRelease = true,
            date = "2025-06-01",
            tags = listOf(),
            stamp = "1234567890",
            user = "user",
            container = "docker"
        )
        
        val result = VersionCompatibilityChecker.checkCompatibility("1.2.3", desktopVersion)
        assertFalse("Major version mismatch should be incompatible", result.isCompatible)
        assertNotNull("Incompatible versions should have update message", result.updateMessage)
    }
    
    /**
     * Test feature support checking
     */
    @Test
    fun testFeatureSupport() {
        val desktopVersion = SystemVersion(
            version = "1.2.0",
            codename = "Fermium",
            longVersion = "1.2.0",
            extra = "",
            os = "linux",
            arch = "amd64",
            isBeta = false,
            isCandidate = false,
            isRelease = true,
            date = "2025-01-01",
            tags = listOf(),
            stamp = "1234567890",
            user = "user",
            container = "docker"
        )
        
        // Assuming "advanced_ignore" requires version 1.2.0
        assertTrue("Feature should be supported with matching version", 
            VersionCompatibilityChecker.isFeatureSupported("advanced_ignore", desktopVersion))
    }
}