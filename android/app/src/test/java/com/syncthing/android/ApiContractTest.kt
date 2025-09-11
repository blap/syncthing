package com.syncthing.android

import com.syncthing.android.util.ApiConstants
import org.junit.Test
import org.junit.Assert.*

/**
 * API Contract tests to verify compatibility between Android app and desktop Syncthing
 */
class ApiContractTest {
    
    /**
     * Test that all required API endpoints are defined in ApiConstants
     */
    @Test
    fun testApiEndpointsExist() {
        // System endpoints
        assertNotNull("SYSTEM_STATUS_ENDPOINT should not be null", ApiConstants.SYSTEM_STATUS_ENDPOINT)
        assertNotNull("SYSTEM_CONFIG_ENDPOINT should not be null", ApiConstants.SYSTEM_CONFIG_ENDPOINT)
        assertNotNull("SYSTEM_CONNECTIONS_ENDPOINT should not be null", ApiConstants.SYSTEM_CONNECTIONS_ENDPOINT)
        assertNotNull("SYSTEM_SHUTDOWN_ENDPOINT should not be null", ApiConstants.SYSTEM_SHUTDOWN_ENDPOINT)
        assertNotNull("SYSTEM_RESTART_ENDPOINT should not be null", ApiConstants.SYSTEM_RESTART_ENDPOINT)
        assertNotNull("SYSTEM_VERSION_ENDPOINT should not be null", ApiConstants.SYSTEM_VERSION_ENDPOINT)
        
        // Database endpoints
        assertNotNull("DB_STATUS_ENDPOINT should not be null", ApiConstants.DB_STATUS_ENDPOINT)
        assertNotNull("DB_BROWSE_ENDPOINT should not be null", ApiConstants.DB_BROWSE_ENDPOINT)
        assertNotNull("DB_NEED_ENDPOINT should not be null", ApiConstants.DB_NEED_ENDPOINT)
        
        // Statistics endpoints
        assertNotNull("STATS_DEVICE_ENDPOINT should not be null", ApiConstants.STATS_DEVICE_ENDPOINT)
        assertNotNull("STATS_FOLDER_ENDPOINT should not be null", ApiConstants.STATS_FOLDER_ENDPOINT)
        
        // Configuration endpoints
        assertNotNull("CONFIG_FOLDERS_ENDPOINT should not be null", ApiConstants.CONFIG_FOLDERS_ENDPOINT)
        assertNotNull("CONFIG_DEVICES_ENDPOINT should not be null", ApiConstants.CONFIG_DEVICES_ENDPOINT)
        assertNotNull("CONFIG_OPTIONS_ENDPOINT should not be null", ApiConstants.CONFIG_OPTIONS_ENDPOINT)
        
        // Events endpoint
        assertNotNull("EVENTS_ENDPOINT should not be null", ApiConstants.EVENTS_ENDPOINT)
    }
    
    /**
     * Test that API endpoint values match expected patterns
     */
    @Test
    fun testApiEndpointPatterns() {
        // All endpoints should start with "/rest/"
        assertTrue("SYSTEM_STATUS_ENDPOINT should start with /rest/", 
            ApiConstants.SYSTEM_STATUS_ENDPOINT.startsWith("/rest/"))
        assertTrue("SYSTEM_CONFIG_ENDPOINT should start with /rest/", 
            ApiConstants.SYSTEM_CONFIG_ENDPOINT.startsWith("/rest/"))
        assertTrue("DB_STATUS_ENDPOINT should start with /rest/", 
            ApiConstants.DB_STATUS_ENDPOINT.startsWith("/rest/"))
        assertTrue("EVENTS_ENDPOINT should start with /rest/", 
            ApiConstants.EVENTS_ENDPOINT.startsWith("/rest/"))
    }
    
    /**
     * Test that port constants are correctly defined
     */
    @Test
    fun testPortConstants() {
        assertTrue("DEFAULT_GUI_PORT should be positive", ApiConstants.DEFAULT_GUI_PORT > 0)
        assertTrue("DEFAULT_SYNC_PORT should be positive", ApiConstants.DEFAULT_SYNC_PORT > 0)
        assertTrue("DEFAULT_DISCOVERY_PORT should be positive", ApiConstants.DEFAULT_DISCOVERY_PORT > 0)
    }
    
    /**
     * Test that header constants are correctly defined
     */
    @Test
    fun testHeaderConstants() {
        assertNotNull("API_KEY_HEADER should not be null", ApiConstants.API_KEY_HEADER)
        assertNotNull("CONTENT_TYPE_HEADER should not be null", ApiConstants.CONTENT_TYPE_HEADER)
        assertNotNull("JSON_CONTENT_TYPE should not be null", ApiConstants.JSON_CONTENT_TYPE)
        assertTrue("API_KEY_HEADER should not be empty", ApiConstants.API_KEY_HEADER.isNotEmpty())
        assertTrue("CONTENT_TYPE_HEADER should not be empty", ApiConstants.CONTENT_TYPE_HEADER.isNotEmpty())
        assertTrue("JSON_CONTENT_TYPE should not be empty", ApiConstants.JSON_CONTENT_TYPE.isNotEmpty())
    }
    
    /**
     * Test that connection state constants are correctly defined
     */
    @Test
    fun testConnectionStateConstants() {
        assertNotNull("CONNECTION_STATE_CONNECTED should not be null", ApiConstants.CONNECTION_STATE_CONNECTED)
        assertNotNull("CONNECTION_STATE_DISCONNECTED should not be null", ApiConstants.CONNECTION_STATE_DISCONNECTED)
        assertNotNull("CONNECTION_STATE_PAUSED should not be null", ApiConstants.CONNECTION_STATE_PAUSED)
    }
    
    /**
     * Test that API version constants are correctly defined
     */
    @Test
    fun testApiVersionConstants() {
        assertNotNull("API_VERSION should not be null", ApiConstants.API_VERSION)
        assertNotNull("API_VERSION_HEADER should not be null", ApiConstants.API_VERSION_HEADER)
        assertTrue("API_VERSION should not be empty", ApiConstants.API_VERSION.isNotEmpty())
        assertTrue("API_VERSION_HEADER should not be empty", ApiConstants.API_VERSION_HEADER.isNotEmpty())
    }
}