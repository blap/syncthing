# Syncthing Folder Path Conflict Resolution Design

## Document Status: Draft

## Overview

This document describes the design for resolving folder path conflicts in Syncthing, specifically addressing the issue where one folder's path contains another folder's path as a subdirectory. The error occurs during configuration loading when Syncthing detects overlapping folder paths that could lead to data corruption or unexpected behavior.

This design follows the Backend Service Documentation Specialization pattern as Syncthing is primarily a Go-based backend service with a web GUI.

## Problem Statement

The error message from the logs indicates:
```
Failed to initialize config (error="failed to load config: folder \"ahqrm-5jgc7\": folder \"ahqrm-5jgc7\": path \"D:\\Syncthing\\Syncthing\" contains folder \"p6qu4-ks3vs\" (\"D:\\Syncthing\\Syncthing\\tea\") as a subdirectory")
```

This happens because Syncthing enforces a policy that prevents folder paths from being subdirectories of each other to avoid:
1. Potential data conflicts during synchronization
2. Recursive scanning issues
3. Unexpected behavior in file operations

## Repository Type

This is a Backend Service project using Go, with a web-based GUI for configuration management.

## Current Implementation Analysis

### Path Validation Logic

The current validation logic in `folderconfiguration.go` prevents folder path overlaps:

```go
// checkPathOverlaps verifies that this folder's path doesn't overlap with any other folder's path
func (f *FolderConfiguration) checkPathOverlaps(allFolders []FolderConfiguration) error {
    if f.Path == "" {
        return nil // Empty path will be caught by other validation
    }
    
    // Normalize paths for comparison
    currentPath := filepath.Clean(f.Path)
    
    for _, otherFolder := range allFolders {
        // Skip self
        if otherFolder.ID == f.ID {
            continue
        }
        
        if otherFolder.Path == "" {
            continue // Skip folders with empty paths
        }
        
        // Normalize the other folder's path for comparison
        otherPath := filepath.Clean(otherFolder.Path)
        
        // Check if paths are the same
        if currentPath == otherPath {
            return fmt.Errorf("folder %q: path %q is the same as folder %q", f.ID, f.Path, otherFolder.ID)
        }
        
        // Check if current path is a subdirectory of other path
        if strings.HasPrefix(currentPath, otherPath+string(filepath.Separator)) {
            return fmt.Errorf("folder %q: path %q is a subdirectory of folder %q (%q)", f.ID, f.Path, otherFolder.ID, otherFolder.Path)
        }
        
        // Check if other path is a subdirectory of current path
        if strings.HasPrefix(otherPath, currentPath+string(filepath.Separator)) {
            return fmt.Errorf("folder %q: path %q contains folder %q (%q) as a subdirectory", f.ID, f.Path, otherFolder.ID, otherFolder.Path)
        }
    }
    
    return nil
}
```

This validation is called during configuration loading in `config.go`:

```go
// Perform additional validation after all folders are prepared
for i := range cfg.Folders {
    folder := &cfg.Folders[i]
    
    // Validate marker name
    if err := folder.validateMarkerName(); err != nil {
        return nil, fmt.Errorf("folder %q: %w", folder.ID, err)
    }
    
    // Check for path overlaps
    if err := folder.checkPathOverlaps(cfg.Folders); err != nil {
        return nil, fmt.Errorf("folder %q: %w", folder.ID, err)
    }
}
```

## Proposed Solutions

### Solution 1: Enhanced Error Handling and User Guidance

**Description**: Improve the error message and provide clearer guidance to users on how to resolve the conflict.

**Implementation**:
1. Enhance error messages to be more descriptive
2. Add links to documentation explaining the restriction
3. Suggest alternative folder structures

### Solution 2: Configuration Validation in GUI

**Description**: Add real-time validation in the web GUI to prevent users from creating conflicting folder configurations.

