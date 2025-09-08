package com.syncthing.android.data.repository

import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.api.model.SystemStatus
import com.syncthing.android.data.api.model.SystemVersion

class SyncthingRepository(private val apiService: SyncthingApiServiceInterface) {
    
    suspend fun getSystemStatus(apiKey: String): SystemStatus {
        return apiService.getSystemStatus(apiKey)
    }
    
    suspend fun getSystemVersion(apiKey: String): SystemVersion {
        return apiService.getSystemVersion(apiKey)
    }
}