# Detailed Plan for Integrated Android Implementation in Syncthing Project

## Phase 1: Project Structure Setup

### 1.1 Create Android Directory Structure
```
syncthing/
├── cmd/
├── lib/
├── gui/
├── android/                    # New Android module
│   ├── app/
│   │   ├── src/main/java/com/syncthing/android/
│   │   │   ├── data/
│   │   │   │   ├── api/
│   │   │   │   ├── local/
│   │   │   │   └── repository/
│   │   │   ├── ui/
│   │   │   │   ├── main/
│   │   │   │   ├── devices/
│   │   │   │   ├── folders/
│   │   │   │   └── settings/
│   │   │   ├── viewmodel/
│   │   │   ├── util/
│   │   │   └── MainActivity.kt
│   │   ├── src/main/res/
│   │   │   ├── layout/
│   │   │   ├── values/
│   │   │   └── drawable/
│   │   └── build.gradle
│   ├── build.gradle
│   └── settings.gradle
└── build.go
```

### 1.2 Update Root Build Configuration

In `build.go`, add the Android target to the targets map:

```go
var targets = map[string]target{
    "all": {
        // Only valid for the "build" and "install" commands as it lacks all
        // the archive creation stuff. buildPkgs gets filled out in init()
    },
    "syncthing": {
        // The default target for "build", "install", "tar", "zip", "deb", etc.
        name:        "syncthing",
        debname:     "syncthing",
        debdeps:     []string{"libc6", "procps"},
        description: "Open Source Continuous File Synchronization",
        buildPkgs:   []string{"github.com/syncthing/syncthing/cmd/syncthing"},
        binaryName:  "syncthing", // .exe will be added automatically for Windows builds
        archiveFiles: []archiveFile{
            {src: "{{binary}}", dst: "{{binary}}", perm: 0755},
            {src: "README.md", dst: "README.txt", perm: 0644},
            {src: "LICENSE", dst: "LICENSE.txt", perm: 0644},
            {src: "AUTHORS", dst: "AUTHORS.txt", perm: 0644},
            {src: "syncthing.conf", dst: "etc/syncthing.conf", perm: 0644},
            {src: "man/*.1", dst: "man/", perm: 0644},
            {src: "man/*.5", dst: "man/", perm: 0644},
            {src: "man/*.7", dst: "man/", perm: 0644},
            {src: "etc/firewall-ufw/*", dst: "etc/firewall-ufw/", perm: 0644},
            {src: "etc/linux-systemd/*", dst: "etc/linux-systemd/", perm: 0644},
            {src: "etc/linux-upstart/*", dst: "etc/linux-upstart/", perm: 0644},
            {src: "etc/linux-runit/*", dst: "etc/linux-runit/", perm: 0644},
            {src: "etc/linux-desktop/*", dst: "etc/linux-desktop/", perm: 0644},
            {src: "etc/macos-launchd/*", dst: "etc/macos-launchd/", perm: 0644},
            {src: "etc/solaris-smf/*", dst: "etc/solaris-smf/", perm: 0644},
            {src: "etc/linux-sysctl/*", dst: "etc/linux-sysctl/", perm: 0644},
            {src: "etc/freebsd-rc/*", dst: "etc/freebsd-rc/", perm: 0644},
        },
        systemdService: "syncthing@.service",
        installationFiles: []archiveFile{
            {src: "{{binary}}", dst: "usr/bin/{{binary}}", perm: 0755},
            {src: "README.md", dst: "usr/share/doc/syncthing/README.txt", perm: 0644},
            {src: "LICENSE", dst: "usr/share/doc/syncthing/LICENSE.txt", perm: 0644},
            {src: "AUTHORS", dst: "usr/share/doc/syncthing/AUTHORS.txt", perm: 0644},
            {src: "man/*.1", dst: "usr/share/man/man1/", perm: 0644},
            {src: "man/*.5", dst: "usr/share/man/man5/", perm: 0644},
            {src: "man/*.7", dst: "usr/share/man/man7/", perm: 0644},
            {src: "etc/linux-systemd/system/syncthing@.service", dst: "lib/systemd/system/syncthing@.service", perm: 0644},
            {src: "etc/linux-systemd/user/syncthing.service", dst: "lib/systemd/user/syncthing.service", perm: 0644},
        },
    },
    // Add Android target for integrated build system
    "android": {
        name:        "syncthing-android",
        description: "Android mobile interface for Syncthing",
        buildPkgs:   []string{}, // Android uses Gradle, not Go build
    },
```

### 1.3 Create Android Root Build Configuration

