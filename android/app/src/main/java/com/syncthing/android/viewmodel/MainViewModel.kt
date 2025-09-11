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
import retrofit2.Response
import okhttp3.ResponseBody

class MainViewModel(private val repository: SyncthingRepository, application: Application) : AndroidViewModel(application) {
    
    private val _systemStatus = MutableLiveData<SystemStatus>()
    val systemStatus: LiveData<SystemStatus> = _systemStatus
    
    private val _systemVersion = MutableLiveData<SystemVersion>()
    val systemVersion: LiveData<SystemVersion> = _systemVersion
    
    private val _versionCompatibility = MutableLiveData<VersionCheckService.VersionCompatibilityResult>()
    val versionCompatibility: LiveData<VersionCheckService.VersionCompatibilityResult> = _versionCompatibility
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    private val _error = MutableLiveData<String?>()
    val error: LiveData<String?> = _error
    
    private val _folders = MutableLiveData<List<com.syncthing.android.data.api.model.Folder>>()
    val folders: LiveData<List<com.syncthing.android.data.api.model.Folder>> = _folders
    
    private val _devices = MutableLiveData<List<com.syncthing.android.data.api.model.Device>>()
    val devices: LiveData<List<com.syncthing.android.data.api.model.Device>> = _devices
    
    private val _options = MutableLiveData<com.syncthing.android.data.api.model.Options>()
    val options: LiveData<com.syncthing.android.data.api.model.Options> = _options
    
    private val _guiSettings = MutableLiveData<com.syncthing.android.data.api.model.GuiConfiguration>()
    val guiSettings: LiveData<com.syncthing.android.data.api.model.GuiConfiguration> = _guiSettings
    
    private val versionCheckService = VersionCheckService(application)
    
    fun fetchSystemStatus(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val status = repository.getSystemStatus(apiKey)
                _systemStatus.value = status
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun fetchSystemVersion(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val version = repository.getSystemVersion(apiKey)
                _systemVersion.value = version
                
                // Check version compatibility
                val compatibilityResult = versionCheckService.checkVersionCompatibility(version)
                _versionCompatibility.value = compatibilityResult
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun fetchFolders(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val folders = repository.getConfigFolders(apiKey)
                _folders.value = folders
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun fetchDevices(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val devices = repository.getConfigDevices(apiKey)
                _devices.value = devices
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun fetchOptions(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val options = repository.getConfigOptions(apiKey)
                _options.value = options
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun fetchGuiSettings(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val guiSettings = repository.getConfigGui(apiKey)
                _guiSettings.value = guiSettings
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun updateFolders(apiKey: String, folders: List<com.syncthing.android.data.api.model.Folder>) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val response = repository.updateConfigFolders(apiKey, folders)
                if (response.isSuccessful) {
                    _folders.value = folders
                } else {
                    _error.value = "Failed to update folders: ${response.message()}"
                }
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun updateDevices(apiKey: String, devices: List<com.syncthing.android.data.api.model.Device>) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val response = repository.updateConfigDevices(apiKey, devices)
                if (response.isSuccessful) {
                    _devices.value = devices
                } else {
                    _error.value = "Failed to update devices: ${response.message()}"
                }
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun updateOptions(apiKey: String, options: com.syncthing.android.data.api.model.Options) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val response = repository.updateConfigOptions(apiKey, options)
                if (response.isSuccessful) {
                    _options.value = options
                } else {
                    _error.value = "Failed to update options: ${response.message()}"
                }
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun updateGuiSettings(apiKey: String, guiSettings: com.syncthing.android.data.api.model.GuiConfiguration) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val response = repository.updateConfigGui(apiKey, guiSettings)
                if (response.isSuccessful) {
                    _guiSettings.value = guiSettings
                } else {
                    _error.value = "Failed to update GUI settings: ${response.message()}"
                }
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun restartSystem(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val response = repository.restartSystem(apiKey)
                if (!response.isSuccessful) {
                    _error.value = "Failed to restart system: ${response.message()}"
                }
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
    
    fun shutdownSystem(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null
            try {
                val response = repository.shutdownSystem(apiKey)
                if (!response.isSuccessful) {
                    _error.value = "Failed to shutdown system: ${response.message()}"
                }
            } catch (e: Exception) {
                _error.value = e.message ?: "Unknown error"
            } finally {
                _isLoading.value = false
            }
        }
    }
}