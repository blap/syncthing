package com.syncthing.android.data.repository

import com.syncthing.android.data.api.SyncthingApiService
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.mockito.Mock
import org.mockito.Mockito.*
import org.mockito.MockitoAnnotations

class SyncthingRepositoryTest {
    
    @Mock
    private lateinit var apiService: SyncthingApiService
    
    private lateinit var repository: SyncthingRepository
    
    @Before
    fun setUp() {
        MockitoAnnotations.openMocks(this)
        repository = SyncthingRepository(apiService)
    }
    
    @Test
    fun `should fetch system status from api`() = runBlocking {
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
        
        whenever(apiService.getSystemStatus("test-api-key"))
            .thenReturn(expectedStatus)
        
        // When
        val result = repository.getSystemStatus("test-api-key")
        
        // Then
        assertEquals(expectedStatus, result)
    }
}