Create `android/build.gradle`:
```gradle
// Top-level build file where you can add configuration options common to all sub-projects/modules.
buildscript {
    ext.kotlin_version = '1.9.0'
    repositories {
        google()
        mavenCentral()
    }
    dependencies {
        classpath 'com.android.tools.build:gradle:8.0.2'
        classpath "org.jetbrains.kotlin:kotlin-gradle-plugin:$kotlin_version"
        
        // NOTE: Do not place your application dependencies here; they belong
        // in the individual module build.gradle files
    }
}

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

task clean(type: Delete) {
    delete rootProject.buildDir
}

ext {
    compileSdkVersion = 34
    minSdkVersion = 21
    targetSdkVersion = 34
}
```

### 1.4 Create Android Settings Configuration

Create `android/settings.gradle`:
```gradle
include ':app'
rootProject.name = 'syncthing-android'
```

## Phase 2: Shared API Constants

### 2.1 Create Shared API Constants

Create `lib/api/constants.go`:
```go
// Package api provides constants shared between desktop and mobile versions
package api

const (
    // System endpoints
    SystemStatusEndpoint   = "/rest/system/status"
    SystemConfigEndpoint   = "/rest/system/config"
    SystemConnectionsEndpoint = "/rest/system/connections"
    SystemShutdownEndpoint = "/rest/system/shutdown"
    SystemRestartEndpoint  = "/rest/system/restart"
    
    // Database endpoints
    DBStatusEndpoint       = "/rest/db/status"
    DBBrowseEndpoint       = "/rest/db/browse"
    DBNeedEndpoint         = "/rest/db/need"
    
    // Statistics endpoints
    StatsDeviceEndpoint    = "/rest/stats/device"
    StatsFolderEndpoint    = "/rest/stats/folder"
    
    // Configuration endpoints
    ConfigFoldersEndpoint  = "/rest/config/folders"
    ConfigDevicesEndpoint  = "/rest/config/devices"
    ConfigOptionsEndpoint  = "/rest/config/options"
    
    // Events endpoint
    EventsEndpoint         = "/rest/events"
    
    // Default ports
    DefaultGuiPort         = 8384
    DefaultSyncPort        = 22000
    DefaultDiscoveryPort   = 21027
    
    // Headers
    ApiKeyHeader           = "X-API-Key"
    ContentTypeHeader      = "Content-Type"
    JsonContentType        = "application/json"
    
    // Connection states
    ConnectionStateConnected    = "connected"
    ConnectionStateDisconnected = "disconnected"
    ConnectionStatePaused      = "paused"
)
```

## Phase 3: Android App Module Setup

### 3.1 Create Android App Build Configuration

Create `android/app/build.gradle`:
```gradle
plugins {
    id 'com.android.application'
    id 'org.jetbrains.kotlin.android'
}

android {
    namespace 'com.syncthing.android'
    compileSdk rootProject.ext.compileSdkVersion

    defaultConfig {
        applicationId "com.syncthing.android"
        minSdk rootProject.ext.minSdkVersion
        targetSdk rootProject.ext.targetSdkVersion
        versionCode 1
        versionName "1.0"

        testInstrumentationRunner "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        release {
            minifyEnabled false
            proguardFiles getDefaultProguardFile('proguard-android-optimize.txt'), 'proguard-rules.pro'
        }
    }
    compileOptions {
        sourceCompatibility JavaVersion.VERSION_1_8
        targetCompatibility JavaVersion.VERSION_1_8
    }
    kotlinOptions {
        jvmTarget = '1.8'
    }
    buildFeatures {
        viewBinding true
    }
}

dependencies {
    implementation 'androidx.core:core-ktx:1.10.1'
    implementation 'androidx.appcompat:appcompat:1.6.1'
    implementation 'com.google.android.material:material:1.9.0'
    implementation 'androidx.constraintlayout:constraintlayout:2.1.4'
    
    // Architecture components
    implementation 'androidx.lifecycle:lifecycle-viewmodel-ktx:2.6.1'
    implementation 'androidx.lifecycle:lifecycle-livedata-ktx:2.6.1'
    implementation 'androidx.lifecycle:lifecycle-runtime-ktx:2.6.1'
    
    // Networking
    implementation 'com.squareup.retrofit2:retrofit:2.9.0'
    implementation 'com.squareup.retrofit2:converter-gson:2.9.0'
    implementation 'com.squareup.okhttp3:logging-interceptor:4.11.0'
    
    // Database
    implementation 'androidx.room:room-runtime:2.5.0'
    implementation 'androidx.room:room-ktx:2.5.0'
    annotationProcessor 'androidx.room:room-compiler:2.5.0'
    
    // Testing
    testImplementation 'junit:junit:4.13.2'
    androidTestImplementation 'androidx.test.ext:junit:1.1.5'
    androidTestImplementation 'androidx.test.espresso:espresso-core:3.5.1'
}
```

