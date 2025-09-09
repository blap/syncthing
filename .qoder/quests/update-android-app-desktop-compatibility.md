# Android App Desktop Compatibility Update

## Summary

This document outlines the necessary updates to ensure the Android app remains compatible with the desktop version of Syncthing and to upgrade all dependencies in the Android app for improved performance, security, and compatibility. The primary focus is on synchronizing API constants, updating version checking mechanisms, and ensuring the Android app can properly communicate with the latest desktop API. Additionally, we'll update all Android dependencies to their latest stable versions.

## Overview

This document outlines the necessary updates to ensure the Android app remains compatible with the desktop version of Syncthing. The primary focus is on synchronizing API constants, updating version checking mechanisms, and ensuring the Android app can properly communicate with the latest desktop API. Additionally, we'll update the Android build system to use Gradle 9 for improved performance and compatibility.

## Architecture

### Current State
The Android app uses a set of constants defined in `ApiConstants.kt` that should match those in the desktop version's `lib/api/constants.go`. A synchronization script (`generate-android-constants.go`) exists to automatically generate the Android constants from the Go source.

The Android app currently uses Gradle 8.5 with Android Gradle Plugin 8.1.2 and Kotlin 1.9.10.

### Proposed Updates
1. Run the synchronization script to ensure constants are up-to-date
2. Update version checking services to handle API versioning
3. Verify compatibility with latest API endpoints
4. Update all dependencies to their latest stable versions
5. Update Gradle to version 9.0
6. Update Android Gradle Plugin to a compatible version
7. Update Kotlin version to latest stable version

## API Endpoints Reference

### System Endpoints
- `/rest/system/status` - System status information
- `/rest/system/config` - System configuration
- `/rest/system/connections` - Connection information
- `/rest/system/shutdown` - Shutdown the system
- `/rest/system/restart` - Restart the system
- `/rest/system/version` - Version information

### Database Endpoints
- `/rest/db/status` - Database status
- `/rest/db/browse` - Browse database contents
- `/rest/db/need` - Items needed by the database

### Statistics Endpoints
- `/rest/stats/device` - Device statistics
- `/rest/stats/folder` - Folder statistics

### Configuration Endpoints
- `/rest/config/folders` - Folder configuration
- `/rest/config/devices` - Device configuration
- `/rest/config/options` - General options

### Events Endpoint
- `/rest/events` - Event stream

## Data Models & API Mapping

### Constants Synchronization
The Android app uses the following constants that must match the desktop version:

| Desktop Constant | Android Constant | Value |
|------------------|------------------|-------|
| SystemStatusEndpoint | SYSTEM_STATUS_ENDPOINT | "/rest/system/status" |
| SystemConfigEndpoint | SYSTEM_CONFIG_ENDPOINT | "/rest/system/config" |
| SystemConnectionsEndpoint | SYSTEM_CONNECTIONS_ENDPOINT | "/rest/system/connections" |
| SystemShutdownEndpoint | SYSTEM_SHUTDOWN_ENDPOINT | "/rest/system/shutdown" |
| SystemRestartEndpoint | SYSTEM_RESTART_ENDPOINT | "/rest/system/restart" |
| SystemVersionEndpoint | SYSTEM_VERSION_ENDPOINT | "/rest/system/version" |
| DBStatusEndpoint | DB_STATUS_ENDPOINT | "/rest/db/status" |
| DBBrowseEndpoint | DB_BROWSE_ENDPOINT | "/rest/db/browse" |
| DBNeedEndpoint | DB_NEED_ENDPOINT | "/rest/db/need" |
| StatsDeviceEndpoint | STATS_DEVICE_ENDPOINT | "/rest/stats/device" |
| StatsFolderEndpoint | STATS_FOLDER_ENDPOINT | "/rest/stats/folder" |
| ConfigFoldersEndpoint | CONFIG_FOLDERS_ENDPOINT | "/rest/config/folders" |
| ConfigDevicesEndpoint | CONFIG_DEVICES_ENDPOINT | "/rest/config/devices" |
| ConfigOptionsEndpoint | CONFIG_OPTIONS_ENDPOINT | "/rest/config/options" |
| EventsEndpoint | EVENTS_ENDPOINT | "/rest/events" |
| DefaultGuiPort | DEFAULT_GUI_PORT | 8384 |
| DefaultSyncPort | DEFAULT_SYNC_PORT | 22000 |
| DefaultDiscoveryPort | DEFAULT_DISCOVERY_PORT | 21027 |
| ApiKeyHeader | API_KEY_HEADER | "X-API-Key" |
| ContentTypeHeader | CONTENT_TYPE_HEADER | "Content-Type" |
| JsonContentType | JSON_CONTENT_TYPE | "application/json" |
| ConnectionStateConnected | CONNECTION_STATE_CONNECTED | "connected" |
| ConnectionStateDisconnected | CONNECTION_STATE_DISCONNECTED | "disconnected" |
| ConnectionStatePaused | CONNECTION_STATE_PAUSED | "paused" |
| APIVersion | API_VERSION | "1.0.0" |
| APIVersionHeader | API_VERSION_HEADER | "X-API-Version" |

