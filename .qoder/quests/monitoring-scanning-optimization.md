# Monitoring and Scanning Optimization Plan

## 1. Overview

This document outlines a comprehensive plan to optimize folder monitoring and scanning operations in Syncthing to improve performance, reduce resource consumption, and enhance user experience. The optimizations target both the file system watching mechanisms and the periodic folder scanning processes.

## 2. Current Architecture

### 2.1 Folder Scanning
- Periodic scanning based on `RescanIntervalS` configuration (default: 3600 seconds)
- File system walking using the `scanner` package
- Parallel hashing with configurable number of hashers
- Progress reporting through events system

### 2.2 File System Monitoring
- Platform-specific file system watching using the `fs` package
- Event aggregation through `watchaggregator` package
- Configurable delay (`FSWatcherDelayS`) and timeout (`FSWatcherTimeoutS`) settings
- Buffer management for handling file system events

### 2.3 Key Components
- `lib/scanner`: Implements file walking and hashing operations
- `lib/watchaggregator`: Aggregates file system events to reduce scan frequency
- `lib/model`: Manages folder state and coordinates scanning operations
- `lib/fs`: Provides file system abstraction and watching capabilities

## 3. Analysis of Existing Implementation

### 3.1 Selective Subtree Scanning
After analyzing the existing codebase, we found that selective subtree scanning is already partially implemented:

- The `scanner` package includes a `WalkSubtree` function that enables scanning of specific directory subtrees rather than entire folders
- The `model` package has a `scanSubtrees` method that utilizes this functionality
- Directory state caching mechanisms exist in `internal/db/directorystate.go` to track directory changes
- Configuration options for selective scanning already exist (`SelectiveScanningEnabled`)

However, the implementation is incomplete:
- The fallback to full folder scans when inconsistencies are detected is not fully implemented
- The directory state cache is not consistently used across all scanning operations
- API endpoints for triggering selective scans manually are not fully exposed

### 3.2 Adaptive Scan Intervals
Adaptive scanning is also partially implemented:

- The `folder` struct includes activity tracking fields (`lastScanTime`, `fileChangeCount`)
- A `getAdaptiveScanInterval` function exists that calculates scan intervals based on folder activity
- Configuration options exist for adaptive scanning (`AdaptiveScanningEnabled`, `MinScanIntervalS`, `MaxScanIntervalS`, `ScanAggressionLevel`)

The implementation needs enhancement:
- The activity tracking logic could be more sophisticated
- The aggression levels need more fine-tuning
- Integration with the directory state cache for better activity detection is missing

### 3.3 Platform-Specific Optimizations
Platform-specific optimizations have a partial implementation:

- Platform-specific buffer sizes are implemented in `getPlatformOptimalBufferSize()`
- Windows uses a larger buffer (2000) compared to other platforms (500-1000)
- Platform-specific event masks are used for file watching
- Android has specific limitations that prevent file watching in some cases

Missing components:
- Specialized watchers for Windows (ReadDirectoryChangesW), macOS (FSEvents), and Linux (inotify) are not fully implemented
- Dynamic adjustment of `fs.inotify.max_user_watches` for Linux is not implemented
- Android-specific watcher that works within background execution limitations is missing

### 3.4 Adaptive Event Buffering
Adaptive buffer management is partially implemented:

- An `overflowTracker` struct exists in `basicfs_watch.go` that tracks buffer overflow events
- Methods for increasing/decreasing buffer size based on overflow patterns are implemented
- Configuration options exist (`AdaptiveBufferEnabled`, `MinBufferSize`, `MaxBufferSize`, `BufferResizeFactor`)

Areas for improvement:
- The overflow detection logic could be more sophisticated
- Integration with system resource monitoring is missing
- Buffer size adjustments could be more responsive to changing conditions

### 3.5 Smart Event Aggregation
Event aggregation improvements are partially implemented:

- The `watchaggregator` package already implements some event grouping logic
- File type-based timeout strategies exist in `getFileTypeTimeout()`
- Configuration options for aggregation exist (`EventAggregationTimeoutS`)

Enhancement opportunities:
- More sophisticated event grouping algorithms are needed
- Better handling of temporary files and build artifacts
- Improved timeout handling for different event patterns is required

## 4. Optimization Strategies

### 4.1 Intelligent Scanning

