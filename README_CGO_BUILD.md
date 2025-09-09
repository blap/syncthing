# CGO Build Instructions for Syncthing on Windows

This document provides clear instructions for building Syncthing with CGO support on Windows to avoid the "panic: no console" error from the modernc.org/libc package.

## Prerequisites

1. Go 1.24 or later installed and added to PATH
2. MinGW-w64 GCC compiler installed (Strawberry Perl includes this)
3. Proper environment setup

## Building with CGO Support

### Using the build-cgo.bat script

The project includes a dedicated batch script for building with CGO support:

```batch
build-cgo.bat
```

This script:
- Verifies Go is installed
- Sets the required environment variables:
  - `CGO_ENABLED=1`
  - `CC=C:\Strawberry\c\bin\x86_64-w64-mingw32-gcc.exe`
- Builds Syncthing with the `-force-cgo` flag and `forcecgo` tag

### Manual build process

If you prefer to build manually:

```batch
set CGO_ENABLED=1
set CC=C:\Strawberry\c\bin\x86_64-w64-mingw32-gcc.exe
go run build.go -force-cgo -tags forcecgo build syncthing
```

## Why CGO is needed

Syncthing disables CGO by default on Windows to avoid issues with the modernc.org/libc package which can cause a "panic: no console" error. When CGO is enabled with the proper compiler, Syncthing uses the github.com/mattn/go-sqlite3 package instead of modernc.org/sqlite, which avoids this issue.

## Troubleshooting

1. **"Binary was compiled with 'CGO_ENABLED=0'" error**: Ensure the environment variables are set correctly before building
2. **Compiler not found**: Verify that MinGW-w64 GCC is installed and the path is correct
3. **"panic: no console" error**: This indicates CGO is not properly enabled or the wrong SQLite driver is being used

## File Structure

The CGO build uses specific files to control SQLite driver selection:

- `internal/db/sqlite/db_open_cgo.go` - Used when building with CGO, imports github.com/mattn/go-sqlite3
- `internal/db/sqlite/db_open_nocgo.go` - Used when building without CGO, imports modernc.org/sqlite

Build tags control which file is used during compilation.