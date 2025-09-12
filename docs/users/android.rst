Android Mobile Interface
========================

Overview
--------

The Syncthing Android app provides a mobile interface for managing your Syncthing instance on the go. It allows you to monitor your devices, manage folders, and configure settings through a user-friendly interface that communicates with your Syncthing server via the REST API.

Installation and Setup
----------------------

The Android app can be installed from the Google Play Store or F-Droid. After installation:

1. Launch the Syncthing app
2. Enter the IP address and port of your Syncthing instance (e.g., http://192.168.1.100:8384)
3. Enter your API key (found in the Syncthing web GUI under Settings > General > API Key)
4. Tap "Connect" to establish the connection

API Key Configuration
---------------------

To configure the API key:

1. Open the Syncthing web interface in your browser
2. Navigate to Settings > General
3. Find the "API Key" field
4. Copy the API key
5. Paste it into the API Key field in the Android app

Main Interface Navigation
-------------------------

The Android app uses a bottom navigation bar with the following sections:

Dashboard
~~~~~~~~~

The Dashboard provides an overview of your Syncthing instance:

- System status (running/stopped)
- Connection information
- Recent events
- Device connectivity status
- Folder synchronization status

Folders
~~~~~~~

The Folders section allows you to:

- View all configured folders
- See synchronization status for each folder
- Add new folders
- Edit folder settings
- Pause/resume folder synchronization
- View folder details and statistics

Devices
~~~~~~~

The Devices section shows:

- All configured devices
- Connection status for each device
- Last seen information
- Device identification
- Add new devices
- Edit device settings

Settings
~~~~~~~~

The Settings section provides access to:

- General app settings
- Connection configuration
- Notification preferences
- About information

Folder and Device Management
----------------------------

Managing Folders
~~~~~~~~~~~~~~~~

To add a new folder:

1. Navigate to the Folders section
2. Tap the "+" button
3. Enter the folder ID
4. Select the folder path on your device
5. Configure folder options (type, rescan interval, etc.)
6. Save the configuration

To edit an existing folder:

1. Navigate to the Folders section
2. Tap on the folder you want to edit
3. Make the necessary changes
4. Save the configuration

Managing Devices
~~~~~~~~~~~~~~~~

To add a new device:

1. Navigate to the Devices section
2. Tap the "+" button
3. Enter the device ID
4. Configure device options (name, addresses, etc.)
5. Save the configuration

To edit an existing device:

1. Navigate to the Devices section
2. Tap on the device you want to edit
3. Make the necessary changes
4. Save the configuration

Settings and Configuration
--------------------------

The Settings section allows you to configure various aspects of the Android app:

General Settings
~~~~~~~~~~~~~~~~

- Theme selection (light/dark)
- Language preferences
- Auto-start options
- Background operation settings

Connection Settings
~~~~~~~~~~~~~~~~~~~

- Syncthing server address
- API key configuration
- Connection timeout settings
- SSL/TLS configuration

Notification Settings
~~~~~~~~~~~~~~~~~~~~~

- Enable/disable notifications
- Notification priority
- Vibration settings
- LED indicator settings

Troubleshooting Common Issues
-----------------------------

Connection Issues
~~~~~~~~~~~~~~~~~

If you're having trouble connecting to your Syncthing instance:

1. Verify the server address and port are correct
2. Ensure your API key is properly configured
3. Check that your Syncthing instance is running
4. Verify network connectivity between your Android device and Syncthing server
5. Check firewall settings if applicable

Synchronization Problems
~~~~~~~~~~~~~~~~~~~~~~~~

If folders aren't synchronizing properly:

1. Check folder status in the Folders section
2. Verify device connectivity in the Devices section
3. Check the Syncthing logs for error messages
4. Ensure folder paths are accessible
5. Verify folder permissions

Performance Issues
~~~~~~~~~~~~~~~~~~

If the app is running slowly:

1. Check available storage space
2. Restart the Syncthing service
3. Reduce the number of monitored folders
4. Adjust rescan intervals for folders