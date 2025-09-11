# Update Android App to Match Desktop App

## Overview

This document outlines the plan to update the Android Syncthing app to match the feature set and functionality of the desktop version. The goal is to ensure feature parity between the two platforms while maintaining the mobile-specific user experience.

Currently, the Android app implements only basic functionality:
- System status monitoring
- Version checking and compatibility verification
- Basic REST API communication with the desktop daemon

The desktop version provides a comprehensive set of features through its REST API that are not yet available in the Android app.

## Architecture

The Android app follows a standard MVVM architecture with the following components:

1. **View Layer**:
   - Activities and Fragments for UI presentation
   - XML layout files defining the user interface
   - Navigation between different sections of the app

2. **ViewModel Layer**:
   - MainViewModel for handling UI-related data
   - Data binding between View and business logic
   - State management for loading, error, and success states

3. **Repository Layer**:
   - SyncthingRepository for data access
   - API service interface abstraction
   - Data transformation and validation

4. **Data Layer**:
   - REST API client using Retrofit
   - Data models for API responses
   - Local data persistence with Room database

5. **Service Layer**:
   - Background services for periodic tasks
   - Version compatibility checking
   - Notification management

## API Endpoints Reference

The desktop Syncthing exposes a comprehensive REST API with the following endpoint categories:

### System Endpoints
- `/rest/system/status` - Get system status information
- `/rest/system/config` - Get system configuration
- `/rest/system/connections` - Get connection information
- `/rest/system/shutdown` - Shutdown the Syncthing daemon
- `/rest/system/restart` - Restart the Syncthing daemon
- `/rest/system/version` - Get version information
- `/rest/system/browse` - Browse file system paths
- `/rest/system/discovery` - Get discovery information
- `/rest/system/error` - Get or clear system errors
- `/rest/system/paths` - Get system paths
- `/rest/system/ping` - Ping the system
- `/rest/system/log` - Get system logs
- `/rest/system/log.txt` - Get system logs as text
- `/rest/system/loglevels` - Get or set log levels
- `/rest/system/upgrade` - Check for upgrades

### Database Endpoints
- `/rest/db/status` - Get database status for a folder
- `/rest/db/browse` - Browse folder contents
- `/rest/db/need` - Get needed files for a folder
- `/rest/db/remoteneed` - Get needed files for a remote device
- `/rest/db/localchanged` - Get locally changed files
- `/rest/db/file` - Get file information
- `/rest/db/ignores` - Get or set ignore patterns
- `/rest/db/prio` - Set file priority
- `/rest/db/override` - Override folder changes
- `/rest/db/revert` - Revert folder changes
- `/rest/db/scan` - Scan folder for changes

### Configuration Endpoints
- `/rest/config` - Get or update complete configuration
- `/rest/config/insync` - Check if configuration is in sync
- `/rest/config/restart-required` - Check if restart is required
- `/rest/config/folders` - Get or update all folders
- `/rest/config/folders/:id` - Get or update specific folder
- `/rest/config/devices` - Get or update all devices
- `/rest/config/devices/:id` - Get or update specific device
- `/rest/config/options` - Get or update options
- `/rest/config/gui` - Get or update GUI configuration
- `/rest/config/ldap` - Get or update LDAP configuration
- `/rest/config/defaults/folder` - Get or update default folder settings
- `/rest/config/defaults/device` - Get or update default device settings
- `/rest/config/defaults/ignores` - Get or update default ignore patterns

### Cluster Management Endpoints
- `/rest/cluster/pending/devices` - Manage pending devices
- `/rest/cluster/pending/folders` - Manage pending folders

### Statistics Endpoints
- `/rest/stats/device` - Get device statistics
- `/rest/stats/folder` - Get folder statistics

### Events Endpoints
- `/rest/events` - Get events stream
- `/rest/events/disk` - Get disk events stream

### Folder Management Endpoints
- `/rest/folder/versions` - Get or restore file versions
- `/rest/folder/errors` - Get folder errors

### Service Endpoints
- `/rest/svc/deviceid` - Get device ID from string
- `/rest/svc/lang` - Get language information
- `/rest/svc/report` - Get usage report
- `/rest/svc/random/string` - Generate random string

## Data Models & Database Mapping

The Android app uses data models that mirror the desktop API responses. These models are persisted using Room database for local data storage:

1. **System Models**:
   - SystemStatus - Contains system resource usage and status
   - SystemVersion - Contains version information including codename and build details
   - SystemConfig - Contains global configuration settings

2. **Folder Models**:
   - Folder configuration entities with all available options
   - Folder status and statistics
   - Folder browse results
   - Folder need lists

