# Multipath Connections

Syncthing supports multipath connections, allowing devices to maintain multiple simultaneous connections for improved reliability and performance.

## Overview

Multipath connections enable Syncthing to establish and maintain multiple network paths to the same device. This provides several benefits:

- **Failover**: Automatic switching to alternative connections when primary connections fail
- **Load Balancing**: Distribution of traffic across multiple connections for better performance
- **Redundancy**: Multiple connection paths reduce the risk of complete disconnection

## Configuration

Multipath connections are controlled through the configuration file or GUI:

### Configuration File

```xml
<options>
  <multipathEnabled>true</multipathEnabled>
</options>
```

### GUI Settings

The multipath setting can be found in:
- Settings → Advanced → Enable Multipath Connections

## How It Works

When multipath is enabled, Syncthing will:

1. **Establish Multiple Connections**: Attempt to create multiple connections to each device when possible
2. **Monitor Connection Health**: Continuously evaluate the quality of each connection path
3. **Select Optimal Paths**: Use the best connection for critical operations while distributing load across available paths
4. **Failover Automatically**: Switch to alternative paths when primary connections degrade or fail

## Connection Selection

Syncthing uses a sophisticated algorithm to select the best connection based on:

- **Health Score**: Connections with better latency, jitter, and packet loss characteristics are preferred
- **Network Type**: LAN connections are generally preferred over WAN connections
- **Historical Performance**: Connections with better success rates are given priority

## Benefits

### Improved Reliability
With multiple connection paths, temporary network issues on one path won't interrupt the synchronization process.

### Better Performance
Load balancing across multiple connections can increase throughput, especially when connections use different network interfaces.

### Automatic Failover
When a connection degrades or fails, Syncthing automatically switches to alternative paths with minimal disruption.

## Limitations

- **Resource Usage**: Maintaining multiple connections uses more system resources
- **Network Requirements**: Both devices must have multiple available network paths
- **Compatibility**: Both devices must support and have multipath enabled

## Troubleshooting

If you're experiencing issues with multipath connections:

1. **Verify Configuration**: Ensure both devices have multipath enabled
2. **Check Network Connectivity**: Confirm multiple network paths exist between devices
3. **Monitor Logs**: Look for connection-related messages in the logs
4. **Test Performance**: Monitor if multipath is providing the expected benefits

## Metrics

When multipath is enabled, additional metrics are available:

- **Multipath Connections**: Number of active connections per device
- **Failover Events**: Count of automatic connection switches
- **Connection Health**: Individual health scores for each connection path

These metrics can be accessed through the REST API or monitoring systems that support Prometheus metrics.