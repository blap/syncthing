package com.syncthing.android.service

import android.content.Context
import android.widget.Toast
import com.syncthing.android.util.FeatureCompatibilityMatrix
import com.syncthing.android.util.VersionCompatibilityChecker

/**
 * Service that handles graceful degradation of features based on version compatibility
 */
class FeatureDegradationService(private val context: Context) {
    
    /**
     * Check if a feature is supported and handle gracefully if not
     * @param feature The feature to check
     * @param androidVersion Current Android app version
     * @param desktopVersion Current desktop Syncthing version
     * @param onSupported Action to perform if feature is supported
     * @param onUnsupported Action to perform if feature is not supported
     */
    fun handleFeature(
        feature: String,
        androidVersion: String,
        desktopVersion: String,
        onSupported: () -> Unit,
        onUnsupported: () -> Unit
    ) {
        if (FeatureCompatibilityMatrix.isFeatureSupported(feature, androidVersion, desktopVersion)) {
            onSupported()
        } else {
            handleUnsupportedFeature(feature, androidVersion, desktopVersion)
            onUnsupported()
        }
    }
    
    /**
     * Handle an unsupported feature gracefully
     */
    private fun handleUnsupportedFeature(
        feature: String,
        androidVersion: String,
        desktopVersion: String
    ) {
        val featureInfo = FeatureCompatibilityMatrix.getFeatureInfo(feature)
        val message = if (featureInfo != null) {
            "Feature '${featureInfo.featureName}' is not supported with your current versions:\n" +
            "Android: $androidVersion, Desktop: $desktopVersion\n" +
            "Requires: Android ${featureInfo.androidVersion}, Desktop ${featureInfo.desktopVersion}"
        } else {
            "Feature '$feature' is not supported with your current versions."
        }
        
        // Show a toast message to inform the user
        Toast.makeText(context, message, Toast.LENGTH_LONG).show()
    }
    
    /**
     * Disable a feature gracefully with user notification
     */
    fun disableFeatureGracefully(
        feature: String,
        androidVersion: String,
        desktopVersion: String
    ) {
        val featureInfo = FeatureCompatibilityMatrix.getFeatureInfo(feature)
        val message = if (featureInfo != null) {
            "The '${featureInfo.featureName}' feature has been disabled because it's not supported " +
            "with your current versions (Android: $androidVersion, Desktop: $desktopVersion). " +
            "Please update to use this feature."
        } else {
            "This feature has been disabled due to version incompatibility."
        }
        
        // Show a toast message to inform the user
        Toast.makeText(context, message, Toast.LENGTH_LONG).show()
    }
    
    /**
     * Get a list of all unsupported features with details
     */
    fun getUnsupportedFeaturesDetails(
        androidVersion: String,
        desktopVersion: String
    ): List<String> {
        val unsupportedFeatures = FeatureCompatibilityMatrix.getUnsupportedFeatures(
            androidVersion, desktopVersion
        )
        
        return unsupportedFeatures.mapNotNull { feature ->
            val featureInfo = FeatureCompatibilityMatrix.getFeatureInfo(feature)
            if (featureInfo != null) {
                "${featureInfo.featureName}: Requires Android ${featureInfo.androidVersion}, " +
                "Desktop ${featureInfo.desktopVersion}"
            } else {
                "$feature: Not supported with current versions"
            }
        }
    }
    
    /**
     * Show a comprehensive compatibility report
     */
    fun showCompatibilityReport(
        androidVersion: String,
        desktopVersion: String
    ) {
        val supportedCount = FeatureCompatibilityMatrix.getSupportedFeatures(
            androidVersion, desktopVersion
        ).size
        
        val unsupportedCount = FeatureCompatibilityMatrix.getUnsupportedFeatures(
            androidVersion, desktopVersion
        ).size
        
        val message = "Compatibility Report:\n" +
            "Supported features: $supportedCount\n" +
            "Unsupported features: $unsupportedCount\n" +
            "Android: $androidVersion\n" +
            "Desktop: $desktopVersion"
        
        Toast.makeText(context, message, Toast.LENGTH_LONG).show()
    }
}