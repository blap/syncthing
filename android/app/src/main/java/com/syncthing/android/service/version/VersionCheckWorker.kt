package com.syncthing.android.service.version

import android.content.Context
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit

class VersionCheckWorker {
    
    companion object {
        private const val WORK_TAG = "version_check_worker"
        private const val WORK_NAME = "version_check_periodic"
        
        /**
         * Schedule periodic version checking
         * This should be called when the app starts
         */
        fun schedulePeriodicVersionCheck(context: Context) {
            // Set constraints - only run when device is charging and on unmetered network
            val constraints = Constraints.Builder()
                .setRequiredNetworkType(NetworkType.UNMETERED)
                .setRequiresCharging(false) // Set to true if you want to only check when charging
                .setRequiresBatteryNotLow(true)
                .build()
            
            // Create periodic work request - run every 24 hours
            val versionCheckRequest = PeriodicWorkRequestBuilder<VersionCheckService>(
                24, TimeUnit.HOURS
            )
                .setConstraints(constraints)
                .addTag(WORK_TAG)
                .build()
            
            // Enqueue the work
            WorkManager.getInstance(context)
                .enqueueUniquePeriodicWork(
                    WORK_NAME,
                    ExistingPeriodicWorkPolicy.KEEP, // Keep existing work if already scheduled
                    versionCheckRequest
                )
        }
        
        /**
         * Run version check immediately
         */
        fun runImmediateVersionCheck(context: Context) {
            val versionCheckRequest = androidx.work.OneTimeWorkRequestBuilder<VersionCheckService>()
                .build()
                
            WorkManager.getInstance(context)
                .enqueue(versionCheckRequest)
        }
        
        /**
         * Cancel all scheduled version checks
         */
        fun cancelVersionChecks(context: Context) {
            WorkManager.getInstance(context)
                .cancelAllWorkByTag(WORK_TAG)
        }
    }
}