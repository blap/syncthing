package com.syncthing.android.data.repository

import com.syncthing.android.data.api.SyncthingApiService
import com.syncthing.android.data.api.model.SystemStatus

class SyncthingRepository(private val apiService: SyncthingApiService) {
    
    suspend fun getSystemStatus(apiKey: String): SystemStatus {
        return apiService.getSystemStatus(apiKey)
    }
}