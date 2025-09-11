# Android-Desktop Synchronization Update Strategy

## Overview

This document outlines a strategy to keep the Syncthing Android app synchronized with the desktop version without requiring structural changes to either application. The approach leverages existing mechanisms and enhances them to ensure seamless compatibility and automatic updates.

## Architecture

The synchronization system is built around three core components:

``mermaid
graph TD
    A[Android App] -->|API Constants Sync| B[Shared Constants Generator]
    B -->|Generates| C[ApiConstants.kt]
    A -->|Version Check| D[Version Compatibility Service]
    D -->|REST API| E[Desktop Syncthing]
    A -->|Background Updates| F[Periodic Worker]
    F -->|Notification| A
```

### Core Components

1. **Shared Constants Generator**
   - Automatically extracts API constants from the desktop application
   - Generates corresponding Kotlin constants for the Android app
   - Ensures API endpoint consistency between platforms

2. **Version Compatibility Service**
   - Checks desktop version compatibility with Android app
   - Provides update recommendations based on version differences
   - Implements semantic versioning comparison logic

3. **Periodic Update Worker**
   - Runs background version checks at regular intervals
   - Triggers notifications when updates are available
   - Respects device constraints (battery, network)

## Synchronization Mechanisms

### 1. API Constants Synchronization

The Android app stays in sync with desktop API changes through automatically generated constants:

``mermaid
flowchart LR
    A[lib/api/constants.go] --> B[Script Execution]
    B --> C[ApiConstants.kt Generation]
    C --> D[Android App Integration]
```

#### Process:
1. Desktop API constants are defined in `lib/api/constants.go`
2. A generation script (`script/generate-android-constants.go`) converts Go constants to Kotlin
3. Generated constants are stored in `android/app/src/main/java/com/syncthing/android/util/ApiConstants.kt`
4. Android app uses these constants for all API interactions

#### Benefits:
- Eliminates manual synchronization errors
- Ensures API endpoint consistency
- Reduces maintenance overhead

### 2. Version Compatibility Checking

The system implements a sophisticated version compatibility checking mechanism:

``mermaid
sequenceDiagram
    participant A as Android App
    participant V as VersionCheckService
    participant D as Desktop Syncthing
    participant C as VersionCompatibilityChecker
    
    A->>V: Trigger version check
    V->>D: GET /rest/system/version
    D-->>V: SystemVersion response
    V->>C: Check compatibility
    C-->>V: CompatibilityResult
    V->>A: Show notification (if needed)
```

#### Components:
- **VersionCheckService**: Fetches desktop version via REST API
- **VersionCompatibilityChecker**: Compares versions and determines compatibility
- **VersionCheckWorker**: Schedules periodic checks and immediate checks

#### Version Comparison Logic:
- Parses semantic version strings (major.minor.patch)
- Compares version components hierarchically
- Identifies major version mismatches
- Recommends updates when desktop version is newer

### 3. Background Update Notifications

The system uses Android WorkManager for efficient background processing:

``kotlin
// Scheduling periodic checks
val constraints = Constraints.Builder()
    .setRequiredNetworkType(NetworkType.UNMETERED)
    .setRequiresBatteryNotLow(true)
    .build()

val versionCheckRequest = PeriodicWorkRequestBuilder<VersionCheckService>(
    24, TimeUnit.HOURS
)
    .setConstraints(constraints)
    .addTag(WORK_TAG)
    .build()
```

#### Features:
- Runs every 24 hours
- Respects device constraints (network, battery)
- Can be triggered manually
- Shows notifications for recommended updates

## Implementation Details

### Constant Generation Process

The synchronization process begins with the constant generation script:

1. **Input**: `lib/api/constants.go` containing all API endpoints and configuration values
2. **Processing**: 
   - Script parses Go constant declarations
   - Converts naming conventions (CamelCase to UPPER_SNAKE_CASE)
   - Handles special cases through a mapping table
3. **Output**: `ApiConstants.kt` with equivalent Kotlin constants

### Version Checking Workflow

1. **Trigger Points**:
   - App startup
   - Manual user action
   - Periodic background check (every 24 hours)

2. **Execution Flow**:
   - Fetch desktop version via `/rest/system/version` endpoint
   - Compare with Android app version
   - Determine compatibility status
   - Generate appropriate user notification

3. **API Interaction**:
   ```kotlin
   val url = URL("http://localhost:8384/rest/system/version")
   val connection = url.openConnection() as HttpURLConnection
   // Process response and check compatibility
   ```

### Compatibility Assessment

The system evaluates several factors when determining compatibility:

| Factor | Assessment |
|--------|------------|
| Major Version Match | Critical for compatibility |
| Minor Version Difference | Feature availability |
| Patch Version Difference | Bug fixes only |
| API Endpoint Availability | Based on shared constants |
| Feature Support | Version-based feature flags |

## Enhancement Recommendations

### 1. Automatic Update Mechanism

Implement automatic downloading and installation of Android app updates:

``mermaid
flowchart TD
    A[Version Check] --> B{Update Available?}
    B -->|Yes| C[Download APK]
    C --> D[Verify Signature]
    D --> E[Install Update]
    B -->|No| F[Continue]
```

### 2. Feature Compatibility Matrix

Create a detailed matrix tracking feature availability across versions:

| Feature | Android 1.0 | Desktop 1.0 | Android 1.2 | Desktop 1.2 |
|---------|-------------|-------------|-------------|-------------|
| Basic Sync | ✓ | ✓ | ✓ | ✓ |
| Versioning | ✓ | ✓ | ✓ | ✓ |
| Advanced Ignore | ✗ | ✓ | ✓ | ✓ |

### 3. Graceful Degradation

Implement automatic disabling of unsupported features:

```kotlin
if (isFeatureSupported(feature, desktopVersion)) {
    enableFeature()
} else {
    disableFeature()
    showCompatibilityMessage()
}
```

### 4. API Contract Testing

Add automated tests to verify API compatibility between versions:

``mermaid
graph LR
    A[Test Suite] --> B{API Contract}
    B --> C[Android Client]
    B --> D[Desktop Server]
    C --> E[Validation]
    D --> E
    E --> F[Compatibility Report]
```

## Testing Strategy

### Version Synchronization Testing

1. **Cross-Version Testing**:
   - Test with older Android app and newer desktop
   - Test with newer Android app and older desktop
   - Validate compatibility assessment logic

2. **Network Failure Scenarios**:
   - Simulate connection timeouts
   - Test offline behavior
   - Verify retry mechanisms

3. **Notification Testing**:
   - Validate notification display
   - Test user interaction with notifications
   - Check background service behavior

### Security Considerations

1. **Encrypted Communication**:
   - All version checks use HTTPS connections
   - API keys are securely stored
   - Certificate validation is enforced

2. **Update Verification**:
   - APK signatures are verified before installation
   - Updates are downloaded from official sources only
   - Version information is validated before use

## Future Enhancements

### 1. Smart Update Scheduling

Implement intelligent update scheduling based on:
- User activity patterns
- Network conditions
- Battery status
- Device usage

### 2. Incremental Update Support

Enable delta updates to reduce bandwidth usage:
- Download only changed components
- Apply patches instead of full updates
- Support for partial feature updates

### 3. Cross-Platform Feature Parity

Establish mechanisms to:
- Track feature availability across platforms
- Notify users of platform-specific limitations
- Provide alternative workflows for missing features

## Conclusion

This synchronization strategy ensures the Android app remains compatible with the desktop version through automated processes that require minimal structural changes. By leveraging existing mechanisms and enhancing them with intelligent version checking and notification systems, users can maintain up-to-date installations without manual intervention.