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

REM Set environment variables for CGO with full path to compiler
set CGO_ENABLED=1
set CC=C:\Strawberry\c\bin\x86_64-w64-mingw32-gcc.exe

echo CGO_ENABLED=%CGO_ENABLED%
echo CC=%CC%

REM Build Syncthing with CGO support using build.go script
echo Building Syncthing with CGO support using build.go script...

go run build.go -force-cgo -tags forcecgo build syncthing

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    .\syncthing.exe --version
) else (
    echo Build failed with error code %ERRORLEVEL%!
    exit /b %ERRORLEVEL%
)

exit /b 0