## Business Logic Layer

### Version Compatibility Checking
The Android app implements version compatibility checking through:

1. `VersionCheckService` - Handles version comparison logic
2. `VersionCheckWorker` - Background worker for periodic version checking
3. `VersionNotificationService` - Shows notifications when updates are available

### API Constant Synchronization
The synchronization process involves:
1. Running `go run script/generate-android-constants.go` to extract constants from `lib/api/constants.go`
2. Generating `android/app/src/main/java/com/syncthing/android/util/ApiConstants.kt`
3. Ensuring the Android app uses the latest API endpoints

## Middleware & Interceptors

### API Versioning
To ensure compatibility between desktop and Android clients:
1. Add API version headers to all responses
2. Implement version checking in the Android app
3. Gracefully handle version mismatches

## Dependency Updates

### Current Build Configuration
- Gradle version: 8.5
- Android Gradle Plugin: 8.1.2
- Kotlin version: 1.9.10

### AndroidX Dependencies
Current versions in `android/app/build.gradle`:
- androidx.core:core-ktx:1.10.1 (Latest: 1.17.0)
- androidx.appcompat:appcompat:1.6.1 (Latest: 1.7.1)
- com.google.android.material:material:1.9.0 (Latest: 1.13.0)
- androidx.constraintlayout:constraintlayout:2.1.4 (Latest: 2.2.1)
- androidx.lifecycle:lifecycle-viewmodel-ktx:2.6.1 (Latest: 2.9.3)
- androidx.lifecycle:lifecycle-livedata-ktx:2.6.1 (Latest: 2.9.3)
- androidx.lifecycle:lifecycle-runtime-ktx:2.6.1 (Latest: 2.9.3)
- androidx.room:room-runtime:2.5.0 (Latest: 2.7.2)
- androidx.room:room-ktx:2.5.0 (Latest: 2.7.2)
- androidx.room:room-compiler:2.5.0 (Latest: 2.7.2)

### Networking Dependencies
- com.squareup.retrofit2:retrofit:2.9.0 (Latest: 3.0.0)
- com.squareup.retrofit2:converter-gson:2.9.0 (Latest: 3.0.0)
- com.squareup.okhttp3:logging-interceptor:4.11.0 (Latest: 5.0.0)

### Testing Dependencies
- junit:junit:4.13.2 (Latest: 5.13.4)
- androidx.test.ext:junit:1.1.5 (Latest: 1.3.0)
- androidx.test.espresso:espresso-core:3.5.1 (Latest: 3.7.0)

### Required Updates
1. Update Gradle Wrapper to version 9.0
2. Update Android Gradle Plugin to version 8.13.0 (minimum compatible version with Gradle 9)
3. Update Kotlin version to 2.2.0 (recommended for Gradle 9)
4. Update all AndroidX dependencies to latest stable versions
5. Update networking dependencies to latest stable versions
6. Update testing dependencies to latest stable versions
7. Update JDK requirement to version 24 (latest LTS for Gradle 9)

### Gradle Wrapper Update
Update `android/gradle/wrapper/gradle-wrapper.properties`:
```
distributionBase=GRADLE_USER_HOME
distributionPath=wrapper/dists
distributionUrl=https\://services.gradle.org/distributions/gradle-9.0-bin.zip
zipStoreBase=GRADLE_USER_HOME
zipStorePath=wrapper/dists
```

### Android Gradle Plugin Update
Update `android/build.gradle`:
```gradle
buildscript {
    ext.kotlin_version = '2.2.0'
    repositories {
        google()
        mavenCentral()
    }
    dependencies {
        classpath 'com.android.tools.build:gradle:8.13.0'
        classpath "org.jetbrains.kotlin:kotlin-gradle-plugin:$kotlin_version"
    }
}
```

