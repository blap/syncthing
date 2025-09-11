package com.syncthing.android

import com.syncthing.android.util.FeatureCompatibilityMatrix
import org.junit.Test
import org.junit.Assert.*

/**
 * Feature compatibility matrix tests
 */
class FeatureCompatibilityTest {
    
    /**
     * Test that the feature matrix contains expected features
     */
    @Test
    fun testFeatureMatrixContainsExpectedFeatures() {
        val features = FeatureCompatibilityMatrix.getAllFeatures()
        
        assertTrue("Feature matrix should contain basic_sync", features.containsKey("basic_sync"))
        assertTrue("Feature matrix should contain versioning", features.containsKey("versioning"))
        assertTrue("Feature matrix should contain advanced_ignore", features.containsKey("advanced_ignore"))
        assertTrue("Feature matrix should contain external_versioning", features.containsKey("external_versioning"))
    }
    
    /**
     * Test feature support checking
     */
    @Test
    fun testFeatureSupport() {
        // Test that basic features are supported with early versions
        assertTrue("Basic sync should be supported with 1.0.0",
            FeatureCompatibilityMatrix.isFeatureSupported("basic_sync", "1.0.0", "1.0.0"))
        
        // Test that advanced features require newer versions
        assertFalse("Advanced ignore should not be supported with 1.0.0",
            FeatureCompatibilityMatrix.isFeatureSupported("advanced_ignore", "1.0.0", "1.0.0"))
        
        assertTrue("Advanced ignore should be supported with 1.2.0",
            FeatureCompatibilityMatrix.isFeatureSupported("advanced_ignore", "1.2.0", "1.2.0"))
    }
    
    /**
     * Test getting supported features
     */
    @Test
    fun testGetSupportedFeatures() {
        val supported = FeatureCompatibilityMatrix.getSupportedFeatures("1.0.0", "1.0.0")
        
        assertTrue("Basic sync should be supported", supported.contains("basic_sync"))
        assertTrue("Versioning should be supported", supported.contains("versioning"))
        assertFalse("Advanced ignore should not be supported", supported.contains("advanced_ignore"))
    }
    
    /**
     * Test getting unsupported features
     */
    @Test
    fun testGetUnsupportedFeatures() {
        val unsupported = FeatureCompatibilityMatrix.getUnsupportedFeatures("1.0.0", "1.0.0")
        
        assertTrue("Advanced ignore should be unsupported", unsupported.contains("advanced_ignore"))
        assertTrue("External versioning should be unsupported", unsupported.contains("external_versioning"))
    }
    
    /**
     * Test getting feature info
     */
    @Test
    fun testGetFeatureInfo() {
        val basicSyncInfo = FeatureCompatibilityMatrix.getFeatureInfo("basic_sync")
        assertNotNull("Basic sync info should not be null", basicSyncInfo)
        assertEquals("Feature name should match", "Basic Synchronization", basicSyncInfo?.featureName)
        assertEquals("Android version should be 1.0.0", "1.0.0", basicSyncInfo?.androidVersion)
        assertEquals("Desktop version should be 1.0.0", "1.0.0", basicSyncInfo?.desktopVersion)
    }
}