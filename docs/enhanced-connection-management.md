# Enhanced Connection Management

This document describes the enhanced connection management features that have been added to Syncthing to improve connection reliability, performance, and resource utilization.

## Table of Contents
1. [Adaptive Health Monitoring](#adaptive-health-monitoring)
2. [Multipath Connection Support](#multipath-connection-support)
3. [Dynamic Path Selection](#dynamic-path-selection)
4. [Bandwidth Aggregation](#bandwidth-aggregation)
5. [Connection Migration](#connection-migration)
6. [Connection Pooling](#connection-pooling)
7. [Lazy Health Monitoring](#lazy-health-monitoring)
8. [Random Port Configuration](#random-port-configuration)
9. [Configuration Options](#configuration-options)

## Adaptive Health Monitoring

The enhanced health monitoring system continuously evaluates connection quality based on three key metrics:

- **Latency**: Round-trip time for packets
- **Jitter**: Variation in latency over time
- **Packet Loss**: Percentage of packets that fail to arrive

These metrics are used to calculate a health score (0-100) for each connection, where higher scores indicate better connection quality. The health score is used to make intelligent decisions about connection selection and maintenance.

### Adaptive Keep-Alive Intervals

Based on the health score, the system dynamically adjusts keep-alive intervals:
- **High Health (80-100)**: Longer intervals to reduce network overhead
- **Medium Health (50-79)**: Moderate intervals for balanced monitoring
- **Low Health (0-49)**: Shorter intervals for more aggressive monitoring

## Multipath Connection Support

Syncthing now supports maintaining multiple simultaneous connections to the same device, providing:
- **Redundancy**: Automatic failover when primary connections degrade
- **Load Balancing**: Distribution of traffic across multiple paths
- **Increased Throughput**: Utilization of combined bandwidth from all connections

When multipath is enabled, Syncthing will attempt to establish and maintain multiple connections to each peer device.

## Dynamic Path Selection

The system dynamically selects the best path for data transfer based on real-time traffic analysis:
- **Health-Based Selection**: Prefer connections with higher health scores
- **Traffic-Based Selection**: Consider current bandwidth utilization
- **Load Balancing**: Distribute traffic to prevent overloading any single connection

## Bandwidth Aggregation

Multiple connections can be used together to increase total throughput:
- **Real-time Measurement**: Continuous monitoring of available bandwidth per connection
- **Chunked Distribution**: Data is split across connections based on their capabilities
- **Adaptive Optimization**: Throughput is optimized based on changing network conditions

## Connection Migration

Active transfers can be seamlessly migrated between connections when better paths become available:
- **State Preservation**: Transfer state is maintained during migration
- **Quality-Based Triggers**: Migration occurs when significantly better connections are detected
- **Seamless Operation**: Users experience no interruption during migration

## Connection Pooling

For devices that frequently disconnect and reconnect, connection pooling provides:
- **Reduced Connection Overhead**: Reuse of existing connection resources
- **Faster Reconnection**: Immediate availability of pooled connections
- **Adaptive Pool Sizing**: Pool sizes adjust based on device behavior patterns

### Pooling Strategies

Different allocation strategies are available:
- **Round-Robin**: Sequential selection for balanced usage
- **Health-Based**: Selection of the healthiest available connection
- **Random**: Random selection for varied distribution
- **Least-Used**: Selection of the least recently used connection

## Lazy Health Monitoring

To optimize resource usage, health monitoring adapts based on connection activity:
- **Active Monitoring**: Frequent checks for active connections
- **Inactive Monitoring**: Reduced frequency for less active connections
- **Dormant Monitoring**: Minimal monitoring for dormant connections

## Random Port Configuration

Devices can be configured to use random ports within a specified range:
- **Port Range Configuration**: Define minimum and maximum port values
- **Conflict Resolution**: Automatic handling of port conflicts
- **Security Benefits**: Reduced predictability of connection endpoints

## Configuration Options

The following configuration options control the enhanced connection management features:

### Adaptive Keep-Alive Settings
```xml
<options>
  <adaptiveKeepAliveEnabled>true</adaptiveKeepAliveEnabled>
  <adaptiveKeepAliveMinS>10</adaptiveKeepAliveMinS>
  <adaptiveKeepAliveMaxS>60</adaptiveKeepAliveMaxS>
</options>
```

### Multipath Settings
```xml
<options>
  <multipathEnabled>true</multipathEnabled>
</options>
```

### Random Port Settings
```xml
<options>
  <randomPortsEnabled>true</randomPortsEnabled>
  <randomPortsMin>1024</randomPortsMin>
  <randomPortsMax>65535</randomPortsMax>
</options>
```

### Transfer Settings
```xml
<options>
  <transferChunkSizeBytes>1048576</transferChunkSizeBytes>
</options>
```

## Performance Benefits

These enhancements provide several performance benefits:
- **Improved Reliability**: Reduced connection loss and faster recovery
- **Better Resource Utilization**: More efficient use of available bandwidth
- **Reduced Latency**: Smarter path selection and load balancing
- **Lower CPU Usage**: Lazy monitoring for inactive connections
- **Enhanced Security**: Random port selection reduces predictability

## Monitoring and Metrics

The enhanced connection management system provides detailed metrics for monitoring:
- **Connection Health Scores**: Real-time quality assessment
- **Bandwidth Utilization**: Per-connection and aggregated metrics
- **Migration Events**: Tracking of connection migration occurrences
- **Pool Statistics**: Connection pool usage and efficiency metrics

These metrics are available through the standard Syncthing metrics endpoint for integration with monitoring systems.