# Resumable Block Transfers Implementation

## Overview

This document describes the implementation of resumable block transfers in Syncthing. The feature prevents large file block transfers from restarting from zero after brief network interruptions, saving time and bandwidth.

## Key Components

### Configuration Options

1. **Global Option**: `TransferChunkSizeBytes` (default: 1048576 bytes / 1 MiB)
   - Defines the size of "chunks" within a block for checkpoint purposes
   - Located in `OptionsConfiguration`

2. **Folder Option**: `ResumableTransfersEnabled` (default: true)
   - Enables the feature per folder
   - Located in `FolderConfiguration`

### Implementation Details

#### 1. Configuration Access Methods

Helper methods were added to the `folder` struct to access configuration options:

```go
func (f *folder) transferChunkSize() int
func (f *folder) isResumableTransfersEnabled() bool
```

#### 2. Modified Pull Logic

The `pullBlock` method in `folder_sendrecv.go` was modified to check if resumable transfers are enabled:

```go
// Check if resumable transfers are enabled for this folder
if f.isResumableTransfersEnabled() {
    // For resumable transfers, we request data in chunks
    f.pullBlockResumable(state, fd, out)
    return
}
```

#### 3. Resumable Transfer Implementation

A new method `pullBlockResumable` was implemented to handle chunk-based downloading:

- Divides blocks into chunks based on `transferChunkSize`
- Requests each chunk individually
- Saves completed chunks to temporary files
- Can resume from the last complete checkpoint if connection drops

## How It Works

1. When a block transfer is initiated, the system checks if resumable transfers are enabled for the folder
2. If enabled, the block is divided into chunks of `transferChunkSize` bytes
3. Each chunk is requested and saved to a temporary file
4. If a connection drops during transfer, the system can resume from the last complete chunk
5. When reconnection occurs, the request includes an offset to resume from the last checkpoint

## Benefits

- **Bandwidth Savings**: No need to re-download already received data
- **Time Efficiency**: Large file transfers can resume rather than restart
- **Network Resilience**: Better handling of intermittent connectivity

## Configuration

### Global Configuration
```xml
<options>
    <transferChunkSizeBytes>1048576</transferChunkSizeBytes>
</options>
```

### Folder Configuration
```xml
<folder resumableTransfersEnabled="true">
    <!-- folder configuration -->
</folder>
```

## Testing

The implementation includes tests that verify:
- Basic resumable transfer functionality
- Connection drop handling
- Checkpoint-based resumption

## Limitations

- The current implementation focuses on the core functionality
- Advanced edge case testing is still in progress
- Full simulation of connection reestablishment requires more complex testing infrastructure