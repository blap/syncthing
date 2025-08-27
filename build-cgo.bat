@echo off
REM Simple script to build Syncthing with CGO-enabled SQLite driver
REM This builds a binary that actually enables CGO but avoids the console panic issue
REM For more information, see docs/windows-cgo-build-guide.md

setlocal

REM Set version from command line argument or use default
if "%1"=="" (
    set VERSION=v2.0.3
) else (
    set VERSION=%1
)

echo Building Syncthing with CGO-enabled SQLite driver...
echo Version: %VERSION%

set CGO_ENABLED=1
REM Clear CC to let Go find the compiler automatically
set CC=
go build -tags forcecgo -ldflags "-X github.com/syncthing/syncthing/lib/build.Version=%VERSION%" -o bin\syncthing-cgo.exe github.com/syncthing/syncthing/cmd/syncthing

if %ERRORLEVEL% EQU 0 (
    echo Build successful! Binary created at bin\syncthing-cgo.exe
    echo Testing the binary...
    bin\syncthing-cgo.exe --version
) else (
    echo Build failed!
    echo For more information, see docs/windows-cgo-build-guide.md
    exit /b %ERRORLEVEL%
)