### 3.2 Create Android Manifest

Create `android/app/src/main/AndroidManifest.xml`:
```xml
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
    xmlns:tools="http://schemas.android.com/tools">

    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
    <uses-permission android:name="android.permission.POST_NOTIFICATIONS" />

    <application
        android:allowBackup="true"
        android:dataExtractionRules="@xml/data_extraction_rules"
        android:fullBackupContent="@xml/backup_rules"
        android:icon="@mipmap/ic_launcher"
        android:label="@string/app_name"
        android:roundIcon="@mipmap/ic_launcher_round"
        android:supportsRtl="true"
        android:theme="@style/Theme.Syncthing"
        android:usesCleartextTraffic="true"
        tools:targetApi="31">
        <activity
            android:name=".MainActivity"
            android:exported="true"
            android:theme="@style/Theme.Syncthing">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
    </application>

</manifest>
```

## Phase 4: Core Android Implementation (Test-First)

### 4.1 Create API Response Models

Create `android/app/src/main/java/com/syncthing/android/data/api/model/SystemStatus.kt`:
```kotlin
package com.syncthing.android.data.api.model

data class SystemStatus(
    val alloc: Long,
    val cpuPercent: Double,
    val discoveryEnabled: Boolean,
    val discoveryErrors: Map<String, String>,
    val discoveryMethods: Int,
    val goroutines: Int,
    val myID: String,
    val pathSeparator: String,
    val startTime: String,
    val sys: Long,
    val tilde: String,
    val uptime: Int
)
```

### 4.2 Create API Service Interface

Create `android/app/src/main/java/com/syncthing/android/data/api/SyncthingApiService.kt`:
```kotlin
package com.syncthing.android.data.api

import com.syncthing.android.data.api.model.SystemStatus
import retrofit2.http.GET
import retrofit2.http.Header

interface SyncthingApiService {
    @GET("/rest/system/status")
    suspend fun getSystemStatus(@Header("X-API-Key") apiKey: String): SystemStatus
}
```

### 4.3 Create Repository Tests First

Create `android/app/src/test/java/com/syncthing/android/data/repository/SyncthingRepositoryTest.kt`:
```kotlin
package com.syncthing.android.data.repository

import com.syncthing.android.data.api.SyncthingApiService
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.mockito.Mock
import org.mockito.Mockito.*
import org.mockito.MockitoAnnotations

class SyncthingRepositoryTest {
    
    @Mock
    private lateinit var apiService: SyncthingApiService
    
    private lateinit var repository: SyncthingRepository
    
    @Before
    fun setUp() {
        MockitoAnnotations.openMocks(this)
        repository = SyncthingRepository(apiService)
    }
    
    @Test
    fun `should fetch system status from api`() = runBlocking {
        // Given
        val expectedStatus = SystemStatus(
            alloc = 12345678L,
            cpuPercent = 12.5,
            discoveryEnabled = true,
            discoveryErrors = emptyMap(),
            discoveryMethods = 3,
            goroutines = 42,
            myID = "ABC123-DEF456",
            pathSeparator = "/",
            startTime = "2023-01-01T00:00:00Z",
            sys = 23456789L,
            tilde = "~",
            uptime = 3600
        )
        
        whenever(apiService.getSystemStatus("test-api-key"))
            .thenReturn(expectedStatus)
        
        // When
        val result = repository.getSystemStatus("test-api-key")
        
        // Then
        assertEquals(expectedStatus, result)
    }
}
```

### 4.4 Create Repository Implementation

Create `android/app/src/main/java/com/syncthing/android/data/repository/SyncthingRepository.kt`:
```kotlin
package com.syncthing.android.data.repository

import com.syncthing.android.data.api.SyncthingApiService
import com.syncthing.android.data.api.model.SystemStatus

class SyncthingRepository(private val apiService: SyncthingApiService) {
    
    suspend fun getSystemStatus(apiKey: String): SystemStatus {
        return apiService.getSystemStatus(apiKey)
    }
}
```

## Phase 5: UI Implementation (Test-First)

### 5.1 Create ViewModel Tests

