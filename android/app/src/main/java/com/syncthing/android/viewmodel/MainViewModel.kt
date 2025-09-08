package com.syncthing.android.viewmodel

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.viewModelScope
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.data.api.model.SystemStatus
import com.syncthing.android.data.api.model.SystemVersion
import com.syncthing.android.data.service.VersionCheckService
import kotlinx.coroutines.launch

class MainViewModel(private val repository: SyncthingRepository, application: Application) : AndroidViewModel(application) {
    
    private val _systemStatus = MutableLiveData<SystemStatus>()
    val systemStatus: LiveData<SystemStatus> = _systemStatus
    
    private val _systemVersion = MutableLiveData<SystemVersion>()
    val systemVersion: LiveData<SystemVersion> = _systemVersion
    
    private val _versionCompatibility = MutableLiveData<VersionCheckService.VersionCompatibilityResult>()
    val versionCompatibility: LiveData<VersionCheckService.VersionCompatibilityResult> = _versionCompatibility
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val versionCheckService = VersionCheckService(application)
    
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
    
    fun fetchSystemVersion(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val version = repository.getSystemVersion(apiKey)
                _systemVersion.value = version
                
                // Check version compatibility
                val compatibilityResult = versionCheckService.checkVersionCompatibility(version)
                _versionCompatibility.value = compatibilityResult
            } catch (e: Exception) {
                // Handle error
            } finally {
                _isLoading.value = false
            }
        }
    }
}