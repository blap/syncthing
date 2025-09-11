# Improve Local Discovery and Version Compatibility

## Overview

This document outlines improvements to enhance local discovery mechanisms and ensure better compatibility between Syncthing version 1 and version 2. The focus is on addressing potential issues in the discovery protocol that could affect interoperability between different versions, and improving the robustness and efficiency of local device discovery.

## Current State Analysis

### Local Discovery Mechanism

Syncthing's local discovery uses UDP broadcasts (IPv4) and multicasts (IPv6) on port 21027 to announce device presence. Key components include:

1. **Protocol Magic Numbers**:
   - Current version uses a specific magic number for protocol identification
   - Previous version (v0.13) used a different magic number

2. **Discovery Packet Structure**:
   - Uses Protocol Buffers for serialization
   - Contains device ID, addresses, and instance ID

3. **Implementation**:
   - Periodic announcements every 30 seconds
   - Cache lifetime of 90 seconds (3 × broadcast interval)
   - Instance ID tracking to detect device restarts

### Version 2 Changes

Version 2 introduced several significant changes that could impact discovery:
- Database backend switched from LevelDB to SQLite
- Logging format changed to structured entries
- Multiple connections used by default between v2 devices
- Some platforms no longer receive prebuilt binaries
- Ed25519 keys now used for sync connections

The last v1.x version was v1.29.6, released in June 2025, before the transition to v2.x began.

## Identified Issues

### Compatibility Concerns

1. **Magic Number Handling**:
   - Only v0.13 magic number is explicitly handled
   - No specific handling for potential v2 magic numbers
   - Warning messages only address v0.13 incompatibility

2. **Cryptographic Changes**:
   - v2.0.0 introduced Ed25519 keys for sync connections which may affect discovery handshake
   - Different certificate handling between versions
   - Potential mismatch in cryptographic negotiation between v1 and v2 devices

3. **Connection Management**:
   - v2 uses multiple connections by default (one for index metadata and two for data exchange)
   - Changes to connection establishment that may affect discovery
   - Reported connectivity issues between v2.0.0 and v1.x devices when using static addresses

4. **Protocol Evolution**:
   - Missing explicit version negotiation in discovery packets
   - No graceful degradation mechanism for newer features

### Local Discovery Limitations

1. **Network Environment Challenges**:
   - Broadcast limitations in complex network topologies
   - Multicast support varies across different network equipment
   - Docker container networking can interfere with discovery

2. **Performance Considerations**:
   - Fixed broadcast interval may not be optimal for all environments
   - Cache expiration timing could be more adaptive

## Proposed Improvements

### 1. Enhanced Version Compatibility

#### Magic Number Management

Define clear magic number constants for different protocol versions to ensure proper identification and handling of discovery packets from different Syncthing versions.

#### Backward Compatibility Layer
Implement a more comprehensive compatibility layer that can handle different versions gracefully:

1. **Version Negotiation**:
   - Add optional version field to discovery packets
   - Implement feature flags for optional capabilities
   - Graceful degradation when features are not supported

2. **Cryptographic Compatibility**:
   - Support for both legacy and Ed25519 key negotiation
   - Fallback mechanisms for certificate validation
   - Enhanced error handling for cryptographic mismatches

3. **Extended Error Handling**:
   - More specific error messages for different version mismatches
   - Automatic suggestions for resolving compatibility issues
   - Better logging for troubleshooting mixed-version environments

#### v1.29.6 to v2.0.0 Transition Support
Specific improvements to handle the transition between these versions:

1. **Connection Establishment Fixes**:
   - Address reported issues with static address configurations
   - Improve local discovery fallback when direct connections fail
   - Enhanced handling of mixed connection types (single vs multiple)

2. **Discovery Cache Improvements**:
   - Better cache invalidation for version-mixed environments
   - Enhanced cache sharing between different version devices
   - Improved cache synchronization for connection preferences

### 2. Improved Local Discovery Protocol

#### Adaptive Broadcast Intervals

