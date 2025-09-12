Android App Development
=======================

Architecture Overview
---------------------

The Syncthing Android app is built using modern Android development practices with the following key components:

1. **MVVM Architecture**: The app follows the Model-View-ViewModel pattern for clean separation of concerns
2. **Retrofit for API Communication**: Uses Retrofit library for REST API communication with the Syncthing server
3. **Kotlin**: Written in Kotlin for modern, concise code
4. **Android Jetpack Components**: Uses Android Jetpack libraries for lifecycle management and UI components
5. **Repository Pattern**: Implements repository pattern for data management

Communication with Desktop Version via REST API
-----------------------------------------------

The Android app communicates with the Syncthing desktop version through the same REST API that the web interface uses. This ensures consistency and feature parity between platforms.

Key aspects of the communication:

1. **Authentication**: Uses API key authentication via the X-API-Key header
2. **Endpoints**: Consumes the same REST endpoints as the web interface
3. **Data Models**: Uses Kotlin data classes that mirror the JSON responses from the API
4. **Error Handling**: Implements proper error handling for network failures and API errors

Shared Constants and Synchronization
------------------------------------

To ensure the Android app stays synchronized with API changes in the desktop version, a constant synchronization mechanism is in place:

1. **Automatic Generation**: A script (``script/generate-android-constants.go``) automatically extracts constants from ``lib/api/constants.go``
2. **Generated File**: The script generates ``android/app/src/main/java/com/syncthing/android/util/ApiConstants.kt``
3. **Regeneration Process**: This should be run whenever the desktop API constants change

To regenerate the constants:
```bash
go run script/generate-android-constants.go
```

Building and Testing the Android App
------------------------------------

Prerequisites
~~~~~~~~~~~~~

- Android Studio or Gradle installed
- JDK 24 or higher
- Android SDK with API level 34

Building the App
~~~~~~~~~~~~~~~~

Debug APK
+++++++++

```bash
cd android
./gradlew assembleDebug
```

The debug APK will be generated at:
``app/build/outputs/apk/debug/app-debug.apk``

Release APK
+++++++++++

```bash
cd android
./gradlew assembleRelease
```

Testing
~~~~~~~

Unit Tests
++++++++++

```bash
cd android
./gradlew test
```

Connected Tests (requires connected device/emulator)
+++++++++++++++++++++++++++++++++++++++++++++++++++

```bash
cd android
./gradlew connectedDebugAndroidTest
```

Project Structure
~~~~~~~~~~~~~~~~~

```
android/
├── app/
│   ├── src/
│   │   ├── main/
│   │   │   ├── java/com/syncthing/android/
│   │   │   ├── res/
│   │   │   └── AndroidManifest.xml
│   │   └── test/
│   └── build.gradle
├── build.gradle
├── gradle.properties
└── settings.gradle
```

Key Components
~~~~~~~~~~~~~~

Main Activities
+++++++++++++++

- [MainActivity]: Initial screen for API key entry
- [NavigationActivity]: Main navigation hub with bottom navigation

Fragments
+++++++++

- [DashboardFragment]: System status, connections, and events monitoring
- [FoldersFragment]: Folder configuration management
- [DevicesFragment]: Device configuration management
- [SettingsFragment]: General settings and system controls

Data Layer
++++++++++

- [SyncthingApiService]: Retrofit interface for Syncthing REST API
- [SyncthingRepository]: Repository pattern implementation
- Data models in ``com.syncthing.android.data.api.model``

ViewModel
+++++++++

- [MainViewModel]: Central ViewModel for managing app state

Dependencies
~~~~~~~~~~~~

- Kotlin 2.2.0
- AndroidX libraries
- Retrofit 3.0.0 for REST API communication
- Gson for JSON serialization
- Mockito for unit testing
- Material Design components

Version Compatibility Matrix
----------------------------

The system implements a feature compatibility matrix that tracks which features are available with which versions:

+----------------------+---------------------+---------------------+---------------------------------+
| Feature              | Android Min Version | Desktop Min Version | Description                     |
+======================+=====================+=====================+=================================+
| Basic Sync           | 1.0.0               | 1.0.0               | Core file synchronization       |
+----------------------+---------------------+---------------------+---------------------------------+
| Versioning           | 1.0.0               | 1.0.0               | Basic file versioning           |
+----------------------+---------------------+---------------------+---------------------------------+
| Advanced Ignore      | 1.2.0               | 1.2.0               | Advanced ignore patterns        |
+----------------------+---------------------+---------------------+---------------------------------+
| External Versioning  | 1.1.0               | 1.1.0               | External versioning scripts     |
+----------------------+---------------------+---------------------+---------------------------------+
| Custom Discovery     | 1.0.0               | 1.0.0               | Custom discovery servers        |
+----------------------+---------------------+---------------------+---------------------------------+
| Bandwidth Limits     | 1.1.0               | 1.1.0               | Bandwidth rate limiting         |
+----------------------+---------------------+---------------------+---------------------------------+

Version Synchronization Mechanism
---------------------------------

The Android app implements a version synchronization mechanism to ensure compatibility with the desktop version of Syncthing:

1. **Periodic Version Checking**: Checks the desktop Syncthing version through the ``/rest/system/version`` endpoint
2. **Compatibility Verification**: Compares Android app version with desktop version
3. **Update Notifications**: Shows notifications when updates are available or compatibility issues detected
4. **Graceful Degradation**: Manages graceful degradation of unsupported features
5. **Automatic Updates**: Handles automatic downloading and installation of Android app updates

Implementation Details
~~~~~~~~~~~~~~~~~~~~~~

Version Checking
++++++++++++++++

The Android app periodically checks the desktop Syncthing version through the ``/rest/system/version`` endpoint. This is done:

1. When the app starts
2. Periodically in the background (every 24 hours)
3. When the user manually triggers a check

Compatibility Checking
++++++++++++++++++++++

The app compares the Android app version with the desktop version to determine:

1. If they are compatible
2. If an update is recommended
3. If there are any breaking changes
4. Which features are supported

Graceful Degradation
++++++++++++++++++++

When features are not supported due to version incompatibility:

1. The app automatically disables unsupported features
2. Users are notified about disabled features
3. Alternative workflows are provided when possible

Testing Guidelines
------------------

The Android app should be tested with:

1. Different version combinations (older Android, newer desktop and vice versa)
2. Network failure scenarios
3. Notification display and interaction
4. Background service behavior
5. API contract verification
6. Feature compatibility matrix validation
7. Update mechanism functionality

Unit Tests
~~~~~~~~~~

- Version parsing and comparison
- Compatibility checking logic
- Feature support verification
- API constant validation

Integration Tests
~~~~~~~~~~~~~~~~~

- REST API communication
- Background worker functionality
- Notification system
- Update mechanism

UI Tests
~~~~~~~~

- Notification display and interaction
- Feature enablement/disablement
- User flows for update handling

Security Considerations
-----------------------

1. All version checks use encrypted HTTPS connections
2. API keys are securely stored
3. Version information is validated before use
4. Update notifications link to official sources only
5. APK signatures are verified before installation
6. Updates are downloaded from official sources only