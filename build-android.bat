@echo off
:: Syncthing Android Build Script for Windows (Compile Only)
:: This script builds the Android APK without installing it on a connected emulator or device

echo ========================================
echo Syncthing Android Compile Only Script
echo ========================================

:: Check if we're in the correct directory
if not exist "android\app" (
    echo Error: Cannot find android project directory
    echo Please run this script from the syncthing root directory
    pause
    exit /b 1
)

:: Set Android SDK path (adjust if needed)
set ANDROID_SDK=C:\Users\%USERNAME%\AppData\Local\Android\Sdk

:: Check if Android SDK exists
if not exist "%ANDROID_SDK%" (
    echo Warning: Android SDK not found at %ANDROID_SDK%
    echo Please ensure Android Studio is installed
)

:: Add Android tools to PATH
set PATH=%PATH%;%ANDROID_SDK%\platform-tools;%ANDROID_SDK%\emulator

echo.
echo 1. Cleaning previous builds...
echo ============================
cd android
if errorlevel 1 (
    echo Error: Failed to change to android directory
    pause
    exit /b 1
)

:: Clean previous builds
call gradlew.bat clean
if errorlevel 1 (
    echo Warning: Failed to clean previous builds, continuing anyway...
)

:: Build the debug APK
echo.
echo 2. Building Android APK...
echo ==========================
call gradlew.bat assembleDebug
if errorlevel 1 (
    echo Error: Failed to build APK
    cd ..
    pause
    exit /b 1
)

cd ..

echo.
echo ========================================
echo APK compiled successfully!
echo APK location: android\app\build\outputs\apk\debug\app-debug.apk
echo ========================================
echo.

pause