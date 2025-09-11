package com.syncthing.android

import com.syncthing.android.util.ApiConstants
import org.junit.Test
import org.junit.Assert.*

/**
 * Test to verify that API constants are properly generated from desktop constants
 * This test ensures that the generation script is working correctly
 */
class ApiConstantsGenerationTest {
    
    /**
     * Test that system endpoints match expected values from desktop constants
     */
    @Test
    fun testSystemEndpoints() {
        assertEquals("System status endpoint should match",
            "/rest/system/status", ApiConstants.SYSTEM_STATUS_ENDPOINT)
        assertEquals("System config endpoint should match",
            "/rest/system/config", ApiConstants.SYSTEM_CONFIG_ENDPOINT)
        assertEquals("System connections endpoint should match",
            "/rest/system/connections", ApiConstants.SYSTEM_CONNECTIONS_ENDPOINT)
        assertEquals("System shutdown endpoint should match",
            "/rest/system/shutdown", ApiConstants.SYSTEM_SHUTDOWN_ENDPOINT)
        assertEquals("System restart endpoint should match",
            "/rest/system/restart", ApiConstants.SYSTEM_RESTART_ENDPOINT)
        assertEquals("System version endpoint should match",
            "/rest/system/version", ApiConstants.SYSTEM_VERSION_ENDPOINT)
    }
    
    /**
     * Test that database endpoints match expected values
     */
    @Test
    fun testDatabaseEndpoints() {
        assertEquals("DB status endpoint should match",
            "/rest/db/status", ApiConstants.DB_STATUS_ENDPOINT)
        assertEquals("DB browse endpoint should match",
            "/rest/db/browse", ApiConstants.DB_BROWSE_ENDPOINT)
        assertEquals("DB need endpoint should match",
            "/rest/db/need", ApiConstants.DB_NEED_ENDPOINT)
    }
    
    /**
     * Test that statistics endpoints match expected values
     */
    @Test
    fun testStatisticsEndpoints() {
        assertEquals("Stats device endpoint should match",
            "/rest/stats/device", ApiConstants.STATS_DEVICE_ENDPOINT)
        assertEquals("Stats folder endpoint should match",
            "/rest/stats/folder", ApiConstants.STATS_FOLDER_ENDPOINT)
    }
    
    /**
     * Test that configuration endpoints match expected values
     */
    @Test
    fun testConfigurationEndpoints() {
        assertEquals("Config folders endpoint should match",
            "/rest/config/folders", ApiConstants.CONFIG_FOLDERS_ENDPOINT)
        assertEquals("Config devices endpoint should match",
            "/rest/config/devices", ApiConstants.CONFIG_DEVICES_ENDPOINT)
        assertEquals("Config options endpoint should match",
            "/rest/config/options", ApiConstants.CONFIG_OPTIONS_ENDPOINT)
    }
    
    /**
     * Test that events endpoint matches expected value
     */
    @Test
    fun testEventsEndpoint() {
        assertEquals("Events endpoint should match",
            "/rest/events", ApiConstants.EVENTS_ENDPOINT)
    }
    
    /**
     * Test that port constants match expected values
     */
    @Test
    fun testPortConstants() {
        assertEquals("Default GUI port should match", 8384, ApiConstants.DEFAULT_GUI_PORT)
        assertEquals("Default sync port should match", 22000, ApiConstants.DEFAULT_SYNC_PORT)
        assertEquals("Default discovery port should match", 21027, ApiConstants.DEFAULT_DISCOVERY_PORT)
    }
    
    /**
     * Test that header constants match expected values
     */
    @Test
    fun testHeaderConstants() {
        assertEquals("API key header should match", "X-API-Key", ApiConstants.API_KEY_HEADER)
        assertEquals("Content type header should match", "Content-Type", ApiConstants.CONTENT_TYPE_HEADER)
        assertEquals("JSON content type should match", "application/json", ApiConstants.JSON_CONTENT_TYPE)
    }
    
    /**
     * Test that connection state constants match expected values
     */
    @Test
    fun testConnectionStateConstants() {
        assertEquals("Connected state should match", "connected", ApiConstants.CONNECTION_STATE_CONNECTED)
        assertEquals("Disconnected state should match", "disconnected", ApiConstants.CONNECTION_STATE_DISCONNECTED)
        assertEquals("Paused state should match", "paused", ApiConstants.CONNECTION_STATE_PAUSED)
    }
}