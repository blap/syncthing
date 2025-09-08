# Syncthing Android Build Script for PowerShell (Compile Only)
# This script builds the Android APK without installing it on a connected emulator or device

Write-Host "========================================" -ForegroundColor Green
Write-Host "Syncthing Android Compile Only Script" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green

# Check if we're in the correct directory
if (-not (Test-Path "android\app")) {
    Write-Error "Cannot find android project directory. Please run this script from the syncthing root directory."
    pause
    exit 1
}

# Set Android SDK path (adjust if needed)
$ANDROID_SDK = "C:\Users\$env:USERNAME\AppData\Local\Android\Sdk"

# Check if Android SDK exists
if (-not (Test-Path $ANDROID_SDK)) {
    Write-Warning "Android SDK not found at $ANDROID_SDK"
    Write-Warning "Please ensure Android Studio is installed"
}

# Add Android tools to PATH
$env:PATH += ";$ANDROID_SDK\platform-tools;$ANDROID_SDK\emulator"

Write-Host "`n1. Cleaning previous builds..." -ForegroundColor Green
Write-Host "============================" -ForegroundColor Green
Set-Location android

try {
    # Clean previous builds
    & .\gradlew.bat clean
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Failed to clean previous builds, continuing anyway..."
    }
}
catch {
    Write-Warning "Failed to clean previous builds, continuing anyway..."
}

try {
    # Build the debug APK
    Write-Host "`n2. Building Android APK..." -ForegroundColor Green
    Write-Host "==========================" -ForegroundColor Green
    & .\gradlew.bat assembleDebug
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to build APK"
    }
}
catch {
    Write-Error "Failed to build APK: $_"
    Set-Location ..
    pause
    exit 1
}

Set-Location ..

Write-Host "`n========================================" -ForegroundColor Green
Write-Host "APK compiled successfully!" -ForegroundColor Green
Write-Host "APK location: android\app\build\outputs\apk\debug\app-debug.apk" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Green

pause