#### 4.1.1 Selective Subtree Scanning
Implement targeted scanning of only directories that have changed, rather than full folder scans. Maintain the option for full folder scans as a fallback mechanism.

**Technical Approach:**
- Track modified time of directories to identify changed subtrees using `fs.Filesystem` Lstat capabilities
- Use file system events from the `watchaggregator` to determine which directories need scanning
- Implement a cache of directory states in the database to compare against
- Add configuration option `selectiveScanningEnabled` to enable/disable selective scanning
- Implement fallback to full folder scan when inconsistencies are detected or when cache is corrupted
- Provide API endpoints to trigger full scans manually through the REST interface
- Add metrics to track when selective vs full scans are performed
- Implement subtree scanning in `lib/scanner/walk.go` by adding a new `WalkSubtree` function
- Add directory change tracking in `lib/model/folder.go` to maintain a list of changed directories
- Implement cache validation to ensure consistency between file system and cached states

#### 4.1.2 Adaptive Scan Intervals
Dynamically adjust scan intervals based on folder activity patterns.

**Technical Approach:**
- Monitor file change frequency in each folder
- Increase scan frequency for active folders, decrease for inactive ones
- Implement exponential backoff for folders with no changes

#### 4.1.3 Improved Hashing Performance
Optimize parallel hashing based on available CPU cores and I/O capacity.

**Technical Approach:**
- Dynamically adjust the number of hashers based on system load
- Implement I/O-aware scheduling to prevent disk contention
- Use more efficient hashing algorithms where appropriate

### 4.2 Enhanced Monitoring

#### 4.2.1 Adaptive Event Buffering
Implement dynamic buffer sizing based on event throughput and system performance.

**Technical Approach:**
- Monitor buffer overflow events and adjust buffer size accordingly
- Implement predictive buffer sizing based on historical event patterns
- Add buffer pressure metrics for monitoring and tuning

#### 4.2.2 Smart Event Aggregation
Improve the aggregation logic to better group related events and reduce scan triggers.

**Technical Approach:**
- Implement more sophisticated event grouping algorithms
- Add configurable aggregation policies based on file types
- Optimize timeout handling for different event patterns

#### 4.2.3 Platform-Specific Optimizations
Implement specialized watchers for Windows, macOS, Linux, and Android to leverage platform-specific features for more efficient file system monitoring.

**Technical Approach:**
- **Windows**: Utilize ReadDirectoryChangesW API with overlapped I/O for efficient file system monitoring, implementing in `lib/fs/basicfs_watch.go`
- **macOS**: Leverage FSEvents API for directory-level notifications, with fallback to kqueue when FSEvents is unavailable
- **Linux**: Optimize inotify usage with adaptive watch descriptors based on directory size, implementing dynamic adjustment of `fs.inotify.max_user_watches` (referencing the existing memory about inotify limits configuration)
- **Android**: Implement specialized watcher that works within Android's background execution limitations, using Android-specific APIs and handling app lifecycle events
- Add platform-specific tuning parameters for buffer sizes and event aggregation in `lib/config/optionsconfiguration.go`
- Implement platform detection and automatic selection of appropriate watcher in `lib/fs/basicfs.go`
- Provide fallback to generic polling mechanism when platform-specific APIs fail
- Add platform-specific performance metrics and monitoring
- Implement adaptive buffer sizing based on platform capabilities and limitations

## 5. Detailed Implementation Plan

### 5.1 Phase 1: Infrastructure Improvements (Months 1-2)

#### 5.1.1 Enhanced Metrics Collection
- Add detailed performance metrics for scanning operations
- Implement event processing latency tracking
- Add buffer utilization and overflow monitoring

#### 5.1.2 Adaptive Buffer Management
- Implement the overflowTracker from basicfs_watch.go
- Add dynamic buffer resizing based on overflow frequency
- Add configuration options for buffer management

#### 5.1.3 Performance Profiling
- Add CPU and memory profiling hooks
- Implement performance baselining for comparison
- Add profiling controls to the API

### 5.2 Phase 2: Scanning Optimizations (Months 3-4)

#### 5.2.1 Selective Subtree Implementation
- Develop directory change tracking mechanism in `lib/model/folder.go`
- Implement subtree scanning in the scanner package (`lib/scanner/walk.go`)
- Add APIs for targeted scanning operations in the REST API (`lib/api`)
- Implement directory state cache in the database layer
- Add configuration options for selective scanning
- Implement fallback to full scan when needed

