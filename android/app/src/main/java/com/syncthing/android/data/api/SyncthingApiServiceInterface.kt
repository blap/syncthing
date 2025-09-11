package com.syncthing.android.data.api

import com.syncthing.android.data.api.model.Config
import com.syncthing.android.data.api.model.SystemStatus
import com.syncthing.android.data.api.model.SystemVersion
import okhttp3.ResponseBody
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Query

interface SyncthingApiServiceInterface {
    // System endpoints
    @GET("/rest/system/status")
    suspend fun getSystemStatus(@Header("X-API-Key") apiKey: String): SystemStatus
    
    @GET("/rest/system/version")
    suspend fun getSystemVersion(@Header("X-API-Key") apiKey: String): SystemVersion
    
    @GET("/rest/system/config")
    suspend fun getSystemConfig(@Header("X-API-Key") apiKey: String): Config
    
    @PUT("/rest/system/config")
    suspend fun updateSystemConfig(
        @Header("X-API-Key") apiKey: String,
        @Body config: Config
    ): Response<ResponseBody>
    
    @GET("/rest/system/connections")
    suspend fun getSystemConnections(@Header("X-API-Key") apiKey: String): ResponseBody
    
    @POST("/rest/system/shutdown")
    suspend fun shutdownSystem(@Header("X-API-Key") apiKey: String): Response<ResponseBody>
    
    @POST("/rest/system/restart")
    suspend fun restartSystem(@Header("X-API-Key") apiKey: String): Response<ResponseBody>
    
    @GET("/rest/system/upgrade")
    suspend fun checkUpgrade(@Header("X-API-Key") apiKey: String): ResponseBody
    
    @POST("/rest/system/upgrade")
    suspend fun performUpgrade(@Header("X-API-Key") apiKey: String): Response<ResponseBody>
    
    @GET("/rest/system/browse")
    suspend fun browseSystem(
        @Header("X-API-Key") apiKey: String,
        @Query("current") current: String? = null
    ): List<String>
    
    @GET("/rest/system/error")
    suspend fun getSystemErrors(@Header("X-API-Key") apiKey: String): ResponseBody
    
    @POST("/rest/system/error")
    suspend fun postSystemError(
        @Header("X-API-Key") apiKey: String,
        @Body error: String
    ): Response<ResponseBody>
    
    @POST("/rest/system/error/clear")
    suspend fun clearSystemErrors(@Header("X-API-Key") apiKey: String): Response<ResponseBody>
    
    @GET("/rest/system/paths")
    suspend fun getSystemPaths(@Header("X-API-Key") apiKey: String): Map<String, String>
    
    @GET("/rest/system/ping")
    suspend fun pingSystemGet(@Header("X-API-Key") apiKey: String): Map<String, String>
    
    @POST("/rest/system/ping")
    suspend fun pingSystemPost(@Header("X-API-Key") apiKey: String): Map<String, String>
    
    @GET("/rest/system/log")
    suspend fun getSystemLog(
        @Header("X-API-Key") apiKey: String,
        @Query("since") since: String? = null
    ): ResponseBody
    
    @GET("/rest/system/log.txt")
    suspend fun getSystemLogTxt(
        @Header("X-API-Key") apiKey: String,
        @Query("since") since: String? = null
    ): ResponseBody
    
    // Database endpoints
    @GET("/rest/db/status")
    suspend fun getDatabaseStatus(
        @Header("X-API-Key") apiKey: String,
        @Query("folder") folder: String
    ): ResponseBody
    
    @GET("/rest/db/browse")
    suspend fun browseDatabase(
        @Header("X-API-Key") apiKey: String,
        @Query("folder") folder: String,
        @Query("prefix") prefix: String? = null,
        @Query("dirsonly") dirsonly: Boolean? = null,
        @Query("levels") levels: Int? = null
    ): ResponseBody
    
    @GET("/rest/db/need")
    suspend fun getDatabaseNeed(
        @Header("X-API-Key") apiKey: String,
        @Query("folder") folder: String,
        @Query("perpage") perpage: Int? = null,
        @Query("page") page: Int? = null
    ): ResponseBody
    
    @POST("/rest/db/scan")
    suspend fun scanDatabase(
        @Header("X-API-Key") apiKey: String,
        @Query("folder") folder: String,
        @Query("sub") sub: List<String>? = null,
        @Query("delay") delay: Int? = null
    ): Response<ResponseBody>
    
    @GET("/rest/db/ignores")
    suspend fun getDatabaseIgnores(
        @Header("X-API-Key") apiKey: String,
        @Query("folder") folder: String
    ): ResponseBody
    
    @POST("/rest/db/ignores")
    suspend fun postDatabaseIgnores(
        @Header("X-API-Key") apiKey: String,
        @Query("folder") folder: String,
        @Body ignores: Map<String, List<String>>
    ): Response<ResponseBody>
    
    // Configuration endpoints
    @GET("/rest/config/folders")
    suspend fun getConfigFolders(@Header("X-API-Key") apiKey: String): List<com.syncthing.android.data.api.model.Folder>
    
    @GET("/rest/config/devices")
    suspend fun getConfigDevices(@Header("X-API-Key") apiKey: String): List<com.syncthing.android.data.api.model.Device>
    
    @GET("/rest/config/options")
    suspend fun getConfigOptions(@Header("X-API-Key") apiKey: String): com.syncthing.android.data.api.model.Options
    
    @GET("/rest/config/gui")
    suspend fun getConfigGui(@Header("X-API-Key") apiKey: String): com.syncthing.android.data.api.model.GuiConfiguration
    
    @PUT("/rest/config/folders")
    suspend fun updateConfigFolders(
        @Header("X-API-Key") apiKey: String,
        @Body folders: List<com.syncthing.android.data.api.model.Folder>
    ): Response<ResponseBody>
    
    @PUT("/rest/config/devices")
    suspend fun updateConfigDevices(
        @Header("X-API-Key") apiKey: String,
        @Body devices: List<com.syncthing.android.data.api.model.Device>
    ): Response<ResponseBody>
    
    @PUT("/rest/config/options")
    suspend fun updateConfigOptions(
        @Header("X-API-Key") apiKey: String,
        @Body options: com.syncthing.android.data.api.model.Options
    ): Response<ResponseBody>
    
    @PUT("/rest/config/gui")
    suspend fun updateConfigGui(
        @Header("X-API-Key") apiKey: String,
        @Body gui: com.syncthing.android.data.api.model.GuiConfiguration
    ): Response<ResponseBody>
    
    // Statistics endpoints
    @GET("/rest/stats/device")
    suspend fun getDeviceStats(@Header("X-API-Key") apiKey: String): ResponseBody
    
    @GET("/rest/stats/folder")
    suspend fun getFolderStats(@Header("X-API-Key") apiKey: String): ResponseBody
    
    // Events endpoint
    @GET("/rest/events")
    suspend fun getEvents(
        @Header("X-API-Key") apiKey: String,
        @Query("since") since: Int? = null,
        @Query("limit") limit: Int? = null,
        @Query("timeout") timeout: Int? = null
    ): ResponseBody
}