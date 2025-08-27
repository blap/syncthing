# Windows CGO Build Guide for Syncthing

## Problem: "panic: no console" with modernc.org/libc

When building and running Syncthing on Windows, you may encounter the following panic error:

```
panic: no console
goroutine 1 [running]:
modernc.org/libc.newFile(0x7ff7d6624100?, 0xd58902c0?)
    modernc.org/libc@v1.66.3/libc_windows.go:362 +0x9e
modernc.org/libc.init()
    modernc.org/libc@v1.66.3/libc.go:61 +0x1b1c
```

### Root Cause

This issue occurs due to the following chain of dependencies:

1. Syncthing uses SQLite for its database storage
2. When CGO is enabled on Windows, Syncthing uses `github.com/mattn/go-sqlite3` driver
3. When CGO is disabled (default for cross-compilation), Syncthing uses `modernc.org/sqlite` driver
4. The `modernc.org/sqlite` driver depends on `modernc.org/libc` package
5. The `modernc.org/libc` package attempts to initialize console functionality on Windows
6. When Syncthing is run with `--no-console` flag or in certain Windows environments, no console is available
7. This causes the panic during initialization

### Solution

To resolve this issue, CGO must be explicitly disabled when building for Windows to ensure the `modernc.org/sqlite` driver (and its problematic libc dependency) is not used.

#### Building with the provided build scripts

The build scripts in the repository have been updated to automatically disable CGO for Windows builds:

1. **Windows (PowerShell)**: Use `build.ps1`
2. **Linux/macOS/Unix**: Use `build.sh`

These scripts will automatically set `CGO_ENABLED=0` when building for Windows targets.

#### Manual building

If building manually, ensure you set the environment variables:

```bash
# For Windows targets
GOOS=windows CGO_ENABLED=0 go build ./cmd/syncthing

# Or on Windows command prompt
set CGO_ENABLED=0
go build ./cmd/syncthing

# Or on Windows PowerShell
$env:CGO_ENABLED = "0"
go build ./cmd/syncthing
```

#### Why This Solution Works

By setting `CGO_ENABLED=0`:

1. The build system uses the pure Go implementation of SQLite (`modernc.org/sqlite`)
2. However, the build constraints in the code ensure that the correct driver is selected at compile time
3. The `db_open_nocgo.go` file (with `!cgo` build tag) is used instead of `db_open_cgo.go`
4. This avoids the problematic libc initialization while still providing SQLite functionality

#### Additional Notes

- This solution only affects Windows builds; other platforms are unaffected
- Performance may be slightly different between CGO-enabled and CGO-disabled builds, but the difference is typically negligible for Syncthing's use case
- All functionality remains the same; this is purely a build configuration change

## Forcing CGO Compilation in Syncthing (Advanced)

By default, Syncthing disables CGO on Windows builds to avoid issues with the `modernc.org/libc` package which can cause a "panic: no console" error. However, if you need to force CGO compilation for testing or debugging purposes, you can use the following approaches:

### Method 1: Using the build script (Recommended)

The easiest way to build Syncthing with CGO-enabled SQLite driver is to use the provided batch script:

```cmd
build-cgo.bat [version]
```

This will create a binary at `bin\syncthing-cgo.exe` that actually enables CGO and uses the CGO-enabled SQLite driver. If you provide a version parameter, it will be embedded in the binary.

Examples:
```cmd
# Build with default version (unknown-dev)
build-cgo.bat

# Build with custom version
build-cgo.bat v2.0.3-custom
```

### Method 2: Direct Go Build with forcecgo tag

To build Syncthing with CGO-enabled SQLite driver directly:

```bash
set CGO_ENABLED=1
go build -tags forcecgo -ldflags "-X github.com/syncthing/syncthing/lib/build.Version=YOUR_VERSION" -o syncthing-cgo.exe github.com/syncthing/syncthing/cmd/syncthing
```

This will produce a binary that uses the CGO-enabled SQLite driver (`github.com/mattn/go-sqlite3`) and will show `[cgo-sqlite, forcecgo-build]` in the version output.

### Method 3: Using the build script with force-cgo flag

You can also use the build script with the `-force-cgo` flag:

```bash
go run build.go -force-cgo -version YOUR_VERSION build syncthing
```

Note: This method requires a properly configured C compiler environment. On Windows, you may need to install and configure a C compiler like MinGW or Visual Studio Build Tools.

### How it works

The solution works by:

1. Creating a new file `internal/db/sqlite/db_open_forcecgo.go` with the build constraint `//go:build forcecgo`
2. Modifying `internal/db/sqlite/db_open_cgo.go` to exclude `forcecgo` builds with the constraint `//go:build cgo && !forcecgo`
3. Modifying `internal/db/sqlite/db_open_nocgo.go` to exclude `forcecgo` builds with the constraint `//go:build !cgo && !wazero && !forcecgo`
4. Adding logic to enable CGO and use the forcecgo tag to select the CGO-enabled SQLite driver
5. The resulting binary actually enables CGO and uses the real CGO-enabled SQLite driver, not just a stub

This approach allows you to use the CGO-enabled SQLite driver while avoiding the "panic: no console" error that occurs with the `modernc.org/libc` package.