#### 5.2.2 Adaptive Scan Scheduling
- Implement activity-based scan interval adjustment
- Add folder activity monitoring and classification
- Develop exponential backoff algorithms for inactive folders

#### 5.2.3 Hashing Optimization
- Implement dynamic hasher count adjustment
- Add I/O-aware scheduling for hashing operations
- Optimize block queue management

### 5.3 Phase 3: Monitoring Enhancements (Months 5-6)

#### 5.3.1 Advanced Event Aggregation
- Enhance the watchaggregator with smarter grouping logic
- Implement file type-based aggregation policies
- Add configurable timeout strategies

#### 5.3.2 Platform-Specific Optimizations
- Implement specialized watchers for each platform in `lib/fs/`
- Add platform-specific tuning parameters in `lib/config/optionsconfiguration.go`
- Optimize for different file system types (NTFS, ext4, APFS, etc.)
- Implement Android-specific watcher that respects background execution limits
- Add platform detection and automatic selection logic
- Implement fallback mechanisms for each platform
- Add platform-specific performance metrics

#### 5.3.3 Large Folder Handling
- Implement folder size analysis and recommendations
- Add automatic exclusion suggestions for temporary files
- Optimize memory usage for large directory structures

## 6. Configuration Improvements

### 6.1 New Configuration Options

#### 6.1.1 Scanning Configuration
- `adaptiveScanningEnabled`: Enable/disable adaptive scan intervals
- `minScanIntervalS`: Minimum scan interval for active folders
- `maxScanIntervalS`: Maximum scan interval for inactive folders
- `scanAggressionLevel`: Control balance between performance and responsiveness (values: conservative, balanced, aggressive)
- `fallbackScanIntervalS`: Interval for fallback full scans to detect missed changes
- `selectiveScanningEnabled`: Enable/disable selective subtree scanning
- `consistencyCheckIntervalS`: Interval for consistency checks between file system and database

#### 6.1.2 Monitoring Configuration
- `adaptiveBufferEnabled`: Enable/disable adaptive buffer sizing
- `minBufferSize`: Minimum event buffer size
- `maxBufferSize`: Maximum event buffer size
- `bufferResizeFactor`: Factor for buffer size adjustments
- `eventAggregationTimeoutS`: Base timeout for event aggregation
- `platformOptimizationsEnabled`: Enable/disable platform-specific optimizations
- `fallbackPollingIntervalS`: Polling interval when file system watching fails

### 6.2 Backward Compatibility
- All new options have sensible defaults
- Existing configurations continue to work unchanged
- Deprecation warnings for inefficient settings

## 7. Performance Metrics and Monitoring

### 7.1 Key Performance Indicators

#### 7.1.1 Scanning Metrics
- **Scan Duration**: Time to complete folder scans
- **CPU Utilization**: Percentage CPU usage during scans
- **Memory Consumption**: Peak memory usage during scanning
- **Files Processed**: Number of files checked per scan
- **Hashing Efficiency**: Ratio of files hashed to files checked

#### 7.1.2 Monitoring Metrics
- **Event Processing Latency**: Time from file change to scan trigger
- **Buffer Utilization**: Percentage of buffer capacity used
- **Overflow Rate**: Number of buffer overflows per hour
- **False Trigger Rate**: Scans triggered by non-actionable events
- **Missed Events**: File changes not detected within threshold

### 7.2 Monitoring and Evaluation
- Built-in performance metrics collection
- Comparative benchmarking against current implementation
- Real-world testing with large folder structures
- Automated performance regression testing

## 8. Risk Assessment and Mitigation

### 8.1 Potential Risks

#### 8.1.1 Performance Regressions
- Risk: Optimizations may degrade performance in some scenarios
- Mitigation: Extensive testing with various folder sizes and types

#### 8.1.2 Platform Compatibility Issues
- Risk: Platform-specific optimizations may introduce bugs
- Mitigation: Comprehensive cross-platform testing

#### 8.1.3 Missed File Changes
- Risk: More aggressive aggregation may miss important events
- Mitigation: Configurable aggressiveness and fallback scanning
- Implement periodic consistency checks to detect missed changes
- Add manual scan triggers through API for user-initiated verification
- Provide logging and metrics for missed events detection

