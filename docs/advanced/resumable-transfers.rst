Resumable Block Transfers
=========================

Overview
--------

Resumable block transfers is an advanced feature that prevents large file block transfers from restarting from zero after brief network interruptions. This document covers the technical details of how the feature works and advanced configuration options.

Technical Implementation
------------------------

The resumable transfer feature is implemented in the puller component of Syncthing's synchronization engine:

1. **Chunk-based Downloading**: When resumable transfers are enabled, the [pullBlockResumable](file:///C:/Users/Admin/Documents/GitHub/syncthing/lib/model/folder_sendrecv.go#L2084-L2182) method divides blocks into chunks based on the ``transferChunkSizeBytes`` configuration.

2. **Checkpoint Management**: Each completed chunk is saved to a temporary file (.stpart) as it's received. The system maintains an in-memory offset of the last successfully received and saved chunk.

3. **Connection Resumption**: When a connection is reestablished after a drop, the puller populates the offset field in the request with the saved checkpoint value.

4. **Block Verification**: After all chunks are downloaded, the complete block is verified against its hash to ensure data integrity.

Configuration Details
---------------------

Transfer Chunk Size
^^^^^^^^^^^^^^^^^^^

The ``transferChunkSizeBytes`` option determines the size of chunks used for checkpointing. The default value is 1048576 bytes (1 MiB). Considerations for setting this value:

- **Smaller values** (e.g., 256 KiB) provide more frequent checkpoints but increase overhead
- **Larger values** (e.g., 4 MiB) reduce overhead but provide fewer resumption points
- **Very small values** (< 64 KiB) may impact performance significantly
- **Very large values** (> 16 MiB) may reduce the effectiveness of resumability

Optimal Performance Tuning
--------------------------

For best performance with resumable transfers:

1. **Match chunk size to network conditions**: Use smaller chunks for unstable connections
2. **Consider file size distribution**: If most files are small, the benefit may be limited
3. **Monitor temporary storage**: Ensure adequate disk space for temporary files
4. **Test with real workloads**: Performance can vary significantly based on specific usage patterns

Integration with Other Features
-------------------------------

Resumable transfers work seamlessly with other Syncthing features:

- **Versioning**: Resumed transfers work correctly with versioned files
- **Encryption**: Encrypted folders support resumable transfers (note that verification works differently)
- **Compression**: Compressed connections work with resumable transfers
- **Selective Sync**: Partial folder synchronization benefits from resumable transfers

Limitations and Edge Cases
--------------------------

Known limitations of the current implementation:

1. **Memory Usage**: Checkpoint information is stored in memory, which could impact systems with many large transfers
2. **Temporary Storage**: Requires adequate temporary storage space for checkpoint files
3. **Network Protocol**: Only works with the current BEP protocol version
4. **Block Size Dependencies**: Effectiveness depends on Syncthing's block size calculation algorithm

Troubleshooting
---------------

Common issues and solutions:

1. **No Resumption Occurring**: Verify both global and folder-level options are enabled
2. **Temporary File Issues**: Check permissions and available space in staging directories
3. **Performance Degradation**: Try adjusting the chunk size for your specific use case
4. **Memory Pressure**: Monitor memory usage during large transfer operations

The logs will contain detailed information about resumable transfer operations, including checkpoint saves and resumptions.