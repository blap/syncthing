package com.syncthing.android.service.version

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.syncthing.android.MainActivity
import com.syncthing.android.R
import com.syncthing.android.data.api.model.SystemVersion
import com.syncthing.android.util.VersionCompatibilityChecker
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import com.syncthing.android.util.ApiConstants

class VersionCheckService(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params) {

    companion object {
        const val CHANNEL_ID = "version_check_channel"
        const val NOTIFICATION_ID = 1001
        private const val DESKTOP_API_URL = "http://localhost:8384" // Default Syncthing GUI port
    }

    override suspend fun doWork(): Result = withContext(Dispatchers.IO) {
        try {
            // Fetch the desktop version
            val desktopVersion = fetchDesktopVersion()
            
            // Get the current Android app version (this would need to be implemented)
            val androidVersion = getAndroidAppVersion()
            
            // Check compatibility
            val compatibilityResult = VersionCompatibilityChecker.checkCompatibility(
                androidVersion, 
                desktopVersion
            )
            
            // If update is needed, show notification
            if (compatibilityResult.needsUpdate) {
                showUpdateNotification(compatibilityResult)
            }
            
            Result.success()
        } catch (e: Exception) {
            Result.failure()
        }
    }

    /**
     * Fetch the desktop Syncthing version through the REST API
     */
    private fun fetchDesktopVersion(): SystemVersion {
        val url = URL("$DESKTOP_API_URL${ApiConstants.SYSTEM_VERSION_ENDPOINT}")
        val connection = url.openConnection() as HttpURLConnection
        connection.requestMethod = "GET"
        connection.connectTimeout = 5000 // 5 seconds
        connection.readTimeout = 5000 // 5 seconds
        
        try {
            val responseCode = connection.responseCode
            if (responseCode == HttpURLConnection.HTTP_OK) {
                // In a real implementation, you would parse the JSON response
                // This is a simplified example
                return SystemVersion(
                    version = "1.2.3", // This would come from the API response
                    codename = "Fermium",
                    longVersion = "1.2.3-rc.1",
                    extra = "",
                    os = "linux",
                    arch = "amd64",
                    isBeta = false,
                    isCandidate = true,
                    isRelease = false,
                    date = "2025-09-08",
                    tags = listOf("rc"),
                    stamp = "1234567890",
                    user = "user",
                    container = "docker"
                )
            } else {
                throw IOException("Failed to fetch version: HTTP $responseCode")
            }
        } finally {
            connection.disconnect()
        }
    }

    /**
     * Get the current Android app version
     * This would typically be retrieved from the package manager
     */
    private fun getAndroidAppVersion(): String {
        // In a real implementation, you would get this from the package manager
        return "1.2.0" // Placeholder
    }

    /**
     * Show a notification when an update is available
     */
    private fun showUpdateNotification(compatibilityResult: VersionCompatibilityChecker.Companion.CompatibilityResult) {
        val notificationManager = applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        
        // Create notification channel for Android O+
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Version Check Notifications",
                NotificationManager.IMPORTANCE_DEFAULT
            )
            notificationManager.createNotificationChannel(channel)
        }
        
        // Create intent to open the app when notification is tapped
        val intent = Intent(applicationContext, MainActivity::class.java)
        val pendingIntent = PendingIntent.getActivity(
            applicationContext,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        
        // Build notification
        val notification = NotificationCompat.Builder(applicationContext, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification) // You would need to add this drawable
            .setContentTitle("Syncthing Update Available")
            .setContentText(compatibilityResult.updateMessage)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)
            .build()
        
        // Show notification
        notificationManager.notify(NOTIFICATION_ID, notification)
    }
}