package com.syncthing.android.data.repository

import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.mockito.Mock
import org.mockito.Mockito.`when`
import org.mockito.MockitoAnnotations

/**
 * Unit tests for SyncthingRepository
 * 
 * Tests the Repository's interaction with the API service and its handling of system status data.
 * Uses JUnit 4 for testing framework and Mockito for mocking the API service dependency.
 */
class SyncthingRepositoryTest {
    
    // Mock dependencies
    @Mock
    private lateinit var apiService: SyncthingApiServiceInterface
    
    // Class under test
    private lateinit var repository: SyncthingRepository
    
    // For closing Mockito mocks after tests
    private lateinit var closeable: AutoCloseable
    
    /**
     * Set up test environment before each test
     * - Initialize Mockito mocks
     * - Create SyncthingRepository instance with mock API service
     */
    @Before
    fun setUp() {
        closeable = MockitoAnnotations.openMocks(this)
        repository = SyncthingRepository(apiService)
    }
    
    /**
     * Test that the Repository correctly fetches system status from the API service
     * Uses runBlocking to test suspend functions in a synchronous manner
     */
    @Test
    fun shouldFetchSystemStatusFromApi() = runBlocking {
        // Given
        val expectedStatus = SystemStatus(
            alloc = 12345678L,
            cpuPercent = 12.5,
            discoveryEnabled = true,
            discoveryErrors = emptyMap(),
            discoveryMethods = 3,
            goroutines = 42,
            myID = "ABC123-DEF456",
            pathSeparator = "/",
            startTime = "2023-01-01T00:00:00Z",
            sys = 23456789L,
            tilde = "~",
            uptime = 3600
        )
        
        `when`(apiService.getSystemStatus("test-api-key"))
            .thenReturn(expectedStatus)
        
        // When
        val result = repository.getSystemStatus("test-api-key")
        
        // Then
        assertEquals(expectedStatus, result)
    }
}