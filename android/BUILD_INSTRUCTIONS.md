# Android App Build Instructions

## Prerequisites
- Android Studio or Gradle installed
- JDK 24 or higher
- Android SDK with API level 34

## Building the App

### Build Debug APK
```bash
cd android
./gradlew assembleDebug
```

The debug APK will be generated at:
`app/build/outputs/apk/debug/app-debug.apk`

### Build Release APK
```bash
cd android
./gradlew assembleRelease
```

### Run Unit Tests
```bash
cd android
./gradlew test
```

### Run Connected Tests (requires connected device/emulator)
```bash
cd android
./gradlew connectedDebugAndroidTest
```

## Project Structure
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

## Key Components

### Main Activities
- [MainActivity]: Initial screen for API key entry
- [NavigationActivity]: Main navigation hub with bottom navigation

### Fragments
- [DashboardFragment]: System status, connections, and events monitoring
- [FoldersFragment]: Folder configuration management
- [DevicesFragment]: Device configuration management
- [SettingsFragment]: General settings and system controls

### Data Layer
- [SyncthingApiService]: Retrofit interface for Syncthing REST API
- [SyncthingRepository]: Repository pattern implementation
- Data models in `com.syncthing.android.data.api.model`

### ViewModel
- [MainViewModel]: Central ViewModel for managing app state

## Dependencies
- Kotlin 2.2.0
- AndroidX libraries
- Retrofit 3.0.0 for REST API communication
- Gson for JSON serialization
- Mockito for unit testing
- Material Design components

## Troubleshooting

### Kotlin Version Issues
If you encounter Kotlin version compatibility errors, ensure you're using a compatible Gradle version.

### Missing Drawable Resources
If the app fails to build due to missing drawable resources, ensure all icon files are present in `app/src/main/res/drawable/`:
- ic_dashboard.xml
- ic_folder.xml
- ic_device.xml
- ic_settings.xml

### Test Compilation Errors
If tests fail to compile, ensure Mockito dependencies are correctly specified in `app/build.gradle`.