### 8.2 Rollout Strategy
- Initial release with conservative defaults
- Gradual enablement of more aggressive optimizations
- User feedback collection and iteration

## 9. Testing Strategy

### 9.1 Unit Testing
- Test individual optimization components
- Validate correctness of selective scanning
- Verify event aggregation logic
- Test adaptive buffer resizing algorithms
- Test platform-specific watcher implementations
- Validate fallback mechanisms for each platform

### 9.2 Integration Testing
- Test end-to-end scanning and monitoring workflows
- Validate performance improvements with large datasets
- Ensure compatibility across different platforms
- Test interaction with ignore patterns and versioning
- Test configurable aggressiveness settings
- Validate fallback scanning mechanisms

### 9.3 Performance Testing
- Benchmark CPU and memory usage
- Measure scan completion times
- Evaluate event processing throughput
- Test with various folder sizes (100, 1K, 10K, 100K+ files)
- Test with different file types (documents, images, binaries)
- Validate performance on low-resource devices
- Compare selective vs full scanning performance
- Test with various file system types (NTFS, ext4, APFS, etc.)
- Benchmark platform-specific implementations
- Test under different load conditions (idle, moderate, heavy)

### 9.4 Real-World Testing
- Deploy to beta users with large file collections
- Monitor performance metrics in production
- Collect user feedback on responsiveness
- Compare resource usage before and after optimizations
- Test configurable aggressiveness with different user profiles
- Validate fallback scanning in production environments
- Test cross-platform compatibility with shared folders
- Monitor for regressions in file synchronization accuracy
- Validate behavior with network file systems (NFS, SMB)

## 10. Conclusion

This optimization plan addresses key performance bottlenecks in Syncthing's folder monitoring and scanning systems. By implementing adaptive algorithms, platform-specific optimizations, and smarter event processing, we can significantly improve the user experience while reducing resource consumption. The phased approach ensures careful testing and validation at each step, minimizing the risk of regressions while maximizing the benefits of these improvements.

The analysis of the existing implementation shows that many of the required components are already partially implemented, which reduces the overall development effort. However, completing and integrating these components properly will still require significant work to ensure robustness and optimal performance across all supported platforms.

## 4. Optimization Strategies

### 4.1 Intelligent Scanning

#### 4.1.1 Selective Subtree Scanning
Implement targeted scanning of only directories that have changed, rather than full folder scans. Maintain the option for full folder scans as a fallback mechanism.

**Technical Approach:**
- Track modified time of directories to identify changed subtrees using `fs.Filesystem` Lstat capabilities
- Use file system events from the `watchaggregator` to determine which directories need scanning
- Implement a cache of directory states in the database to compare against
- Add configuration option `selectiveScanningEnabled` to enable/disable selective scanning
- Implement fallback to full folder scan when inconsistencies are detected or when cache is corrupted
- Provide API endpoints to trigger full scans manually through the REST interface
- Add metrics to track when selective vs full scans are performed
- Implement subtree scanning in `lib/scanner/walk.go` by adding a new `WalkSubtree` function
- Add directory change tracking in `lib/model/folder.go` to maintain a list of changed directories
- Implement cache validation to ensure consistency between file system and cached states

#### 4.1.2 Adaptive Scan Intervals
Dynamically adjust scan intervals based on folder activity patterns.

**Technical Approach:**
- Monitor file change frequency in each folder
- Increase scan frequency for active folders, decrease for inactive ones
- Implement exponential backoff for folders with no changes

#### 4.1.3 Improved Hashing Performance
Optimize parallel hashing based on available CPU cores and I/O capacity.

**Technical Approach:**
- Dynamically adjust the number of hashers based on system load
- Implement I/O-aware scheduling to prevent disk contention
- Use more efficient hashing algorithms where appropriate

### 4.2 Enhanced Monitoring

#### 4.2.1 Adaptive Event Buffering
Implement dynamic buffer sizing based on event throughput and system performance.

**Technical Approach:**
- Monitor buffer overflow events and adjust buffer size accordingly
- Implement predictive buffer sizing based on historical event patterns
- Add buffer pressure metrics for monitoring and tuning

#### 4.2.2 Smart Event Aggregation
Improve the aggregation logic to better group related events and reduce scan triggers.

