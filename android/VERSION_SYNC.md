# Syncthing Android Version Synchronization

This document describes how the Android app stays synchronized with the desktop version of Syncthing.

## Overview

The Android app implements a version synchronization mechanism to ensure compatibility with the desktop version of Syncthing. This includes:

1. Periodic version checking
2. Compatibility verification
3. Update notifications
4. Automatic API constant synchronization

## Components

### 1. Data Models

- `SystemVersion`: Represents the version information returned by the desktop Syncthing API
- `VersionCompatibilityResult`: Contains the result of version compatibility checking

### 2. Services

- `VersionCheckService`: Handles version comparison logic
- `VersionCheckWorker`: Background worker for periodic version checking
- `VersionNotificationService`: Shows notifications when updates are available

### 3. Utilities

- `ApiConstants`: Shared constants between desktop and Android versions
- `VersionCompatibilityChecker`: Sophisticated version compatibility checking logic

## Implementation Details

### Version Checking

The Android app periodically checks the desktop Syncthing version through the `/rest/system/version` endpoint. This is done:

1. When the app starts
2. Periodically in the background (every 24 hours)
3. When the user manually triggers a check

### Compatibility Checking

The app compares the Android app version with the desktop version to determine:

1. If they are compatible
2. If an update is recommended
3. If there are any breaking changes

### Update Notifications

When a newer desktop version is detected, the app shows a notification to the user with:

1. The current desktop version
2. The current Android app version
3. A recommendation to update

## API Constant Synchronization

To ensure the Android app stays in sync with API changes in the desktop version:

1. A script (`script/generate-android-constants.go`) automatically extracts constants from `lib/api/constants.go`
2. The script generates `android/app/src/main/java/com/syncthing/android/util/ApiConstants.kt`
3. This should be run whenever the desktop API constants change

To regenerate the constants:
```bash
go run script/generate-android-constants.go
```

## Future Enhancements

1. **Automatic Updates**: Implement automatic downloading and installation of Android app updates
2. **Feature Compatibility Matrix**: Track which features are available in which versions
3. **Graceful Degradation**: Automatically disable unsupported features instead of showing errors
4. **API Contract Testing**: Automated tests to verify API compatibility between versions

## Testing

The version synchronization feature should be tested with:

1. Different version combinations (older Android, newer desktop and vice versa)
2. Network failure scenarios
3. Notification display and interaction
4. Background service behavior

## Security Considerations

1. All version checks use encrypted HTTPS connections
2. API keys are securely stored
3. Version information is validated before use
4. Update notifications link to official sources only