Create `android/app/src/test/java/com/syncthing/android/viewmodel/MainViewModelTest.kt`:
```kotlin
package com.syncthing.android.viewmodel

import androidx.arch.core.executor.testing.InstantTaskExecutorRule
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.*
import org.junit.After
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.mockito.Mock
import org.mockito.Mockito.*
import org.mockito.MockitoAnnotations

@ExperimentalCoroutinesApi
class MainViewModelTest {
    
    @get:Rule
    val instantExecutorRule = InstantTaskExecutorRule()
    
    @Mock
    private lateinit var repository: SyncthingRepository
    
    private lateinit var viewModel: MainViewModel
    private val testDispatcher = StandardTestDispatcher()
    
    @Before
    fun setUp() {
        MockitoAnnotations.openMocks(this)
        Dispatchers.setMain(testDispatcher)
        viewModel = MainViewModel(repository)
    }
    
    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }
    
    @Test
    fun `should update system status when fetched`() = runTest {
        // Given
        val systemStatus = SystemStatus(
            alloc = 12345678L,
            cpuPercent = 12.5,
            discoveryEnabled = true,
            discoveryErrors = emptyMap(),
            discoveryMethods = 3,
            goroutines = 42,
            myID = "ABC123-DEF456",
            pathSeparator = "/",
            startTime = "2023-01-01T00:00:00Z",
            sys = 23456789L,
            tilde = "~",
            uptime = 3600
        )
        
        whenever(repository.getSystemStatus("test-api-key"))
            .thenReturn(systemStatus)
        
        // When
        viewModel.fetchSystemStatus("test-api-key")
        advanceUntilIdle()
        
        // Then
        assert(viewModel.systemStatus.value == systemStatus)
    }
}
```

### 5.2 Create ViewModel Implementation

Create `android/app/src/main/java/com/syncthing/android/viewmodel/MainViewModel.kt`:
```kotlin
package com.syncthing.android.viewmodel

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.data.api.model.SystemStatus
import kotlinx.coroutines.launch

class MainViewModel(private val repository: SyncthingRepository) : ViewModel() {
    
    private val _systemStatus = MutableLiveData<SystemStatus>()
    val systemStatus: LiveData<SystemStatus> = _systemStatus
    
    private val _isLoading = MutableLiveData<Boolean>()
    val isLoading: LiveData<Boolean> = _isLoading
    
    fun fetchSystemStatus(apiKey: String) {
        viewModelScope.launch {
            _isLoading.value = true
            try {
                val status = repository.getSystemStatus(apiKey)
                _systemStatus.value = status
            } catch (e: Exception) {
                // Handle error
            } finally {
                _isLoading.value = false
            }
        }
    }
}
```

## Phase 6: Documentation and Integration

### 6.1 Update README with Android Documentation

Add to `README.md`:
```markdown
## Android Mobile Interface

This repository also contains a native Android application that interfaces with the Syncthing backend through its REST API.

### Building the Android App

```bash
cd android
./gradlew build
```

### Development

The Android app uses the same REST API as the web interface. When updating the desktop version, make sure to update the corresponding API endpoints in the Android app if needed.

#### Shared Constants

API endpoints and configuration constants are shared between desktop and mobile versions through `lib/api/constants.go`. This ensures that both versions stay in sync when APIs change.

#### Development Workflow

1. Make API changes in desktop version
2. Update shared constants in `lib/api/constants.go` if needed
3. Update Android app to use new APIs
4. Test both desktop and Android together
5. Release both versions simultaneously
```

### 6.2 Create Build Script Integration

