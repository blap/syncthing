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
import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.api.model.SystemVersion
import com.syncthing.android.util.VersionCompatibilityChecker
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
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

    override suspend fun doWork(): Result {
        return withContext(Dispatchers.IO) {
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
                if (compatibilityResult.needsUpdate && compatibilityResult.updateMessage != null) {
                    showUpdateNotification(compatibilityResult)
                }
                
                Result.success()
            } catch (e: Exception) {
                Result.failure()
            }
        }
    }

    /**
     * Fetch the desktop Syncthing version through the REST API using Retrofit
     */
    private suspend fun fetchDesktopVersion(): SystemVersion {
        // Create Retrofit instance
        val retrofit = Retrofit.Builder()
            .baseUrl(DESKTOP_API_URL)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
        
        // Create API service
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        
        // Make the API call (in a real implementation, you would need to handle API keys)
        // For now, we'll use a placeholder API key
        return apiService.getSystemVersion("YOUR_API_KEY_HERE")
    }

    /**
     * Get the current Android app version from package manager
     */
    private fun getAndroidAppVersion(): String {
        return try {
            val packageInfo = applicationContext.packageManager.getPackageInfo(
                applicationContext.packageName, 
                0
            )
            packageInfo.versionName ?: "1.0.0" // Default fallback
        } catch (e: Exception) {
            "1.0.0" // Default fallback
        }
    }

    /**
     * Show a notification when an update is available
     */
    private fun showUpdateNotification(compatibilityResult: VersionCompatibilityChecker.CompatibilityResult) {
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
        val intent = Intent(applicationContext, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
        }
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
            .setContentText(compatibilityResult.updateMessage ?: "Update available")
            .setStyle(
                NotificationCompat.BigTextStyle()
                    .bigText("Desktop version: ${compatibilityResult.desktopVersion}\nAndroid version: ${compatibilityResult.androidVersion}\n\n${compatibilityResult.updateMessage ?: "Update available"}")
            )
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)
            .build()
        
        // Show notification
        notificationManager.notify(NOTIFICATION_ID, notification)
    }
}