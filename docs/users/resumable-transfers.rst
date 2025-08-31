.. default-domain:: stconf

Resumable Block Transfers
=========================

Overview
--------

Resumable block transfers is a feature that prevents large file block transfers from restarting from zero after brief network interruptions, saving time and bandwidth. When enabled, large blocks are downloaded in chunks, and transfers can resume from the last complete checkpoint if a connection drops.

How It Works
------------

When resumable transfers are enabled:

1. Large blocks are divided into chunks based on the ``transferChunkSizeBytes`` configuration
2. Each chunk is requested and saved to a temporary file as it's received
3. If a connection drops during transfer, the system remembers the last complete checkpoint
4. When reconnection occurs, the request includes an offset to resume from the last checkpoint
5. Only the remaining data is transferred, not the entire block

Configuration
-------------

Global Options
^^^^^^^^^^^^^^

.. option:: options.transferChunkSizeBytes

    Defines the size of "chunks" within a block for resumable transfer checkpoint purposes.
    When resumable transfers are enabled, large blocks are downloaded in chunks of this size,
    allowing transfers to resume from the last complete checkpoint if a connection drops.
    The default value is 1048576 bytes (1 MiB).

.. option:: options.resumableTransfersEnabled

    Enables or disables resumable block transfers globally. When enabled (default: true),
    folders can use resumable transfers if they have the folder-level option enabled.
    When disabled, no resumable transfers will occur regardless of folder settings.

Folder Options
^^^^^^^^^^^^^^

.. option:: folder.resumableTransfersEnabled

    Enables or disables resumable block transfers for this folder. When enabled (default: true),
    large blocks will be downloaded in chunks, allowing transfers to resume from the last
    complete checkpoint if a connection drops. Requires the global resumable transfers option
    to also be enabled.

Benefits
--------

- **Bandwidth Savings**: No need to re-download already received data
- **Time Efficiency**: Large file transfers can resume rather than restart
- **Network Resilience**: Better handling of intermittent connectivity
- **Resource Optimization**: Reduced CPU and disk I/O for resumed transfers

Example Configuration
---------------------

To enable resumable transfers globally with a 2 MiB chunk size:

.. code-block:: xml

    <options>
        <resumableTransfersEnabled>true</resumableTransfersEnabled>
        <transferChunkSizeBytes>2097152</transferChunkSizeBytes>
        <!-- other options -->
    </options>

To enable resumable transfers for a specific folder:

.. code-block:: xml

    <folder id="xyz" label="My Folder" path="/path/to/folder" resumableTransfersEnabled="true">
        <!-- folder configuration -->
    </folder>

Performance Considerations
--------------------------

Resumable transfers introduce a small overhead for very small files due to the chunking mechanism, but provide significant benefits for large files:

- For files smaller than the chunk size, there's no benefit to resumable transfers
- For files larger than the chunk size, benefits increase with file size
- Network interruptions become less costly as only the incomplete chunk needs to be retransmitted

The default chunk size of 1 MiB provides a good balance between overhead and resumability for most use cases.

Troubleshooting
---------------

If resumable transfers don't seem to be working:

1. Verify that both global and folder-level resumable transfer options are enabled
2. Check that the ``transferChunkSizeBytes`` is set to an appropriate value
3. Ensure that temporary files can be created in the folder's staging directory
4. Monitor the logs for any errors related to temporary file operations

The feature works best with stable temporary storage, as the checkpoint data needs to be reliably saved to disk.