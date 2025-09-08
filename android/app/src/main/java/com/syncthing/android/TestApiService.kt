package com.syncthing.android

import retrofit2.http.GET
import retrofit2.http.Header

interface TestApiService {
    @GET("/rest/system/status")
    suspend fun getSystemStatus(@Header("X-API-Key") apiKey: String): String
}