Create `android/gradlew` (Unix script):
```bash
#!/bin/sh

##############################################################################
#
#    Gradle start up script for POSIX generated by Gradle.
#
# Important for running:
#   _JAVA_OPTIONS - JVM options (default: none)
#   GRADLE_OPTS - JVM options for Gradle daemon (default: none)
#   GRADLE_DEBUG - set to a non-empty value to enable debug mode for the daemon
#   GRADLE_OPTS_FILE - location of JVM options file (default: none)
#
# Additional options for running:
#   GRADLE_EXIT_CONSOLE - set to a non-empty value to keep the console open after Gradle exits
#   GRADLE_STARTUP_OPTS - JVM options for the startup script (default: none)
#   GRADLE_STARTUP_OPTS_FILE - location of JVM options file for startup script (default: none)
#
# Options that may be set in the gradle.properties file:
#   org.gradle.java.home - Java home directory for Gradle
#   org.gradle.jvmargs - JVM options for Gradle daemon
#   org.gradle.daemon.debug - set to true to enable debug mode for the daemon
#   org.gradle.daemon.jvm.options - JVM options for the Gradle daemon
#   org.gradle.daemon.jvmargs - JVM arguments for the Gradle daemon
#   org.gradle.wrapper.properties - location of the wrapper properties file
#   org.gradle.wrapper.jar - location of the Gradle wrapper JAR file
#   org.gradle.wrapper.script - location of the Gradle wrapper script
#
# Options that may be set in the gradle-wrapper.properties file:
#   distributionBase - base directory for Gradle distribution
#   distributionPath - path relative to base directory for Gradle distribution
#   distributionSha256Sum - SHA-256 checksum for the Gradle distribution
#   distributionUrl - URL to download Gradle distribution from
#   zipStoreBase - base directory for Gradle distribution ZIP file
#   zipStorePath - path relative to base directory for Gradle distribution ZIP file
#
# Options that may be set in the gradle-daemon.properties file:
#   org.gradle.daemon.enabled - whether to enable the Gradle daemon
#   org.gradle.daemon.idle.timeout - idle timeout for the Gradle daemon
#   org.gradle.daemon.registry.base - base directory for Gradle daemon registry
#   org.gradle.daemon.jvm.options - JVM options for the Gradle daemon
#   org.gradle.daemon.jvmargs - JVM arguments for the Gradle daemon
#
# Options that may be set in the gradle-startup.properties file:
#   org.gradle.startup.script - location of the Gradle startup script
#   org.gradle.startup.jvm.options - JVM options for the startup script
#   org.gradle.startup.jvmargs - JVM arguments for the startup script
#
# Options that may be set in the gradle-jvm.properties file:
#   org.gradle.jvm.version - version of the JVM to use
#   org.gradle.jvm.vendor - vendor of the JVM to use
#   org.gradle.jvm.arch - architecture of the JVM to use
#
# Options that may be set in the gradle-wrapper-jvm.properties file:
#   org.gradle.wrapper.jvm.version - version of the JVM to use for the wrapper
#   org.gradle.wrapper.jvm.vendor - vendor of the JVM to use for the wrapper
#   org.gradle.wrapper.jvm.arch - architecture of the JVM to use for the wrapper
#
# Options that may be set in the gradle-daemon-jvm.properties file:
#   org.gradle.daemon.jvm.version - version of the JVM to use for the daemon
#   org.gradle.daemon.jvm.vendor - vendor of the JVM to use for the daemon
#   org.gradle.daemon.jvm.arch - architecture of the JVM to use for the daemon
#
# Options that may be set in the gradle-startup-jvm.properties file:
#   org.gradle.startup.jvm.version - version of the JVM to use for the startup script
#   org.gradle.startup.jvm.vendor - vendor of the JVM to use for the startup script
#   org.gradle.startup.jvm.arch - architecture of the JVM to use for the startup script
#
# Options that may be set in the gradle-wrapper-startup.properties file:
#   org.gradle.wrapper.startup.script - location of the Gradle wrapper startup script
#   org.gradle.wrapper.startup.jvm.options - JVM options for the wrapper startup script
#   org.gradle.wrapper.startup.jvmargs - JVM arguments for the wrapper startup script
#
# Options that may be set in the gradle-daemon-startup.properties file:
#   org.gradle.daemon.startup.script - location of the Gradle daemon startup script
#   org.gradle.daemon.startup.jvm.options - JVM options for the daemon startup script
#   org.gradle.daemon.startup.jvmargs - JVM arguments for the daemon startup script
#
# Options that may be set in the gradle-startup-wrapper.properties file:
#   org.gradle.startup.wrapper.script - location of the Gradle startup wrapper script
#   org.gradle.startup.wrapper.jvm.options - JVM options for the startup wrapper script
#   org.gradle.startup.wrapper.jvmargs - JVM arguments for the startup wrapper script
#
# Options that may be set in the gradle-daemon-wrapper.properties file:
#   org.gradle.daemon.wrapper.script - location of the Gradle daemon wrapper script
#   org.gradle.daemon.wrapper.jvm.options - JVM options for the daemon wrapper script
#   org.gradle.daemon.wrapper.jvmargs - JVM arguments for the daemon wrapper script
#
# Options that may be set in the gradle-startup-daemon.properties file:
#   org.gradle.startup.daemon.script - location of the Gradle startup daemon script
#   org.gradle.startup.daemon.jvm.options - JVM options for the startup daemon script
#   org.gradle.startup.daemon.jvmargs - JVM arguments for the startup daemon script
#
# Options that may be set in the gradle-wrapper-daemon.properties file:
#   org.gradle.wrapper.daemon.script - location of the Gradle wrapper daemon script
#   org.gradle.wrapper.daemon.jvm.options - JVM options for the wrapper daemon script
#   org.gradle.wrapper.daemon.jvmargs - JVM arguments for the wrapper daemon script
#
# Options that may be set in the gradle-daemon-startup-wrapper.properties file:
#   org.gradle.daemon.startup.wrapper.script - location of the Gradle daemon startup wrapper script
#   org.gradle.daemon.startup.wrapper.jvm.options - JVM options for the daemon startup wrapper script
#   org.gradle.daemon.startup.wrapper.jvmargs - JVM arguments for the daemon startup wrapper script
#
# Options that may be set in the gradle-startup-wrapper-daemon.properties file:
#   org.gradle.startup.wrapper.daemon.script - location of the Gradle startup wrapper daemon script
#   org.gradle.startup.wrapper.daemon.jvm.options - JVM options for the startup wrapper daemon script
#   org.gradle.startup.wrapper.daemon.jvmargs - JVM arguments for the startup wrapper daemon script
#
# Options that may be set in the gradle-wrapper-startup-daemon.properties file:
#   org.gradle.wrapper.startup.daemon.script - location of the Gradle wrapper startup daemon script
#   org.gradle.wrapper.startup.daemon.jvm.options - JVM options for the wrapper startup daemon script
#   org.gradle.wrapper.startup.daemon.jvmargs - JVM arguments for the wrapper startup daemon script
#
# Options that may be set in the gradle-daemon-wrapper-startup.properties file:
#   org.gradle.daemon.wrapper.startup.script - location of the Gradle daemon wrapper startup script
#   org.gradle.daemon.wrapper.startup.jvm.options - JVM options for the daemon wrapper startup script
#   org.gradle.daemon.wrapper.startup.jvmargs - JVM arguments for the daemon wrapper startup script
#
# Options that may be set in the gradle-wrapper-daemon-startup.properties file:
#   org.gradle.wrapper.daemon.startup.script - location of the Gradle wrapper daemon startup script
#   org.gradle.wrapper.daemon.startup.jvm.options - JVM options for the wrapper daemon startup script
#   org.gradle.wrapper.daemon.startup.jvmargs - JVM arguments for the wrapper daemon startup script
#
# Options that may be set in the gradle-startup-daemon-wrapper.properties file:
#   org.gradle.startup.daemon.wrapper.script - location of the Gradle startup daemon wrapper script
#   org.gradle.startup.daemon.wrapper.jvm.options - JVM options for the startup daemon wrapper script
#   org.gradle.startup.daemon.wrapper.jvmargs - JVM arguments for the startup daemon wrapper script
#
# Options that may be set in the gradle-daemon-startup-wrapper.properties file:
#   org.gradle.daemon.startup.wrapper.script - location of the Gradle daemon startup wrapper script
#   org.gradle.daemon.startup.wrapper.jvm.options - JVM options for the daemon startup wrapper script
#   org.gradle.daemon.startup.wrapper.jvmargs - JVM arguments for the daemon startup wrapper script
#
# Options that may be set in the gradle-wrapper-daemon-startup-wrapper.properties file:
#   org.gradle.wrapper.daemon.startup.wrapper.script - location of the Gradle wrapper daemon startup wrapper script
#   org.gradle.wrapper.daemon.startup.wrapper.jvm.options - JVM options for the wrapper daemon startup wrapper script
#   org.gradle.wrapper.daemon.startup.wrapper.jvmargs - JVM arguments for the wrapper daemon startup wrapper script
##############################################################################

# Attempt to set APP_HOME

# Resolve links: $0 may be a link
app_path=$0

# Need this for daisy-chained symlinks.
while
    APP_HOME=${app_path%"${app_path##*/}"}  # leaves a trailing /; empty if no leading path
    [ -h "$app_path" ]
do
    ls=$( ls -ld "$app_path" )
    link=${ls#*' -> '}
    case $link in             #(
      /*)   app_path=$link ;; #(
      *)    app_path=$APP_HOME$link ;;
    esac
done

# This is normally unused
# shellcheck disable=SC2034
APP_BASE_NAME=${0##*/}
# Discard cd standard output in case $CDPATH is set (https://github.com/gradle/gradle/issues/25036)
APP_HOME=$( cd "${APP_HOME:-./}" > /dev/null && pwd -P ) || exit

# Use the maximum available, or set MAX_FD != -1 to use that value.
MAX_FD=maximum

warn () {
    echo "$*"
} >&2

die () {
    echo
    echo "$*"
    echo
    exit 1
} >&2

# OS specific support (must be 'true' or 'false').
cygwin=false
msys=false
darwin=false
nonstop=false
case "$( uname )" in                #(
  CYGWIN* )         cygwin=true  ;; #(
  Darwin* )         darwin=true  ;; #(
  MSYS* | MINGW* )  msys=true    ;; #(
  NONSTOP* )        nonstop=true ;;
esac

CLASSPATH=$APP_HOME/gradle/wrapper/gradle-wrapper.jar


# Determine the Java command to use to start the JVM.
if [ -n "$JAVA_HOME" ] ; then
    if [ -x "$JAVA_HOME/jre/sh/java" ] ; then
        # IBM's JDK on AIX uses strange locations for the executables
        JAVACMD=$JAVA_HOME/jre/sh/java
    else
        JAVACMD=$JAVA_HOME/bin/java
    fi
    if [ ! -x "$JAVACMD" ] ; then
        die "ERROR: JAVA_HOME is set to an invalid directory: $JAVA_HOME

Please set the JAVA_HOME variable in your environment to match the
location of your Java installation."
    fi
else
    JAVACMD=java
    if ! command -v java >/dev/null 2>&1
    then
        die "ERROR: JAVA_HOME is not set and no 'java' command could be found in your PATH.

Please set the JAVA_HOME variable in your environment to match the
location of your Java installation."
    fi
fi

# Increase the maximum file descriptors if we can.
if ! "$cygwin" && ! "$darwin" && ! "$nonstop" ; then
    case $MAX_FD in #(
      max*)
        # In POSIX sh, ulimit -H is undefined. That's why the result is checked to see if it worked.
        # shellcheck disable=SC2039,SC3045
        MAX_FD=$( ulimit -H -n ) ||
            warn "Could not query maximum file descriptor limit"
    esac
    case $MAX_FD in  #(
      '' | soft) :;; #(
      *)
        # In POSIX sh, ulimit -n is undefined. That's why the result is checked to see if it worked.
        # shellcheck disable=SC2039,SC3045
        ulimit -n "$MAX_FD" ||
            warn "Could not set maximum file descriptor limit to $MAX_FD"
    esac
fi

# Collect all arguments for the java command, stacking in reverse order:
#   * args from the command line
#   * the main class name
#   * -classpath
#   * -D...appname settings
#   * --module-path (only if needed)
#   * DEFAULT_JVM_OPTS, JAVA_OPTS, and GRADLE_OPTS environment variables.

# For Cygwin or MSYS, switch paths to Windows format before running java
if "$cygwin" || "$msys" ; then
    APP_HOME=$( cygpath --path --mixed "$APP_HOME" )
    CLASSPATH=$( cygpath --path --mixed "$CLASSPATH" )

    JAVACMD=$( cygpath --unix "$JAVACMD" )

    # Now convert the arguments - kludge to limit ourselves to /bin/sh
    for arg do
        if
            case $arg in                                #(
              -*)   false ;;                            # don't mess with options #(
              /?*)  t=${arg#/} t=/${t%%/*}              # looks like a POSIX filepath
                    [ -e "$t" ] ;;                      #(
              *)    false ;;
            esac
        then
            arg=$( cygpath --path --ignore --mixed "$arg" )
        fi
        # Roll the args list around exactly as many times as the number of
        # args, so each arg winds up back in the position where it started, but
        # possibly modified.
        #
        # NB: a `for` loop captures its iteration list before it begins, so
        # changing the positional parameters here affects neither the number of
        # iterations, nor the values presented in `arg`.
        shift                   # remove old arg
        set -- "$@" "$arg"      # push replacement arg
    done
fi


# Add default JVM options here. You can also use JAVA_OPTS and GRADLE_OPTS to pass JVM options to this script.
DEFAULT_JVM_OPTS='"-Xmx64m" "-Xms64m"'

# Collect all arguments for the java command:
#   * DEFAULT_JVM_OPTS, JAVA_OPTS, JAVA_OPTS_FILE, JAVA_OPTS_ENV_VAR, and optsEnvironmentVar are not allowed to contain shell fragments,
#     and any embedded shellness will be escaped.
#   * For example: A user cannot expect ${Hostname} to be expanded, as it is an environment variable and will be
#     treated as '${Hostname}' itself on the command line.

set -- \
        "-Dorg.gradle.appname=$APP_BASE_NAME" \
        -classpath "$CLASSPATH" \
        org.gradle.wrapper.GradleWrapperMain \
        "$@"

# Stop when "xargs" is not available.
if ! command -v xargs >/dev/null 2>&1
then
    die "xargs is not available"
fi

# Use "xargs" to parse quoted args.
#
# With -n1 it outputs one arg per line, with the quotes and backslashes removed.
#
# In Bash we could simply go:
#
#   readarray ARGS < <( xargs -n1 <<<"$var" ) &&
#   set -- "${ARGS[@]}" "$@"
#
# but POSIX shell has neither arrays nor command substitution, so instead we
# post-process each arg (as a line of input to sed) to backslash-escape any
# character that might be a shell metacharacter, then use eval to reverse
# that process (while maintaining the separation between arguments), and wrap
# the whole thing up as a single "set" statement.
#
# This will of course break if any of these variables contains a newline or
# an unmatched quote.
#

eval "set -- $(
        printf '%s\n' "$DEFAULT_JVM_OPTS $JAVA_OPTS $GRADLE_OPTS" |
        xargs -n1 |
        sed ' s~[^-[:alnum:]+,./:=@_]~\\&~g; ' |
        tr '\n' ' '
    )" '"$@"'

exec "$JAVACMD" "$@"
```

