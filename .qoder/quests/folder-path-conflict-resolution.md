# Folder Path Conflict Resolution Design

## Overview

This design document proposes a solution to automatically allow folders within other folders in Syncthing while maintaining data integrity. Currently, Syncthing strictly prohibits folder path overlaps, but this design modifies the approach to automatically allow nesting with appropriate safeguards to prevent synchronization issues.

## Architecture

### Current Implementation

The current implementation in `lib/config/folderconfiguration.go` enforces strict path validation that prevents any folder path overlaps:

1. Identical paths are not allowed (two folders cannot have the exact same path)
2. Subdirectory relationships are not allowed (one folder cannot be a subdirectory of another)
3. Parent-child relationships are not allowed (one folder cannot contain another folder as a subdirectory)

This is implemented in the `checkPathOverlaps` function which validates all folder configurations during startup and configuration changes.

### Proposed Changes

The proposed solution automatically allows controlled folder nesting with appropriate safeguards:

1. Modify the path validation logic to allow nesting while preventing synchronization issues
2. Update the business logic to handle nested folders appropriately
3. Ensure proper scanning, indexing, and synchronization of nested folders

## Configuration Changes

### Updated Path Validation Logic

Modify the `checkPathOverlaps` function to allow nesting while preventing problematic scenarios:

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
            return fmt.Errorf("folder %q: path %q is the same as folder %q. Folder path conflict detected: Two folders cannot use the same path.", f.ID, f.Path, otherFolder.ID)
        }
        
        // Allow nesting but prevent problematic scenarios
        // Check if current path is a subdirectory of other path
        if strings.HasPrefix(currentPath, otherPath+string(filepath.Separator)) {
            // This is now allowed - current folder is a subdirectory of another
            // The model layer will handle this appropriately
            continue
        }
        
        // Check if other path is a subdirectory of current path
        if strings.HasPrefix(otherPath, currentPath+string(filepath.Separator)) {
            // This is now allowed - another folder is a subdirectory of current
            // The model layer will handle this appropriately
            continue
        }
    }
    
    return nil
}
```

## GUI Changes

### Path Validation Directive

Update the path validation directive to allow nesting but show warnings:

```javascript
// In pathIsSubDirDirective.js
if (isSubDir(scope.folders[folderID].path, viewValue)) {
    // Show informational message but don't prevent saving
    scope.folderPathErrors.isSub = true;
    scope.folderPathErrors.message = "Folder path note: This folder is a subdirectory of folder '" + 
        (scope.folders[folderID].label || folderID) + "'. This configuration is now supported but please ensure proper organization.";
}

if (viewValue !== "" && isSubDir(viewValue, scope.folders[folderID].path)) {
    // Show informational message but don't prevent saving
    scope.folderPathErrors.isParent = true;
    scope.folderPathErrors.message = "Folder path note: Folder '" + 
        (scope.folders[folderID].label || folderID) + "' is a subdirectory of this folder. This configuration is now supported but please ensure proper organization.";
}
```

## Business Logic Layer

### Model Layer Changes

Update the model layer to handle nested folders appropriately:

1. **Scanning Logic**: Implement hierarchical scanning that properly handles parent-child relationships
   - Parent folders should scan their contents but respect child folder boundaries
   - Child folders should be scanned independently without interference from parent scans
   - Implement proper exclusion patterns to prevent double scanning of files
   - Use filesystem notifications to efficiently track changes in nested structures

2. **Ignore Patterns**: Ensure ignore patterns work correctly across nested folders
   - Parent folder ignore patterns should not affect child folders
   - Child folder ignore patterns should be independent of parent patterns
   - Implement proper inheritance mechanisms where appropriate
   - Ensure .stignore files in parent folders don't unintentionally exclude child folders

3. **File Operations**: Prevent file operations in parent folders from affecting child folders
   - Ensure deletions in parent folders properly handle child folder boundaries
   - Implement safeguards to prevent accidental modification of child folder contents
   - Ensure proper handling of moves and renames across folder boundaries
   - Validate all file operations to maintain folder integrity

4. **Indexing**: Ensure proper indexing of files in nested folder structures
   - Implement hierarchical indexing that maintains clear boundaries between folders
   - Ensure database entries correctly reflect folder relationships
   - Prevent index conflicts between parent and child folders
   - Use separate index namespaces for each folder to avoid collisions

### Synchronization Logic

Implement safeguards in the synchronization logic:

1. **Conflict Detection**: Enhanced conflict detection for nested folders
   - Implement proper conflict resolution that respects folder boundaries
   - Ensure conflicts in child folders don't affect parent folder synchronization
   - Handle cross-folder conflicts appropriately
   - Use folder-specific metadata to track synchronization state

2. **File Transfer**: Ensure file transfers work correctly with nested structures
   - Implement proper transfer queuing that respects folder hierarchies
   - Ensure bandwidth allocation works correctly across nested folders
   - Handle transfer failures in nested structures appropriately
   - Prioritize transfers to maintain consistency across folder boundaries

3. **Database Handling**: Properly handle database entries for files in nested folders
   - Implement proper database partitioning by folder
   - Ensure database queries correctly handle nested folder structures
   - Prevent database corruption from cross-folder operations
   - Use transactions to ensure atomic operations across related folders

4. **Block Synchronization**: Ensure block-level synchronization works correctly with nested folders
   - Implement proper block indexing that respects folder boundaries
   - Ensure block deduplication works correctly across nested structures
   - Handle block conflicts in nested folders appropriately
   - Maintain separate block indexes for each folder to prevent cross-contamination

## Data Models

### Folder Configuration Schema

No changes needed to the folder configuration schema as nesting is now automatically supported:

```xml
<folder id="ahqrm-5jgc7" path="D:\Syncthing\Syncthing">
    <!-- folder configuration -->