3. **Device Models**:
   - Device configuration entities
   - Connection status and statistics
   - Pending device requests

4. **Event Models**:
   - Event data structures for real-time updates
   - Disk event tracking

## Business Logic Layer

### Current Implementation

The Android app currently implements:
1. Basic system status monitoring through `/rest/system/status` and `/rest/system/version`
2. Version compatibility checking between Android app and desktop daemon
3. Notification system for version updates
4. API constants synchronization with desktop version

### Missing Features to Implement

#### Configuration Management
- Full folder configuration editing (currently only basic models exist)
- Device management interface
- Global options configuration
- GUI settings management
- LDAP configuration support

#### Advanced Synchronization Features
- Folder versioning controls
- Ignore pattern management interface
- Custom scan scheduling
- Bandwidth limiting configuration
- Path browsing capabilities

#### Monitoring & Diagnostics
- Connection statistics viewing
- Device statistics dashboard
- Folder error reporting
- System logs interface
- Performance metrics visualization

#### Cluster Management
- Pending devices management
- Pending folders management
- Discovery services configuration

#### System Management
- Restart/shutdown controls
- Upgrade checking and installation
- Configuration backup/restore
- Log level management

## Middleware & Interceptors

1. **Authentication**:
   - API key management and injection
   - Session handling for authenticated requests
   - CSRF protection for state-changing operations

2. **Network Management**:
   - Request/response logging for debugging
   - Error handling and retry mechanisms
   - Timeout and connection management
   - Offline mode handling

3. **Data Processing**:
   - Response caching for better performance
   - Data transformation between API and UI models
   - Validation of input data before API calls

## Testing

### Unit Tests
1. **API Client Tests**:
   - Mocked API responses for all endpoints
   - Error handling scenarios (network errors, authentication failures)
   - Data model serialization/deserialization

2. **Business Logic Tests**:
   - Version parsing and comparison logic
   - Compatibility matrix validation
   - Feature support verification

3. **Data Model Tests**:
   - Model creation and validation
   - Data transformation logic
   - Edge case handling

### Integration Tests
1. **REST API Integration**:
   - Live API communication testing (where possible)
   - Configuration synchronization workflows
   - Real-time event handling

2. **Service Tests**:
   - Background version checking service
   - Notification system integration
   - Data persistence layer

### UI Tests
1. **Screen Navigation Tests**:
   - Activity and fragment transitions
   - Configuration screen workflows
   - Settings management flows

2. **User Interaction Tests**:
   - Form input validation
   - Error state handling
   - Responsive design across device sizes

## Implementation Roadmap

### Phase 1: Core Configuration Management (2 weeks)
- Implement missing API endpoints in SyncthingApiServiceInterface
- Create data models for all configuration entities
- Build repository methods for configuration management
- Create basic UI for folder and device management

### Phase 2: Advanced Synchronization Features (3 weeks)
- Implement folder versioning controls
- Add ignore pattern management interface
- Create bandwidth limiting configuration UI
- Implement path browsing capabilities

### Phase 3: Monitoring & Diagnostics (2 weeks)
- Add system logs viewing interface
- Implement statistics dashboards
- Create error reporting features
- Add connection quality metrics

### Phase 4: Cluster & System Management (2 weeks)
- Implement pending devices/folders management
- Add system restart/shutdown controls
- Create upgrade checking interface
- Implement configuration backup/restore

### Phase 5: Polish & Optimization (1 week)
- UI/UX improvements based on user feedback
- Performance optimizations
- Comprehensive testing and bug fixes
- Documentation updates

## TODO Tracking

To systematically track the progress of this implementation according to development best practices:

1. **Pre-Implementation Analysis**:
   - Analyze existing codebase to identify implemented vs. missing features
   - Verify extension compatibility before starting new development
   - Create detailed task breakdown for each phase

2. **Continuous Integration**:
   - Maintain and update the development plan to reflect current status
   - Track technical debt items and unused functions/parameters/constants
   - Perform regular codebase cleanup (remove backup files, temporary directories, redundant comments)

3. **Testing-First Approach**:
   - Follow test-driven development (TDD) for all new features
   - Write unit tests before implementation
   - Ensure comprehensive test coverage (target 90%+) across unit, integration, and UI tests
   - Include integration tests that verify complete flow between components

4. **Progress Monitoring**:
   - Update task status only after full verification of completion
   - Maintain systematic workflow with regular commit preparation
   - Write comprehensive commit messages summarizing improvements
   - Conduct thorough code reviews before merging

