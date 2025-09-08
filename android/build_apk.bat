@echo off
cd /d "%~dp0"
echo Cleaning project...
call gradlew.bat clean
echo Building debug APK...
call gradlew.bat assembleDebug
echo Build complete!
pause