**Implementation**:
1. Add client-side validation in the folder editing interface
2. Show warnings before saving configurations with overlapping paths
3. Provide visual indicators of path conflicts

### Solution 3: Documentation and User Education

**Description**: Create comprehensive documentation and user guides to help users understand and avoid path conflicts.

**Implementation**:
1. Create detailed documentation about folder path restrictions
2. Add troubleshooting guides for common path conflict scenarios
3. Include examples of proper folder organization

## Design Details

### Enhanced Error Messages

Current error:
```
folder "ahqrm-5jgc7": path "D:\Syncthing\Syncthing" contains folder "p6qu4-ks3vs" ("D:\Syncthing\Syncthing\tea") as a subdirectory
```

Improved error:
```
Folder path conflict detected: Folder "ahqrm-5jgc7" (D:\Syncthing\Syncthing) contains folder "p6qu4-ks3vs" (D:\Syncthing\Syncthing\tea) as a subdirectory. 
This configuration is not allowed to prevent data synchronization issues. 
Please restructure your folders so they don't overlap. 
See documentation at https://docs.syncthing.net/users/folderconfiguration.html#path-conflicts for more information.
```

### GUI Validation Implementation

Add validation to the folder editing interface:

1. **Path Validation Directive**:
   - Extend the existing `pathIsSubDir` directive in `pathIsSubDirDirective.js`
   - Add real-time checking against all existing folders
   - Display warnings immediately when a conflicting path is entered

2. **Warning Messages**:
   - Clear indication when a path is a subdirectory of an existing folder
   - Clear indication when a path contains an existing folder as subdirectory
   - Links to documentation for resolving conflicts

### Example Folder Restructuring

For the given error scenario:
- Folder "ahqrm-5jgc7": "D:\Syncthing\Syncthing"
- Folder "p6qu4-ks3vs": "D:\Syncthing\Syncthing\tea"

Recommended solutions:
1. Move the "tea" folder outside the "Syncthing" folder:
   - Folder "ahqrm-5jgc7": "D:\Syncthing\Syncthing"
   - Folder "p6qu4-ks3vs": "D:\Syncthing\tea"

2. Create a common parent folder:
   - Create a new folder: "D:\Syncthing\Projects"
   - Folder "ahqrm-5jgc7": "D:\Syncthing\Projects\Syncthing"
   - Folder "p6qu4-ks3vs": "D:\Syncthing\Projects\tea"

## Implementation Plan

### Phase 1: Immediate Improvements (v1.0)
1. Enhance error messages with more descriptive text
2. Add documentation links to error messages
3. Improve GUI validation warnings

### Phase 2: Documentation and User Education (v1.1)
1. Create comprehensive documentation about folder path restrictions
2. Add troubleshooting guides for common path conflict scenarios
3. Include examples of proper folder organization

## Security Considerations

1. **Data Integrity**: Preventing folder path overlaps maintains data integrity by avoiding recursive synchronization scenarios
2. **User Awareness**: Enhanced error messages ensure users understand why the restriction exists
3. **Override Mechanism**: Any override mechanism must include clear warnings about potential risks

## Testing Strategy

### Unit Tests
1. Test path overlap detection with various path combinations
2. Test error message formatting
3. Test GUI validation logic

### Integration Tests
1. Test configuration loading with overlapping paths
2. Test GUI folder creation with conflicting paths
3. Test error message display in different scenarios

## Backward Compatibility

The proposed changes maintain backward compatibility by:
1. Not changing the core validation logic
2. Only enhancing error messages and user interface
3. Keeping existing configuration files valid

## Performance Impact

The proposed changes have minimal performance impact:
1. Error message enhancements: No performance impact
2. GUI validation: Minimal client-side processing
3. Path validation logic remains unchanged

## Documentation Updates

1. Update folder configuration documentation to explain path overlap restrictions
2. Add troubleshooting guide for resolving path conflicts
3. Update GUI documentation to reflect new warning messages
4. Create examples of proper folder organization patterns