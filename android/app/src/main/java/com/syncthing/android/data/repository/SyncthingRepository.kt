package com.syncthing.android.data.repository

import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.api.model.Config
import com.syncthing.android.data.api.model.SystemStatus
import com.syncthing.android.data.api.model.SystemVersion
import okhttp3.ResponseBody
import retrofit2.Response

class SyncthingRepository(private val apiService: SyncthingApiServiceInterface) {
    
    suspend fun getSystemStatus(apiKey: String): SystemStatus {
        return apiService.getSystemStatus(apiKey)
    }
    
    suspend fun getSystemVersion(apiKey: String): SystemVersion {
        return apiService.getSystemVersion(apiKey)
    }
    
    // System endpoints
    suspend fun getSystemConfig(apiKey: String): Config {
        return apiService.getSystemConfig(apiKey)
    }
    
    suspend fun updateSystemConfig(apiKey: String, config: Config): Response<ResponseBody> {
        return apiService.updateSystemConfig(apiKey, config)
    }
    
    suspend fun getSystemConnections(apiKey: String): ResponseBody {
        return apiService.getSystemConnections(apiKey)
    }
    
    suspend fun shutdownSystem(apiKey: String): Response<ResponseBody> {
        return apiService.shutdownSystem(apiKey)
    }
    
    suspend fun restartSystem(apiKey: String): Response<ResponseBody> {
        return apiService.restartSystem(apiKey)
    }
    
    suspend fun checkUpgrade(apiKey: String): ResponseBody {
        return apiService.checkUpgrade(apiKey)
    }
    
    suspend fun performUpgrade(apiKey: String): Response<ResponseBody> {
        return apiService.performUpgrade(apiKey)
    }
    
    suspend fun browseSystem(apiKey: String, current: String? = null): List<String> {
        return apiService.browseSystem(apiKey, current)
    }
    
    suspend fun getSystemErrors(apiKey: String): ResponseBody {
        return apiService.getSystemErrors(apiKey)
    }
    
    suspend fun postSystemError(apiKey: String, error: String): Response<ResponseBody> {
        return apiService.postSystemError(apiKey, error)
    }
    
    suspend fun clearSystemErrors(apiKey: String): Response<ResponseBody> {
        return apiService.clearSystemErrors(apiKey)
    }
    
    suspend fun getSystemPaths(apiKey: String): Map<String, String> {
        return apiService.getSystemPaths(apiKey)
    }
    
    suspend fun pingSystemGet(apiKey: String): Map<String, String> {
        return apiService.pingSystemGet(apiKey)
    }
    
    suspend fun pingSystemPost(apiKey: String): Map<String, String> {
        return apiService.pingSystemPost(apiKey)
    }
    
    suspend fun getSystemLog(apiKey: String, since: String? = null): ResponseBody {
        return apiService.getSystemLog(apiKey, since)
    }
    
    suspend fun getSystemLogTxt(apiKey: String, since: String? = null): ResponseBody {
        return apiService.getSystemLogTxt(apiKey, since)
    }
    
    // Database endpoints
    suspend fun getDatabaseStatus(apiKey: String, folder: String): ResponseBody {
        return apiService.getDatabaseStatus(apiKey, folder)
    }
    
    suspend fun browseDatabase(
        apiKey: String,
        folder: String,
        prefix: String? = null,
        dirsonly: Boolean? = null,
        levels: Int? = null
    ): ResponseBody {
        return apiService.browseDatabase(apiKey, folder, prefix, dirsonly, levels)
    }
    
    suspend fun getDatabaseNeed(
        apiKey: String,
        folder: String,
        perpage: Int? = null,
        page: Int? = null
    ): ResponseBody {
        return apiService.getDatabaseNeed(apiKey, folder, perpage, page)
    }
    
    suspend fun scanDatabase(
        apiKey: String,
        folder: String,
        sub: List<String>? = null,
        delay: Int? = null
    ): Response<ResponseBody> {
        return apiService.scanDatabase(apiKey, folder, sub, delay)
    }
    
    suspend fun getDatabaseIgnores(apiKey: String, folder: String): ResponseBody {
        return apiService.getDatabaseIgnores(apiKey, folder)
    }
    
    suspend fun postDatabaseIgnores(
        apiKey: String,
        folder: String,
        ignores: Map<String, List<String>>
    ): Response<ResponseBody> {
        return apiService.postDatabaseIgnores(apiKey, folder, ignores)
    }
    
    // Configuration endpoints
    suspend fun getConfigFolders(apiKey: String): List<com.syncthing.android.data.api.model.Folder> {
        return apiService.getConfigFolders(apiKey)
    }
    
    suspend fun getConfigDevices(apiKey: String): List<com.syncthing.android.data.api.model.Device> {
        return apiService.getConfigDevices(apiKey)
    }
    
    suspend fun getConfigOptions(apiKey: String): com.syncthing.android.data.api.model.Options {
        return apiService.getConfigOptions(apiKey)
    }
    
    suspend fun getConfigGui(apiKey: String): com.syncthing.android.data.api.model.GuiConfiguration {
        return apiService.getConfigGui(apiKey)
    }
    
    suspend fun updateConfigFolders(
        apiKey: String,
        folders: List<com.syncthing.android.data.api.model.Folder>
    ): Response<ResponseBody> {
        return apiService.updateConfigFolders(apiKey, folders)
    }
    
    suspend fun updateConfigDevices(
        apiKey: String,
        devices: List<com.syncthing.android.data.api.model.Device>
    ): Response<ResponseBody> {
        return apiService.updateConfigDevices(apiKey, devices)
    }
    
    suspend fun updateConfigOptions(
        apiKey: String,
        options: com.syncthing.android.data.api.model.Options
    ): Response<ResponseBody> {
        return apiService.updateConfigOptions(apiKey, options)
    }
    
    suspend fun updateConfigGui(
        apiKey: String,
        gui: com.syncthing.android.data.api.model.GuiConfiguration
    ): Response<ResponseBody> {
        return apiService.updateConfigGui(apiKey, gui)
    }
    
    // Statistics endpoints
    suspend fun getDeviceStats(apiKey: String): ResponseBody {
        return apiService.getDeviceStats(apiKey)
    }
    
    suspend fun getFolderStats(apiKey: String): ResponseBody {
        return apiService.getFolderStats(apiKey)
    }
    
    // Events endpoint
    suspend fun getEvents(
        apiKey: String,
        since: Int? = null,
        limit: Int? = null,
        timeout: Int? = null
    ): ResponseBody {
        return apiService.getEvents(apiKey, since, limit, timeout)
    }
}