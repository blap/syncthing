@echo off
setlocal

echo Building Syncthing with CGO support...

REM Check if Go is installed
go version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Error: Go is not installed or not in PATH.
    echo Please install Go 1.24 or later from https://golang.org/dl/
    echo Make sure to add Go to your PATH environment variable.
    exit /b 1
)

echo Found Go: 
for /f "delims=" %%i in ('go version') do set GOVERSION=%%i
echo %GOVERSION%

REM Set CGO environment variables
set CGO_ENABLED=1

echo CGO_ENABLED=%CGO_ENABLED%

REM Check if GCC is available (needed for CGO)
gcc --version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Warning: GCC not found. You may need to install MinGW-w64 for CGO support.
    echo Continuing build anyway...
)

if not exist "bin" mkdir bin

REM Build Syncthing directly with go build command
echo Building Syncthing with CGO support using direct go build...

go build -o bin\syncthing-cgo.exe -tags forcecgo ./cmd/syncthing

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    bin\syncthing-cgo.exe --version
) else (
    echo Build failed with error code %ERRORLEVEL%!
    echo Please ensure you have a C compiler (like MinGW-w64) installed for CGO support.
    exit /b %ERRORLEVEL%
)

exit /b 0