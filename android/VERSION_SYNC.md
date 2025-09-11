# Syncthing Android Version Synchronization

This document describes how the Android app stays synchronized with the desktop version of Syncthing.

## Overview

The Android app implements a version synchronization mechanism to ensure compatibility with the desktop version of Syncthing. This includes:

1. Periodic version checking
2. Compatibility verification
3. Update notifications
4. Automatic API constant synchronization
5. Feature compatibility matrix
6. Graceful degradation for unsupported features
7. Automatic update mechanism

## Components

### 1. Data Models

- `SystemVersion`: Represents the version information returned by the desktop Syncthing API
- `VersionCompatibilityResult`: Contains the result of version compatibility checking

### 2. Services

- `VersionCheckService`: Handles version comparison logic and REST API communication
- `VersionCheckWorker`: Background worker for periodic version checking with flexible scheduling
- `VersionNotificationService`: Shows notifications when updates are available or compatibility issues detected
- `AutoUpdateService`: Handles automatic downloading and installation of Android app updates
- `FeatureDegradationService`: Manages graceful degradation of unsupported features

### 3. Utilities

- `ApiConstants`: Shared constants between desktop and Android versions
- `VersionCompatibilityChecker`: Sophisticated version compatibility checking logic
- `FeatureCompatibilityMatrix`: Tracks feature availability across versions

## Implementation Details

### Version Checking

The Android app periodically checks the desktop Syncthing version through the `/rest/system/version` endpoint. This is done:

1. When the app starts
2. Periodically in the background (every 24 hours)
3. When the user manually triggers a check

The system uses Retrofit for API communication and handles API keys securely.

### Compatibility Checking

The app compares the Android app version with the desktop version to determine:

1. If they are compatible
2. If an update is recommended
3. If there are any breaking changes
4. Which features are supported

The compatibility checker considers:
- Major version compatibility (critical)
- Minor version differences (feature availability)
- Patch version differences (bug fixes)
- API endpoint availability based on shared constants
- Feature support based on version

### Update Notifications

When a newer desktop version is detected, the app shows a notification to the user with:

1. The current desktop version
2. The current Android app version
3. A recommendation to update
4. Codename and status information (beta, release candidate)

Different notification types are used for updates vs. compatibility issues.

### API Constant Synchronization

To ensure the Android app stays in sync with API changes in the desktop version:

1. A script (`script/generate-android-constants.go`) automatically extracts constants from `lib/api/constants.go`
2. The script generates `android/app/src/main/java/com/syncthing/android/util/ApiConstants.kt`
3. This should be run whenever the desktop API constants change

To regenerate the constants:
```bash
go run script/generate-android-constants.go
```

### Feature Compatibility Matrix

The system implements a feature compatibility matrix that tracks which features are available with which versions:

| Feature | Android Min Version | Desktop Min Version | Description |
|---------|---------------------|---------------------|-------------|
| Basic Sync | 1.0.0 | 1.0.0 | Core file synchronization |
| Versioning | 1.0.0 | 1.0.0 | Basic file versioning |
| Advanced Ignore | 1.2.0 | 1.2.0 | Advanced ignore patterns |
| External Versioning | 1.1.0 | 1.1.0 | External versioning scripts |
| Custom Discovery | 1.0.0 | 1.0.0 | Custom discovery servers |
| Bandwidth Limits | 1.1.0 | 1.1.0 | Bandwidth rate limiting |

### Graceful Degradation

When features are not supported due to version incompatibility:

1. The app automatically disables unsupported features
2. Users are notified about disabled features
3. Alternative workflows are provided when possible

### Automatic Updates

The system can automatically download and install Android app updates:

1. Checks for new versions from official sources
2. Downloads updates securely
3. Verifies update signatures
4. Installs updates with user consent

## Future Enhancements

1. **Smart Update Scheduling**: Implement intelligent update scheduling based on user activity patterns, network conditions, and battery status
2. **Incremental Update Support**: Enable delta updates to reduce bandwidth usage
3. **Cross-Platform Feature Parity**: Establish mechanisms to track feature availability across platforms
4. **Enhanced Security**: Add additional security measures for update verification

## Testing

The version synchronization feature should be tested with:

1. Different version combinations (older Android, newer desktop and vice versa)
2. Network failure scenarios
3. Notification display and interaction
4. Background service behavior
5. API contract verification
6. Feature compatibility matrix validation
7. Update mechanism functionality

### Test Categories

1. **Unit Tests**: 
   - Version parsing and comparison
   - Compatibility checking logic
   - Feature support verification
   - API constant validation

2. **Integration Tests**:
   - REST API communication
   - Background worker functionality
   - Notification system
   - Update mechanism

3. **UI Tests**:
   - Notification display and interaction
   - Feature enablement/disablement
   - User flows for update handling

## Security Considerations

1. All version checks use encrypted HTTPS connections
2. API keys are securely stored
3. Version information is validated before use
4. Update notifications link to official sources only
5. APK signatures are verified before installation
6. Updates are downloaded from official sources only