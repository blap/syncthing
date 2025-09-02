package com.syncthing.android.viewmodel

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.launch

class MainViewModel(private val repository: SyncthingRepository) : ViewModel() {
    
    private val _systemStatus = MutableLiveData<SystemStatus>()
    val systemStatus: LiveData<SystemStatus> = _systemStatus
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    fun fetchSystemStatus(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val status = repository.getSystemStatus(apiKey)
                _systemStatus.value = status
            } catch (e: Exception) {
                // Handle error
            } finally {
                _isLoading.value = false
            }
        }
    }
}