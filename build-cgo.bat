@echo off
setlocal

echo Building Syncthing with CGO support...

REM Set CGO environment variables
set CGO_ENABLED=1

echo CGO_ENABLED=%CGO_ENABLED%

if not exist "bin" mkdir bin

REM Build Syncthing directly with go build command
echo Building Syncthing with CGO support using direct go build...

go build -o bin\syncthing-cgo.exe -tags forcecgo ./cmd/syncthing

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    bin\syncthing-cgo.exe --version
) else (
    echo Build failed with error code %ERRORLEVEL%!
    exit /b %ERRORLEVEL%
)

exit /b 0