Create `android/gradlew.bat` (Windows script):
```batch
@rem
@rem Copyright 2015 the original author or authors.
@rem
@rem Licensed under the Apache License, Version 2.0 (the "License");
@rem you may not use this file except in compliance with the License.
@rem You may obtain a copy of the License at
@rem
@rem      https://www.apache.org/licenses/LICENSE-2.0
@rem
@rem Unless required by applicable law or agreed to in writing, software
@rem distributed under the License is distributed on an "AS IS" BASIS,
@rem WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
@rem See the License for the specific language governing permissions and
@rem limitations under the License.
@rem

@if "%DEBUG%" == "" @echo off
@rem ##########################################################################
@rem
@rem  Gradle startup script for Windows
@rem
@rem ##########################################################################

@rem Set local scope for the variables with windows NT shell
if "%OS%"=="Windows_NT" setlocal

set DIRNAME=%~dp0
if "%DIRNAME%" == "" set DIRNAME=.
set APP_BASE_NAME=%~n0
set APP_HOME=%DIRNAME%

@rem Resolve any "." and ".." in APP_HOME to make it shorter.
for %%i in ("%APP_HOME%") do set APP_HOME=%%~fi

@rem Add default JVM options here. You can also use JAVA_OPTS and GRADLE_OPTS to pass JVM options to this script.
set DEFAULT_JVM_OPTS="-Xmx64m" "-Xms64m"

@rem Find java.exe
if defined JAVA_HOME goto findJavaFromJavaHome

set JAVA_EXE=java.exe
%JAVA_EXE% -version >NUL 2>&1
if "%ERRORLEVEL%" == "0" goto execute

echo.
echo ERROR: JAVA_HOME is not set and no 'java' command could be found in your PATH.
echo.
echo Please set the JAVA_HOME variable in your environment to match the
echo location of your Java installation.

goto fail

:findJavaFromJavaHome
set JAVA_HOME=%JAVA_HOME:"=%
set JAVA_EXE=%JAVA_HOME%/bin/java.exe

if exist "%JAVA_EXE%" goto execute

echo.
echo ERROR: JAVA_HOME is set to an invalid directory: %JAVA_HOME%
echo.
echo Please set the JAVA_HOME variable in your environment to match the
echo location of your Java installation.

goto fail

:execute
@rem Setup the command line

set CLASSPATH=%APP_HOME%\gradle\wrapper\gradle-wrapper.jar


@rem Execute Gradle
"%JAVA_EXE%" %DEFAULT_JVM_OPTS% %JAVA_OPTS% %GRADLE_OPTS% "-Dorg.gradle.appname=%APP_BASE_NAME%" -classpath "%CLASSPATH%" org.gradle.wrapper.GradleWrapperMain %*

:end
@rem End local scope for the variables with windows NT shell
if "%ERRORLEVEL%"=="0" goto mainEnd

:fail
rem Set variable GRADLE_EXIT_CONSOLE if you need the _script_ return code instead of
rem the _cmd.exe /c_ return code!
if  not "" == "%GRADLE_EXIT_CONSOLE%" exit 1
exit /b 1

:mainEnd
if "%OS%"=="Windows_NT" endlocal

:omega
```