**Technical Approach:**
- Implement more sophisticated event grouping algorithms
- Add configurable aggregation policies based on file types
- Optimize timeout handling for different event patterns

#### 4.2.3 Platform-Specific Optimizations
Implement specialized watchers for Windows, macOS, Linux, and Android to leverage platform-specific features for more efficient file system monitoring.

**Technical Approach:**
- **Windows**: Utilize ReadDirectoryChangesW API with overlapped I/O for efficient file system monitoring, implementing in `lib/fs/basicfs_watch.go`
- **macOS**: Leverage FSEvents API for directory-level notifications, with fallback to kqueue when FSEvents is unavailable
- **Linux**: Optimize inotify usage with adaptive watch descriptors based on directory size, implementing dynamic adjustment of `fs.inotify.max_user_watches` (referencing the existing memory about inotify limits configuration)
- **Android**: Implement specialized watcher that works within Android's background execution limitations, using Android-specific APIs and handling app lifecycle events
- Add platform-specific tuning parameters for buffer sizes and event aggregation in `lib/config/optionsconfiguration.go`
- Implement platform detection and automatic selection of appropriate watcher in `lib/fs/basicfs.go`
- Provide fallback to generic polling mechanism when platform-specific APIs fail
- Add platform-specific performance metrics and monitoring
- Implement adaptive buffer sizing based on platform capabilities and limitations

## 5. Detailed Implementation Plan

### 5.1 Phase 1: Infrastructure Improvements (Months 1-2)

#### 5.1.1 Enhanced Metrics Collection
- Add detailed performance metrics for scanning operations
- Implement event processing latency tracking
- Add buffer utilization and overflow monitoring

#### 5.1.2 Adaptive Buffer Management
- Implement the overflowTracker from basicfs_watch.go
- Add dynamic buffer resizing based on overflow frequency
- Add configuration options for buffer management

#### 5.1.3 Performance Profiling
- Add CPU and memory profiling hooks
- Implement performance baselining for comparison
- Add profiling controls to the API

### 5.2 Phase 2: Scanning Optimizations (Months 3-4)

#### 5.2.1 Selective Subtree Implementation
- Develop directory change tracking mechanism in `lib/model/folder.go`
- Implement subtree scanning in the scanner package (`lib/scanner/walk.go`)
- Add APIs for targeted scanning operations in the REST API (`lib/api`)
- Implement directory state cache in the database layer
- Add configuration options for selective scanning
- Implement fallback to full scan when needed

#### 5.2.2 Adaptive Scan Scheduling
- Implement activity-based scan interval adjustment
- Add folder activity monitoring and classification
- Develop exponential backoff algorithms for inactive folders

#### 5.2.3 Hashing Optimization
- Implement dynamic hasher count adjustment
- Add I/O-aware scheduling for hashing operations
- Optimize block queue management

### 5.3 Phase 3: Monitoring Enhancements (Months 5-6)

#### 5.3.1 Advanced Event Aggregation
- Enhance the watchaggregator with smarter grouping logic
- Implement file type-based aggregation policies
- Add configurable timeout strategies

#### 5.3.2 Platform-Specific Optimizations
- Implement specialized watchers for each platform in `lib/fs/`
- Add platform-specific tuning parameters in `lib/config/optionsconfiguration.go`
- Optimize for different file system types (NTFS, ext4, APFS, etc.)
- Implement Android-specific watcher that respects background execution limits
- Add platform detection and automatic selection logic
- Implement fallback mechanisms for each platform
- Add platform-specific performance metrics

#### 5.3.3 Large Folder Handling
- Implement folder size analysis and recommendations
- Add automatic exclusion suggestions for temporary files
- Optimize memory usage for large directory structures

## 6. Configuration Improvements

### 6.1 New Configuration Options

#### 6.1.1 Scanning Configuration
- `adaptiveScanningEnabled`: Enable/disable adaptive scan intervals
- `minScanIntervalS`: Minimum scan interval for active folders
- `maxScanIntervalS`: Maximum scan interval for inactive folders
- `scanAggressionLevel`: Control balance between performance and responsiveness (values: conservative, balanced, aggressive)
- `fallbackScanIntervalS`: Interval for fallback full scans to detect missed changes
- `selectiveScanningEnabled`: Enable/disable selective subtree scanning
- `consistencyCheckIntervalS`: Interval for consistency checks between file system and database