Implement dynamic broadcast intervals that can adapt based on network conditions, device count, and discovery success rates to optimize network traffic and discovery latency.

#### Enhanced Discovery Packet
Extend the discovery message structure to include:
- Protocol version information
- Feature capability flags
- Network type indicators (LAN, VPN, etc.)

### 3. Network Environment Adaptation

#### Discovery Method Selection
Implement intelligent selection of discovery methods based on:
- Network interface capabilities
- Container environment detection
- User configuration preferences

#### Docker-Specific Improvements
- Automatic detection of host networking mode
- Alternative discovery mechanisms for containerized environments
- Better handling of port mappings

## Implementation Plan

### Phase 1: Protocol Enhancement
1. Extend discovery protocol with version and capability fields
2. Implement version negotiation logic
3. Add comprehensive compatibility checking

### Phase 2: Adaptive Discovery
1. Implement adaptive broadcast intervals
2. Add network environment detection
3. Optimize cache management

### Phase 3: Environment-Specific Improvements
1. Enhance Docker compatibility
2. Improve multicast handling
3. Add detailed logging for troubleshooting

## Technical Details

### Extended Discovery Protocol

Extend the discovery protocol with additional fields to support version negotiation and feature advertisement:
- Protocol version information
- Feature capability flags
- Network type indicators
- Cryptographic support indicators (Ed25519, RSA, etc.)
- Connection preference information (single vs multiple connections)

This would involve extending the existing `discoproto.Announce` Protocol Buffer message structure used in local discovery.

### Version Compatibility Matrix

| Feature | Version 1.x | Version 2.x | Backward Compatible |
|---------|-------------|-------------|---------------------|
| Basic Discovery | ✓ | ✓ | ✓ |
| Multiple Connections | ✗ | ✓ | Limited |
| Extended Protocol | ✗ | ✓ | Via feature flags |
| Ed25519 Keys | ✗ | ✓ | No (requires negotiation) |

### Implementation Considerations

1. **Backward Compatibility**:
   - Ensure v1 devices can still discover v2 devices
   - Maintain existing magic number handling
   - Provide graceful degradation for new features
   - Support mixed environments with v1.29.6 and v2.0.0 devices

2. **Performance Impact**:
   - Minimize additional overhead in discovery packets
   - Optimize cache lookup and update operations
   - Reduce unnecessary network traffic
   - Implement efficient version negotiation

3. **Security**:
   - Validate extended protocol fields
   - Prevent potential DoS through discovery packets
   - Maintain existing encryption and authentication
   - Secure cryptographic negotiation between versions

## Testing Strategy

### Unit Tests
1. Version compatibility scenarios
2. Protocol extension handling
3. Adaptive interval calculations

### Integration Tests
1. Mixed version network discovery
2. Container environment testing
3. Network topology variations

### Performance Tests
1. Discovery latency measurements
2. Network traffic analysis
3. Cache efficiency evaluation

## Monitoring and Metrics

### Key Metrics to Track
1. Discovery success rate by version combination
2. Average discovery latency
3. Cache hit/miss ratios
4. Network traffic volume

### Diagnostic Information
1. Detailed compatibility error reporting
2. Network environment classification
3. Discovery method effectiveness tracking

## Rollout Plan

### Initial Deployment
1. Implement extended protocol with backward compatibility
2. Deploy to beta testing group
3. Monitor compatibility metrics

### Gradual Rollout
1. Enable adaptive features for subset of users
2. Collect performance data
3. Optimize based on real-world usage

### Full Deployment
1. Enable all features by default
2. Maintain fallback mechanisms
3. Provide migration path for existing deployments

## Future Considerations

1. **Protocol Evolution**:
   - Plan for future version compatibility
   - Establish clear deprecation policies

2. **Advanced Discovery Features**:
   - Peer-to-peer discovery enhancements
   - Integration with external service discovery

3. **Cross-Platform Consistency**:
   - Ensure consistent behavior across all supported platforms
   - Address platform-specific networking limitations