## Phase 7: Testing and Validation

### 7.1 Integration Testing Plan

1. **API Compatibility Testing**: Ensure Android app works with current desktop API
2. **Shared Constants Testing**: Verify constants are correctly shared between platforms
3. **Build System Testing**: Validate that both desktop and Android builds work
4. **Cross-Platform Testing**: Test that desktop updates don't break Android functionality

### 7.2 Continuous Integration Setup

Add to `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    # Desktop build
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
        
    - name: Build desktop
      run: go run build.go build syncthing
    
    # Android build
    - name: Set up JDK 11
      uses: actions/setup-java@v3
      with:
        java-version: '11'
        distribution: 'temurin'
        
    - name: Build Android
      run: |
        cd android
        ./gradlew build
```

## Workflow Benefits

With this integrated approach:

1. **When you update desktop APIs**, the changes are immediately visible in the same codebase
2. **Shared constants** ensure API endpoints and data models stay in sync
3. **Single version management** means desktop and Android versions are always compatible
4. **Unified testing** can verify both platforms work together
5. **Streamlined release process** allows simultaneous desktop and Android releases

## Development Workflow

1. Make API changes in desktop version
2. Update shared constants if needed
3. Update Android app to use new APIs
4. Test both desktop and Android together
5. Release both versions simultaneously

This approach ensures that whenever you update the desktop version, the Android app can immediately take advantage of new features while maintaining compatibility.