#### 6.1.2 Monitoring Configuration
- `adaptiveBufferEnabled`: Enable/disable adaptive buffer sizing
- `minBufferSize`: Minimum event buffer size
- `maxBufferSize`: Maximum event buffer size
- `bufferResizeFactor`: Factor for buffer size adjustments
- `eventAggregationTimeoutS`: Base timeout for event aggregation
- `platformOptimizationsEnabled`: Enable/disable platform-specific optimizations
- `fallbackPollingIntervalS`: Polling interval when file system watching fails

### 6.2 Backward Compatibility
- All new options have sensible defaults
- Existing configurations continue to work unchanged
- Deprecation warnings for inefficient settings

## 7. Performance Metrics and Monitoring

### 7.1 Key Performance Indicators

#### 7.1.1 Scanning Metrics
- **Scan Duration**: Time to complete folder scans
- **CPU Utilization**: Percentage CPU usage during scans
- **Memory Consumption**: Peak memory usage during scanning
- **Files Processed**: Number of files checked per scan
- **Hashing Efficiency**: Ratio of files hashed to files checked

#### 7.1.2 Monitoring Metrics
- **Event Processing Latency**: Time from file change to scan trigger
- **Buffer Utilization**: Percentage of buffer capacity used
- **Overflow Rate**: Number of buffer overflows per hour
- **False Trigger Rate**: Scans triggered by non-actionable events
- **Missed Events**: File changes not detected within threshold

### 7.2 Monitoring and Evaluation
- Built-in performance metrics collection
- Comparative benchmarking against current implementation
- Real-world testing with large folder structures
- Automated performance regression testing

## 8. Risk Assessment and Mitigation

### 8.1 Potential Risks

#### 8.1.1 Performance Regressions
- Risk: Optimizations may degrade performance in some scenarios
- Mitigation: Extensive testing with various folder sizes and types

#### 8.1.2 Platform Compatibility Issues
- Risk: Platform-specific optimizations may introduce bugs
- Mitigation: Comprehensive cross-platform testing

#### 8.1.3 Missed File Changes
- Risk: More aggressive aggregation may miss important events
- Mitigation: Configurable aggressiveness and fallback scanning
- Implement periodic consistency checks to detect missed changes
- Add manual scan triggers through API for user-initiated verification
- Provide logging and metrics for missed events detection

### 8.2 Rollout Strategy
- Initial release with conservative defaults
- Gradual enablement of more aggressive optimizations
- User feedback collection and iteration

## 9. Testing Strategy

### 9.1 Unit Testing
- Test individual optimization components
- Validate correctness of selective scanning
- Verify event aggregation logic
- Test adaptive buffer resizing algorithms
- Test platform-specific watcher implementations
- Validate fallback mechanisms for each platform

### 9.2 Integration Testing
- Test end-to-end scanning and monitoring workflows
- Validate performance improvements with large datasets
- Ensure compatibility across different platforms
- Test interaction with ignore patterns and versioning
- Test configurable aggressiveness settings
- Validate fallback scanning mechanisms

### 9.3 Performance Testing
- Benchmark CPU and memory usage
- Measure scan completion times
- Evaluate event processing throughput
- Test with various folder sizes (100, 1K, 10K, 100K+ files)
- Test with different file types (documents, images, binaries)
- Validate performance on low-resource devices
- Compare selective vs full scanning performance
- Test with various file system types (NTFS, ext4, APFS, etc.)
- Benchmark platform-specific implementations
- Test under different load conditions (idle, moderate, heavy)

### 9.4 Real-World Testing
- Deploy to beta users with large file collections
- Monitor performance metrics in production
- Collect user feedback on responsiveness
- Compare resource usage before and after optimizations
- Test configurable aggressiveness with different user profiles
- Validate fallback scanning in production environments
- Test cross-platform compatibility with shared folders
- Monitor for regressions in file synchronization accuracy
- Validate behavior with network file systems (NFS, SMB)

## 10. Conclusion

This optimization plan addresses key performance bottlenecks in Syncthing's folder monitoring and scanning systems. By implementing adaptive algorithms, platform-specific optimizations, and smarter event processing, we can significantly improve the user experience while reducing resource consumption. The phased approach ensures careful testing and validation at each step, minimizing the risk of regressions while maximizing the benefits of these improvements.