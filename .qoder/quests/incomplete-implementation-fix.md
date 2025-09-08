# Incomplete Implementation Fix Design Document

## Overview

This document outlines the fixes needed for several incomplete implementations in the Syncthing codebase that are causing compilation errors. The issues include missing methods in the PacketScheduler, unused imports, incorrect function signatures, and type mismatches.

## Issues Analysis

### 1. Missing Methods in PacketScheduler

The `complete_enhanced_integration_test.go` file references several methods that don't exist in the `PacketScheduler` implementation:

- `SelectConnectionBasedOnTraffic`
- `GetAggregatedBandwidth`
- `GetConnectionBandwidth`
- `DistributeDataChunks`

### 2. Unused Imports

The `intelligent_reconnection_test.go` file has unused imports:
- `"testing"`
- `"github.com/syncthing/syncthing/lib/protocol"`

### 3. Incorrect Function Signatures

Several tests have incorrect function signatures:
- `discovery_cache_test.go` - `NewManager` call missing required parameters
- `deviceactivity_test.go` - Type mismatch between `[]TestAvailability` and `[]Availability`
- `requests_test.go` - `NewModel` call missing required parameter

## Design Solutions

### 1. PacketScheduler Enhancement

Add the missing methods to the `PacketScheduler` struct in `packetscheduler.go`:

#### SelectConnectionBasedOnTraffic
```go
// SelectConnectionBasedOnTraffic selects the best connection based on traffic metrics
func (ps *PacketScheduler) SelectConnectionBasedOnTraffic(deviceID protocol.DeviceID) protocol.Connection {
    ps.mut.RLock()
    defer ps.mut.RUnlock()

    conns, ok := ps.connections[deviceID]
    if !ok || len(conns) == 0 {
        return nil
    }

    // If only one connection, return it
    if len(conns) == 1 {
        return conns[0]
    }

    // Select based on traffic metrics (bandwidth, latency, packet loss)
    return ps.selectBestConnectionByTraffic(conns)
}
```

#### GetAggregatedBandwidth
```go
// GetAggregatedBandwidth returns the total bandwidth across all connections for a device
func (ps *PacketScheduler) GetAggregatedBandwidth(deviceID protocol.DeviceID) float64 {
    ps.mut.RLock()
    defer ps.mut.RUnlock()

    conns, ok := ps.connections[deviceID]
    if !ok {
        return 0
    }

    var totalBandwidth float64
    for _, conn := range conns {
        if trafficConn, ok := conn.(interface{ GetBandwidth() float64 }); ok {
            totalBandwidth += trafficConn.GetBandwidth()
        }
    }

    return totalBandwidth
}
```

#### GetConnectionBandwidth
```go
// GetConnectionBandwidth returns the bandwidth for a specific connection
func (ps *PacketScheduler) GetConnectionBandwidth(deviceID protocol.DeviceID, connID string) float64 {
    ps.mut.RLock()
    defer ps.mut.RUnlock()

    conns, ok := ps.connections[deviceID]
    if !ok {
        return 0
    }

    for _, conn := range conns {
        if conn.ConnectionID() == connID {
            if trafficConn, ok := conn.(interface{ GetBandwidth() float64 }); ok {
                return trafficConn.GetBandwidth()
            }
            break
        }
    }

    return 0
}
```

#### DistributeDataChunks
```go
// DistributeDataChunks distributes data chunks across connections based on their capabilities
func (ps *PacketScheduler) DistributeDataChunks(deviceID protocol.DeviceID, chunkSize int64) map[string]int64 {
    ps.mut.RLock()
    defer ps.mut.RUnlock()

    result := make(map[string]int64)
    
    conns, ok := ps.connections[deviceID]
    if !ok || len(conns) == 0 {
        return result
    }

    // Distribute chunks based on connection bandwidth
    totalBandwidth := ps.GetAggregatedBandwidth(deviceID)
    if totalBandwidth <= 0 {
        // Distribute evenly if no bandwidth info
        chunkPerConn := chunkSize / int64(len(conns))
        for _, conn := range conns {
            result[conn.ConnectionID()] = chunkPerConn
        }
        return result
    }

    for _, conn := range conns {
        if trafficConn, ok := conn.(interface{ GetBandwidth() float64 }); ok {
            bandwidth := trafficConn.GetBandwidth()
            allocation := int64((bandwidth / totalBandwidth) * float64(chunkSize))
            result[conn.ConnectionID()] = allocation
        } else {
            result[conn.ConnectionID()] = 0
        }
    }

    return result
}
```

#### Helper Methods
```go
// selectBestConnectionByTraffic selects the best connection based on traffic metrics
func (ps *PacketScheduler) selectBestConnectionByTraffic(connections []protocol.Connection) protocol.Connection {
    if len(connections) == 0 {
        return nil
    }

    bestConn := connections[0]
    bestScore := ps.getTrafficScore(bestConn)

    for _, conn := range connections[1:] {
        score := ps.getTrafficScore(conn)
        if score > bestScore {
            bestConn = conn
            bestScore = score
        }
    }

    return bestConn
}

// getTrafficScore calculates a score based on traffic metrics
func (ps *PacketScheduler) getTrafficScore(conn protocol.Connection) float64 {
    // Try to get traffic metrics from the connection
    if trafficConn, ok := conn.(interface {
        GetBandwidth() float64
        GetLatency() time.Duration
        GetPacketLoss() float64
    }); ok {
        bandwidth := trafficConn.GetBandwidth()
        latency := trafficConn.GetLatency()
        packetLoss := trafficConn.GetPacketLoss()

        // Calculate weighted score
        // Higher bandwidth = better, lower latency = better, lower packet loss = better
        latencyScore := 1.0 / (1.0 + latency.Seconds())
        packetLossScore := 1.0 - packetLoss
        return bandwidth * latencyScore * packetLossScore
    }

    // Fallback to health score if traffic metrics not available
    return ps.getHealthScore(conn)
}
```

