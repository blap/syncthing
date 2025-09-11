package com.syncthing.android.service

import android.app.DownloadManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.database.Cursor
import android.net.Uri
import android.os.Environment
import androidx.core.content.FileProvider
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import java.io.File

/**
 * Service that handles automatic updates for the Android app
 */
class AutoUpdateService(
    private val context: Context,
    private val workerParams: WorkerParameters
) : CoroutineWorker(context, workerParams) {
    
    companion object {
        private const val UPDATE_URL = "https://github.com/syncthing/syncthing-android/releases/latest/download/syncthing-android.apk"
        private const val APK_NAME = "syncthing-update.apk"
    }
    
    override suspend fun doWork(): Result {
        return try {
            // Check for updates
            if (isUpdateAvailable()) {
                downloadAndInstallUpdate()
            }
            Result.success()
        } catch (e: Exception) {
            Result.failure()
        }
    }
    
    /**
     * Check if an update is available
     * In a real implementation, this would check against a version API
     */
    private fun isUpdateAvailable(): Boolean {
        // This is a placeholder - in a real implementation, you would:
        // 1. Check the latest version from a server
        // 2. Compare with the current version
        // 3. Return true if an update is available
        return false // Placeholder
    }
    
    /**
     * Download and install the update
     */
    private fun downloadAndInstallUpdate() {
        val downloadManager = context.getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
        
        val request = DownloadManager.Request(Uri.parse(UPDATE_URL)).apply {
            setTitle("Syncthing Update")
            setDescription("Downloading latest version...")
            setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
            setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, APK_NAME)
            setMimeType("application/vnd.android.package-archive")
        }
        
        val downloadId = downloadManager.enqueue(request)
        
        // Register receiver to handle download completion
        val receiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                val id = intent.getLongExtra(DownloadManager.EXTRA_DOWNLOAD_ID, -1)
                if (id == downloadId) {
                    installUpdate(downloadManager, downloadId)
                    context.unregisterReceiver(this)
                }
            }
        }
        
        context.registerReceiver(
            receiver,
            IntentFilter(DownloadManager.ACTION_DOWNLOAD_COMPLETE)
        )
    }
    
    /**
     * Install the downloaded update
     */
    private fun installUpdate(downloadManager: DownloadManager, downloadId: Long) {
        val query = DownloadManager.Query().setFilterById(downloadId)
        val cursor: Cursor = downloadManager.query(query)
        
        if (cursor.moveToFirst()) {
            val status = cursor.getInt(cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_STATUS))
            if (status == DownloadManager.STATUS_SUCCESSFUL) {
                val localUri = cursor.getString(cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_LOCAL_URI))
                val apkUri = Uri.parse(localUri)
                
                installApk(apkUri)
            }
        }
        cursor.close()
    }
    
    /**
     * Install the APK file
     */
    private fun installApk(apkUri: Uri) {
        val intent = Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(apkUri, "application/vnd.android.package-archive")
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_GRANT_READ_URI_PERMISSION
        }
        
        context.startActivity(intent)
    }
    
    /**
     * Schedule an automatic update check
     */
    fun scheduleUpdateCheck() {
        val updateCheckRequest = OneTimeWorkRequestBuilder<AutoUpdateService>()
            .build()
            
        WorkManager.getInstance(context)
            .enqueue(updateCheckRequest)
    }
    
    /**
     * Trigger an immediate update check
     */
    fun checkForUpdatesNow() {
        scheduleUpdateCheck()
    }
}