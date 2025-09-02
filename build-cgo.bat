@echo off
setlocal

set VERSION=v2.0.4
echo Building Syncthing version: %VERSION%

set CGO_ENABLED=1
set CC=x86_64-w64-mingw32-gcc

echo CGO_ENABLED=%CGO_ENABLED%
echo CC=%CC%

REM Check if goversioninfo is available
where goversioninfo >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo goversioninfo not found, installing...
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
    if %ERRORLEVEL% NEQ 0 (
        echo Failed to install goversioninfo. Windows binaries will not have file information encoded.
    )
)

REM Check if logo.ico exists
if not exist "assets\logo.ico" (
    echo Warning: assets\logo.ico not found. Windows binaries will not have an icon.
)

if not exist "bin" mkdir bin

REM Build Syncthing using build.go script which will handle icon embedding
echo Building Syncthing with embedded resources using build.go...
echo Using CGO_ENABLED=%CGO_ENABLED% and CC=%CC%
go run build.go -goos windows -goarch amd64 -force-cgo -tags forcecgo build syncthing

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    if exist "syncthing.exe" (
        move "syncthing.exe" "bin\syncthing-cgo.exe"
        bin\syncthing-cgo.exe --version
    ) else (
        echo Executable not found in current directory
    )
) else (
    echo Build failed with error code %ERRORLEVEL%!
    exit /b %ERRORLEVEL%
)

exit /b 0