### 2. Unused Import Cleanup

Remove unused imports from `intelligent_reconnection_test.go`:
- Remove `"testing"` import
- Remove `"github.com/syncthing/syncthing/lib/protocol"` import

### 3. Function Signature Fixes

#### Discovery Cache Test Fix
Update the `NewManager` call in `discovery_cache_test.go` to include the missing parameters:

```go
manager := NewManager(
    protocol.LocalDeviceID, 
    config.Wrap("", cfg, protocol.LocalDeviceID, events.NoopLogger), 
    tls.Certificate{}, 
    events.NoopLogger, 
    nil, 
    registry.New(),
    nil, // Add the missing ConnectionServiceSubsetInterface parameter
).(*manager)
```

#### Device Activity Test Fix
Fix the type mismatch in `deviceactivity_test.go` by using the proper conversion:

Change line 48 from:
```go
if lb := da.leastBusy(availability); lb != 0 {
```

To:
```go
if lb := da.leastBusy(convertTestAvailability(availability)); lb != 0 {
```

#### Requests Test Fix
Update the `NewModel` call in `requests_test.go` to include the missing discoverer parameter:

Change:
```go
model:    NewModel(m.cfg, m.id, m.sdb, m.protectedFiles, m.evLogger, protocol.NewKeyGenerator()).(*model),
```

To:
```go
model:    NewModel(m.cfg, m.id, m.sdb, m.protectedFiles, m.evLogger, protocol.NewKeyGenerator(), nil).(*model),
```

## Implementation Plan

1. **Enhance PacketScheduler**:
   - Add the missing methods to `packetscheduler.go`:
     - Add `SelectConnectionBasedOnTraffic` method that selects connections based on traffic metrics
     - Add `GetAggregatedBandwidth` method that calculates total bandwidth across all connections
     - Add `GetConnectionBandwidth` method that retrieves bandwidth for a specific connection
     - Add `DistributeDataChunks` method that distributes data based on connection capabilities
     - Implement helper methods `selectBestConnectionByTraffic` and `getTrafficScore`

2. **Clean up unused imports**:
   - Remove unused `"testing"` import from `intelligent_reconnection_test.go`
   - Remove unused `"github.com/syncthing/syncthing/lib/protocol"` import from `intelligent_reconnection_test.go`

3. **Fix function signatures**:
   - Update `NewManager` call in `discovery_cache_test.go` by adding the missing `protocol.ConnectionServiceSubsetInterface` parameter
   - Fix type conversion in `deviceactivity_test.go` by using `convertTestAvailability(availability)` instead of `availability`
   - Update `NewModel` call in `requests_test.go` by adding the missing `discover.Finder` parameter

4. **Verify compilation**:
   - Run `build-cgo.bat` to ensure all compilation errors are resolved

## Testing Strategy

1. **Unit Testing**:
   - Ensure all existing tests continue to pass
   - Add new tests for the enhanced PacketScheduler methods to verify:
     - Traffic-based connection selection works correctly
     - Bandwidth aggregation calculates correctly
     - Data chunk distribution is balanced based on connection capabilities

2. **Integration Testing**:
   - Run the complete integration test (`complete_enhanced_integration_test.go`) to verify all components work together
   - Verify that the enhanced connection management features function as expected

3. **Compilation Verification**:
   - Confirm that all compiler errors are resolved:
     - PacketScheduler method undefined errors
     - Unused import errors
     - Wrong argument count errors
     - Type mismatch errors
   - Verify successful build with `build-cgo.bat`

## Dependencies

The implementation depends on the existing connection interfaces and requires the following specific changes to be made in the codebase:

1. **lib/connections/packetscheduler.go**: Add the missing methods:
   - `SelectConnectionBasedOnTraffic`
   - `GetAggregatedBandwidth` 
   - `GetConnectionBandwidth`
   - `DistributeDataChunks`
   - Helper methods `selectBestConnectionByTraffic` and `getTrafficScore`

2. **lib/connections/intelligent_reconnection_test.go**: Remove unused imports:
   - `"testing"`
   - `"github.com/syncthing/syncthing/lib/protocol"`

3. **lib/discover/discovery_cache_test.go**: Fix NewManager call by adding the missing parameter:
   - Add `nil` as the last parameter for `protocol.ConnectionServiceSubsetInterface`

4. **lib/model/deviceactivity_test.go**: Fix type conversion:
   - Change `da.leastBusy(availability)` to `da.leastBusy(convertTestAvailability(availability))`

5. **lib/model/requests_test.go**: Fix NewModel call by adding the missing parameter:
   - Add `nil` as the last parameter for `discover.Finder`

## Summary

This design document outlines the necessary fixes for incomplete implementations in the Syncthing codebase that are causing compilation errors. The main issues include missing methods in the PacketScheduler, unused imports, and incorrect function signatures. By implementing the solutions described above, all compiler errors should be resolved and the codebase should compile successfully with `build-cgo.bat`.