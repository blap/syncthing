.. _folder-path-conflicts:

Folder Path Conflicts
=====================

Syncthing prevents folder paths from overlapping to avoid data synchronization issues. This document explains the restrictions and how to resolve conflicts.

Why Path Restrictions Exist
---------------------------

Syncthing enforces path restrictions to prevent:

1. **Data conflicts during synchronization** - When folders overlap, it can lead to unexpected file overwrites or conflicts
2. **Recursive scanning issues** - Overlapping paths can cause infinite loops during file scanning
3. **Unexpected behavior** - File operations may not work as expected with overlapping folders

Path Overlap Detection
----------------------

Syncthing detects three types of path overlaps:

1. **Identical paths** - Two folders using exactly the same path
2. **Subdirectory relationships** - One folder's path is a subdirectory of another folder's path
3. **Parent-child relationships** - One folder's path contains another folder's path as a subdirectory

Example Error Messages
----------------------

When path conflicts are detected, you'll see error messages like:

::

    Folder path conflict detected: Folder "ahqrm-5jgc7" (D:\Syncthing\Syncthing) contains folder "p6qu4-ks3vs" (D:\Syncthing\Syncthing\tea) as a subdirectory. This configuration is not allowed to prevent data synchronization issues. Please restructure your folders so they don't overlap.

Resolving Path Conflicts
------------------------

To resolve path conflicts, you need to restructure your folders so they don't overlap. Here are some recommended approaches:

1. **Move folders to separate locations**:
   - Move one folder outside the path of the other
   - Example: Move "tea" folder from "D:\Syncthing\Syncthing\tea" to "D:\Syncthing\tea"

2. **Create a common parent directory**:
   - Create a new parent folder to contain both folders
   - Example: Create "D:\Syncthing\Projects" and move both folders under it

3. **Use more specific folder paths**:
   - Ensure each folder has a distinct, non-overlapping path

Best Practices
--------------

To avoid path conflicts:

1. Plan your folder structure carefully before creating folders
2. Avoid creating folders that are subdirectories of other folders
3. Use descriptive, specific paths for each folder
4. Regularly review your folder configuration to ensure no overlaps exist

Example Folder Restructuring
----------------------------

For the error scenario:
- Folder "ahqrm-5jgc7": "D:\Syncthing\Syncthing"
- Folder "p6qu4-ks3vs": "D:\Syncthing\Syncthing\tea"

Recommended solutions:

1. **Move the "tea" folder outside the "Syncthing" folder**:
   - Folder "ahqrm-5jgc7": "D:\Syncthing\Syncthing"
   - Folder "p6qu4-ks3vs": "D:\Syncthing\tea"

2. **Create a common parent folder**:
   - Create a new folder: "D:\Syncthing\Projects"
   - Folder "ahqrm-5jgc7": "D:\Syncthing\Projects\Syncthing"
   - Folder "p6qu4-ks3vs": "D:\Syncthing\Projects\tea"

GUI Validation
--------------

The Syncthing web GUI provides real-time validation to help prevent path conflicts:

1. When editing a folder path, the GUI will immediately warn you if the path conflicts with existing folders
2. Clear error messages explain the nature of the conflict
3. Links to this documentation are provided for more information

Additional Resources
--------------------

For more information about folder configuration, see:
- :doc:`Configuration <config>`
- :doc:`Folder Types <foldertypes>`
- :doc:`Syncing <syncing>`