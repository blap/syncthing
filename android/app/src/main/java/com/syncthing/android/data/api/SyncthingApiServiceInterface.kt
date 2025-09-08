package com.syncthing.android.data.api

import com.syncthing.android.data.api.model.SystemStatus
import com.syncthing.android.data.api.model.SystemVersion
import retrofit2.http.GET
import retrofit2.http.Header

interface SyncthingApiServiceInterface {
    @GET("/rest/system/status")
    suspend fun getSystemStatus(@Header("X-API-Key") apiKey: String): SystemStatus
    
    @GET("/rest/system/version")
    suspend fun getSystemVersion(@Header("X-API-Key") apiKey: String): SystemVersion
}