</folder>
```

```json
{
    "id": "ahqrm-5jgc7",
    "path": "D:\\Syncthing\\Syncthing"
}
```

## Testing

### Unit Tests

Add new unit tests for the modified path validation logic:

```go
func TestCheckPathOverlapsWithNesting(t *testing.T) {
    // Test cases for folder path overlap detection with nesting allowed
    testCases := []struct {
        name          string
        folder        FolderConfiguration
        allFolders    []FolderConfiguration
        expectError   bool
    }{
        {
            name: "nested folders - parent contains child",
            folder: FolderConfiguration{
                ID:   "folder1",
                Path: "/home/user/folder",
            },
            allFolders: []FolderConfiguration{
                {
                    ID:   "folder2",
                    Path: "/home/user/folder/subfolder",
                },
            },
            expectError: false,
        },
        {
            name: "nested folders - child in parent",
            folder: FolderConfiguration{
                ID:   "folder1",
                Path: "/home/user/folder/subfolder",
            },
            allFolders: []FolderConfiguration{
                {
                    ID:   "folder2",
                    Path: "/home/user/folder",
                },
            },
            expectError: false,
        },
        {
            name: "identical paths still not allowed",
            folder: FolderConfiguration{
                ID:   "folder1",
                Path: "/home/user/folder",
            },
            allFolders: []FolderConfiguration{
                {
                    ID:   "folder2",
                    Path: "/home/user/folder",
                },
            },
            expectError: true,
        },
    }
    
    // ... test implementation
}
```

### Integration Tests

Create integration tests for nested folders:

1. Test synchronization between nested folders
2. Test conflict resolution in nested scenarios
3. Test ignore patterns with nested folders
4. Test database operations with nested folders
5. Test scanning performance with nested folders
6. Test file operations across folder boundaries

## Security Considerations

1. **Path Traversal**: Ensure nested folders don't introduce path traversal vulnerabilities
   - Implement proper path validation to prevent access to unauthorized directories
   - Ensure symlinks and junction points are handled securely
   - Validate all folder paths against allowed base directories
   - Implement canonical path resolution to prevent directory traversal attacks

2. **Access Control**: Ensure folder permissions are properly enforced for nested folders
   - Implement proper permission inheritance mechanisms
   - Ensure device-level permissions are respected across folder boundaries
   - Validate that users can only access folders they have permission to
   - Ensure that child folders cannot bypass parent folder restrictions

3. **Resource Exhaustion**: Prevent excessive resource consumption with nested folders
   - Implement depth limits for folder nesting to prevent infinite recursion
   - Monitor memory and CPU usage during scanning of deeply nested structures
   - Implement timeouts for operations on deeply nested folder structures
   - Limit the total number of nested folders to prevent resource exhaustion

## Performance Considerations

1. **Scanning Performance**: Ensure folder scanning performance isn't significantly impacted
   - Implement optimized scanning algorithms that respect folder boundaries
   - Use efficient data structures for tracking nested folder relationships
   - Implement caching mechanisms for frequently accessed nested folder metadata
   - Use parallel scanning for independent folder branches

2. **Memory Usage**: Monitor memory usage with deeply nested structures
   - Implement memory-efficient data structures for folder relationship tracking
   - Use streaming algorithms where possible to reduce memory footprint
   - Implement proper garbage collection for temporary folder scanning data
   - Limit the depth of folder nesting to control memory usage

3. **Database Performance**: Ensure database queries remain efficient
   - Implement proper indexing strategies for nested folder queries
   - Optimize database schemas to handle folder relationship data efficiently
   - Use query optimization techniques for nested folder operations
   - Cache frequently accessed folder relationship data

## Migration Plan

1. **Backward Compatibility**: Existing configurations remain fully compatible
2. **Documentation Updates**: Update user documentation to reflect the new automatic nesting support
3. **GUI Updates**: Update the GUI to show informational messages about nesting instead of errors
4. **Testing**: Comprehensive testing of the new functionality

## Example Usage

For the original error scenario:
- Folder "ahqrm-5jgc7": "D:\Syncthing\Syncthing"
- Folder "p6qu4-ks3vs": "D:\Syncthing\Syncthing\tea"

This configuration would now be automatically valid with informational messages in the GUI instead of errors.