package com.syncthing.android.service

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.syncthing.android.MainActivity
import com.syncthing.android.R
import com.syncthing.android.util.VersionCompatibilityChecker

class VersionNotificationService(private val context: Context) {
    
    companion object {
        private const val CHANNEL_ID = "syncthing_version_updates"
        private const val NOTIFICATION_ID = 1001
        private const val UPDATE_NOTIFICATION_ID = 1002
    }
    
    init {
        createNotificationChannel()
    }
    
    /**
     * Show a notification about version updates
     */
    fun showVersionUpdateNotification(
        compatibilityResult: VersionCompatibilityChecker.CompatibilityResult
    ) {
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            // Pass data to the activity
            putExtra("notification_type", "version_update")
            putExtra("desktop_version", compatibilityResult.desktopVersion)
            putExtra("android_version", compatibilityResult.androidVersion)
        }
        
        val pendingIntent = PendingIntent.getActivity(
            context, 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        
        val message = compatibilityResult.updateMessage ?: "A new version of Syncthing is available"
        
        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification) // You'll need to add this drawable
            .setContentTitle("Syncthing Update Available")
            .setContentText(message)
            .setStyle(
                NotificationCompat.BigTextStyle()
                    .bigText(buildNotificationContent(compatibilityResult))
            )
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setCategory(NotificationCompat.CATEGORY_RECOMMENDATION)
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)
            .build()
        
        NotificationManagerCompat.from(context).notify(UPDATE_NOTIFICATION_ID, notification)
    }
    
    /**
     * Show a notification about compatibility issues
     */
    fun showCompatibilityIssueNotification(
        compatibilityResult: VersionCompatibilityChecker.CompatibilityResult
    ) {
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            putExtra("notification_type", "compatibility_issue")
            putExtra("desktop_version", compatibilityResult.desktopVersion)
            putExtra("android_version", compatibilityResult.androidVersion)
        }
        
        val pendingIntent = PendingIntent.getActivity(
            context, 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        
        val message = compatibilityResult.updateMessage ?: "Version compatibility issue detected"
        
        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification) // You'll need to add this drawable
            .setContentTitle("Syncthing Compatibility Issue")
            .setContentText(message)
            .setStyle(
                NotificationCompat.BigTextStyle()
                    .bigText(buildCompatibilityIssueContent(compatibilityResult))
            )
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_ERROR)
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)
            .build()
        
        NotificationManagerCompat.from(context).notify(NOTIFICATION_ID, notification)
    }
    
    /**
     * Build detailed notification content for version updates
     */
    private fun buildNotificationContent(
        compatibilityResult: VersionCompatibilityChecker.CompatibilityResult
    ): String {
        val sb = StringBuilder()
        sb.append("Desktop version: ${compatibilityResult.desktopVersion}")
        sb.append("\nAndroid version: ${compatibilityResult.androidVersion}")
        
        if (compatibilityResult.desktopCodename.isNotEmpty()) {
            sb.append("\nCodename: ${compatibilityResult.desktopCodename}")
        }
        
        if (compatibilityResult.isBeta) {
            sb.append("\nStatus: Beta version")
        } else if (compatibilityResult.isCandidate) {
            sb.append("\nStatus: Release candidate")
        }
        
        sb.append("\n\n${compatibilityResult.updateMessage ?: "Update recommended"}")
        
        return sb.toString()
    }
    
    /**
     * Build detailed notification content for compatibility issues
     */
    private fun buildCompatibilityIssueContent(
        compatibilityResult: VersionCompatibilityChecker.CompatibilityResult
    ): String {
        val sb = StringBuilder()
        sb.append("Compatibility issue detected between your Android app and desktop Syncthing.")
        sb.append("\n\nDesktop version: ${compatibilityResult.desktopVersion}")
        sb.append("\nAndroid version: ${compatibilityResult.androidVersion}")
        
        if (compatibilityResult.desktopCodename.isNotEmpty()) {
            sb.append("\nCodename: ${compatibilityResult.desktopCodename}")
        }
        
        sb.append("\n\n${compatibilityResult.updateMessage ?: "Action required"}")
        
        return sb.toString()
    }
    
    /**
     * Create notification channel for Android O and above
     */
    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val name = "Syncthing Version Updates"
            val descriptionText = "Notifications about Syncthing version updates and compatibility"
            val importance = NotificationManager.IMPORTANCE_DEFAULT
            val channel = NotificationChannel(CHANNEL_ID, name, importance).apply {
                description = descriptionText
                // Enable lights and vibration for high priority notifications
                enableLights(true)
                enableVibration(true)
            }
            
            val notificationManager: NotificationManager =
                context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            notificationManager.createNotificationChannel(channel)
        }
    }
    
    /**
     * Dismiss the version update notification
     */
    fun dismissVersionUpdateNotification() {
        NotificationManagerCompat.from(context).cancel(UPDATE_NOTIFICATION_ID)
    }
    
    /**
     * Dismiss all version-related notifications
     */
    fun dismissAllVersionNotifications() {
        NotificationManagerCompat.from(context).cancel(UPDATE_NOTIFICATION_ID)
        NotificationManagerCompat.from(context).cancel(NOTIFICATION_ID)
    }
}