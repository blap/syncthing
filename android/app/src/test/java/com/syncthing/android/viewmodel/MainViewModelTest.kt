package com.syncthing.android.viewmodel

import androidx.arch.core.executor.testing.InstantTaskExecutorRule
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.*
import org.junit.After
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.mockito.Mock
import org.mockito.Mockito.*
import org.mockito.MockitoAnnotations

@ExperimentalCoroutinesApi
class MainViewModelTest {
    
    @get:Rule
    val instantExecutorRule = InstantTaskExecutorRule()
    
    @Mock
    private lateinit var repository: SyncthingRepository
    
    private lateinit var viewModel: MainViewModel
    private val testDispatcher = StandardTestDispatcher()
    
    @Before
    fun setUp() {
        MockitoAnnotations.openMocks(this)
        Dispatchers.setMain(testDispatcher)
        viewModel = MainViewModel(repository)
    }
    
    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }
    
    @Test
    fun `should update system status when fetched`() = runTest {
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
        
        whenever(repository.getSystemStatus("test-api-key"))
            .thenReturn(systemStatus)
        
        // When
        viewModel.fetchSystemStatus("test-api-key")
        advanceUntilIdle()
        
        // Then
        assert(viewModel.systemStatus.value == systemStatus)
    }
}