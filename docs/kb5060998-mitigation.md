# Windows 10 Update KB5060998 Impact Mitigation

## Overview

This document describes the mitigation strategies implemented in Syncthing to address potential connectivity issues caused by Windows 10 update KB5060998. The update, released on June 10, 2025, may affect network connectivity components that Syncthing relies on for peer-to-peer communication.

## Key Changes in KB5060998

While Microsoft's official release notes do not document specific networking issues, the update includes modifications to several network-related components:

- Network driver updates (netvsc.sys, storvsc.sys, vmbus.sys)
- Windows Defender components
- Network profile and policy handling components
- TCP/IP stack components
- Bluetooth networking components
- Remote Access Service components

## Enhanced Windows Network Monitoring

Syncthing now includes enhanced network monitoring specifically designed to detect and mitigate issues related to KB5060998:

### 1. Real-time Network Change Detection

- **Immediate Notifications**: Registered for real-time IP address and route change notifications using Windows IP Helper API
- **Debounced Processing**: Added small delays to prevent excessive reconnection attempts during rapid network changes
- **Comprehensive Event Logging**: Detailed logging of all network change events for diagnostics

### 2. Adaptive Timeout Mechanisms

- **Dynamic Scan Intervals**: Adjusts network scanning frequency based on network stability (2-10 seconds)
- **Adaptive Connection Timeouts**: Modifies connection timeouts based on recent network behavior
- **Stability Scoring**: Maintains a stability score to determine optimal network behavior

### 3. Enhanced Network Profile Detection

- **Full COM Integration**: Complete integration with Windows Network List Manager COM interface for precise network category detection
- **Fallback Heuristics**: Improved network profile detection using interface-based heuristics when COM APIs are unavailable
- **Domain Network Detection**: Better identification of domain networks based on adapter characteristics
- **Profile Change Monitoring**: Tracks network profile changes that might affect connectivity

### 4. KB5060998-Specific Detection

- **Frequent Change Detection**: Monitors for frequent network adapter state changes that may indicate KB5060998 impact
- **Warning Logs**: Generates specific warning messages when KB5060998-related issues are suspected
- **Aggressive Reconnection**: Triggers immediate reconnection attempts when network instability is detected

## Implementation Details

### Network Stability Metrics

The enhanced monitor tracks several metrics to determine network stability:

- Total network changes
- Recent changes (last 30 seconds)
- Stability score (0.0 to 1.0)
- Last error time
- Adaptive timeout values

### Event Logging

Comprehensive event logging captures:

- Adapter additions/removals
- State changes (up/down)
- Profile changes
- Type/media/speed changes
- KB5060998 suspected events
- Reconnection triggers

### Diagnostic Reporting

Periodic diagnostic reports include:

- Adapter details and status
- Stability metrics
- Recent network events
- Profile information

## Configuration

The enhanced network monitoring is automatically enabled on Windows platforms and requires no additional configuration.

## Troubleshooting

If you experience connectivity issues after installing KB5060998:

1. **Check Logs**: Look for "KB5060998 impact suspected" warnings in the Syncthing logs
2. **Verify Ports**: Ensure port 22000 is open in Windows Firewall
3. **Restart Syncthing**: A service restart may resolve temporary issues
4. **Network Diagnostics**: Run the built-in network diagnostics using `syncthing -verbose`

## Technical Implementation

### Core Components

- `WindowsNetworkMonitor`: Main monitoring class with full COM integration
- `NetworkAdapterInfo`: Detailed adapter information structure
- `NetworkStabilityMetrics`: Stability tracking and adaptive behavior
- `NetworkChangeEvent`: Event logging structure

### Key Methods

- `checkForNetworkChanges()`: Primary change detection logic
- `updateAdaptiveTimeouts()`: Adjusts timeouts based on stability
- `logNetworkEvent()`: Comprehensive event logging
- `triggerReconnection()`: Forces immediate reconnection to all devices
- `getNetworkProfileWindows()`: Uses COM interface for precise network category detection

## Testing

The implementation includes comprehensive unit and integration tests:

- Basic functionality tests
- Adapter state change detection
- Network profile handling with COM integration
- Stability metric calculations
- Event logging verification
- KB5060998 detection scenarios

## Future Enhancements

Planned improvements include:

- Enhanced WiFi roaming support
- Better virtual network adapter handling
- Integration with Windows Event Log for system-level diagnostics