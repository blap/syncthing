package com.syncthing.android.viewmodel

import androidx.arch.core.executor.testing.InstantTaskExecutorRule
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.mockito.Mock
import org.mockito.Mockito.`when`
import org.mockito.MockitoAnnotations
import android.app.Application
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull

/**
 * Unit tests for MainViewModel
 * 
 * Tests the ViewModel's interaction with the SyncthingRepository and its handling of system status data.
 * Uses JUnit 4 for testing framework, Mockito for mocking dependencies, and Kotlin coroutines test utilities
 * for testing coroutine-based operations.
 */
@ExperimentalCoroutinesApi
class MainViewModelTest {
    
    // Rule to instantly execute LiveData operations on the main thread
    @get:Rule
    val instantExecutorRule = InstantTaskExecutorRule()
    
    // Mock dependencies
    @Mock
    private lateinit var repository: SyncthingRepository
    
    @Mock
    private lateinit var application: Application
    
    // Class under test
    private lateinit var viewModel: MainViewModel
    
    // For closing Mockito mocks after tests
    private lateinit var closeable: AutoCloseable
    
    // Test dispatcher for controlling coroutine execution in tests
    private val testDispatcher = StandardTestDispatcher()
    
    /**
     * Set up test environment before each test
     * - Initialize Mockito mocks
     * - Set main dispatcher to test dispatcher
     * - Create MainViewModel instance with mock dependencies
     */
    @Before
    fun setUp() {
        closeable = MockitoAnnotations.openMocks(this)
        Dispatchers.setMain(testDispatcher)
        viewModel = MainViewModel(repository, application)
    }
    
    /**
     * Clean up test environment after each test
     * - Reset main dispatcher to original
     * - Close Mockito mocks
     */
    @After
    fun tearDown() {
        Dispatchers.resetMain()
        closeable.close()
    }
    
    /**
     * Test that the ViewModel initializes with a null system status
     */
    @Test
    fun shouldInitializeWithNullSystemStatus() {
        // Then
        assertNull(viewModel.systemStatus.value)
    }
    
    /**
     * Test that the ViewModel correctly updates system status when fetched from repository
     * Uses runTest with a StandardTestDispatcher to control coroutine execution
     */
    @Test
    fun shouldUpdateSystemStatusWhenFetched() = runTest(testDispatcher) {
        // Given
        val systemStatus = SystemStatus(
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
        
        `when`(repository.getSystemStatus("test-api-key"))
            .thenReturn(systemStatus)
        
        // When
        viewModel.fetchSystemStatus("test-api-key")
        advanceUntilIdle()
        
        // Then
        assertEquals(systemStatus, viewModel.systemStatus.value)
    }
}