### Dependency Updates
Update `android/app/build.gradle` with latest stable versions:
```gradle
dependencies {
    implementation 'androidx.core:core-ktx:1.17.0'
    implementation 'androidx.appcompat:appcompat:1.7.1'
    implementation 'com.google.android.material:material:1.13.0'
    implementation 'androidx.constraintlayout:constraintlayout:2.2.1'
    
    // Architecture components
    implementation 'androidx.lifecycle:lifecycle-viewmodel-ktx:2.9.3'
    implementation 'androidx.lifecycle:lifecycle-livedata-ktx:2.9.3'
    implementation 'androidx.lifecycle:lifecycle-runtime-ktx:2.9.3'
    
    // Networking
    implementation 'com.squareup.rerofit2:retrofit:3.0.0'
    implementation 'com.squareup.retrofit2:converter-gson:3.0.0'
    implementation 'com.squareup.okhttp3:logging-interceptor:5.0.0'
    
    // Database
    implementation 'androidx.room:room-runtime:2.7.2'
    implementation 'androidx.room:room-ktx:2.7.2'
    annotationProcessor 'androidx.room:room-compiler:2.7.2'
    
    // Testing
    testImplementation 'org.junit.jupiter:junit-jupiter:5.13.4'
    androidTestImplementation 'androidx.test.ext:junit:1.3.0'
    androidTestImplementation 'androidx.test.espresso:espresso-core:3.7.0'
}
```

### Kotlin and JVM Settings Update
Update `android/app/build.gradle` to use JVM 24 compatibility:
```gradle
compileOptions {
    sourceCompatibility JavaVersion.VERSION_24
    targetCompatibility JavaVersion.VERSION_24
}
kotlinOptions {
    jvmTarget = '24'
}
```

Update `android/gradle.properties` to increase memory for Gradle 9:
```
org.gradle.jvmargs=-Xmx4096m -Dfile.encoding=UTF-8
```

### Compatibility Considerations
- Gradle 9 requires Android Gradle Plugin 8.4 or higher
- Kotlin 2.2.0 is recommended for Gradle 9 compatibility
- JDK 24 is the latest LTS version and fully supported by Gradle 9
- Retrofit 3.0.0 requires OkHttp 4.12 or higher
- All existing build scripts should remain compatible

## Testing

### Unit Tests
- Test version compatibility checking logic
- Verify API constant synchronization
- Test API endpoint communication

### Integration Tests
- Verify communication with desktop Syncthing instance
- Test version checking functionality
- Validate API responses handling

### Build System Tests
- Verify Gradle 9 build success
- Test APK generation
- Validate all dependencies work with new Gradle version
- Run all existing tests to ensure no regressions

### Verification Steps
After implementing the changes, verify the update was successful:

1. Run `./gradlew --version` to confirm Gradle 9.0 is being used
2. Check that the Android Gradle Plugin version is 8.13.0
3. Verify Kotlin version is 2.2.0
4. Confirm JVM target is set to 24
5. Run a clean build to ensure no errors
6. Run all unit tests to ensure functionality is intact
7. Generate a debug APK and verify it works correctly

### Documentation Updates
Update `android/BUILD_INSTRUCTIONS.md` to reflect the new requirements:
- Change "JDK 11 or higher" to "JDK 24 or higher"
- Update Kotlin version from 1.9.0 to 2.2.0
- Add notes about Gradle 9 compatibility

The following change needs to be made in `android/BUILD_INSTRUCTIONS.md`:
```
## Prerequisites
- Android Studio or Gradle installed
- JDK 24 or higher
- Android SDK with API level 34
```

## Implementation Steps

1. Update Gradle Wrapper to version 9.0
2. Update Android Gradle Plugin to version 8.13.0
3. Update Kotlin version to 2.2.0
4. Update JVM compatibility settings
5. Increase memory allocation for Gradle
6. Update BUILD_INSTRUCTIONS.md to reflect JDK 24 requirement
7. Run `./gradlew clean build` to verify the build
8. Run all unit tests
9. Run integration tests
10. Verify APK generation

## Rollback Plan

If issues are encountered during the Gradle 9 migration:

1. Revert `android/gradle/wrapper/gradle-wrapper.properties` to previous version
2. Revert `android/build.gradle` changes
3. Revert `android/app/build.gradle` JVM settings
4. Revert `android/gradle.properties` memory settings
5. Revert `android/BUILD_INSTRUCTIONS.md` changes
6. Run build to confirm rollback success

## Potential Issues and Solutions

### JVM Compatibility Issues
- **Issue**: Build fails with JVM compatibility errors
- **Solution**: Ensure JDK 24 is installed and JAVA_HOME is set correctly

### Memory Issues
- **Issue**: Gradle build fails with out of memory errors
- **Solution**: Increase memory allocation in `gradle.properties`

### Dependency Compatibility
- **Issue**: Some dependencies may not be compatible with Gradle 9
- **Solution**: Update dependencies to compatible versions or use compatible alternatives

### Kotlin Compilation Issues
- **Issue**: Kotlin code fails to compile with new version
- **Solution**: Update Kotlin syntax if needed or rollback to a compatible Kotlin version

### Android Gradle Plugin Issues
- **Issue**: Plugin incompatibility with Gradle 9
- **Solution**: Ensure Android Gradle Plugin version 8.4 or higher is used