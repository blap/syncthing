@echo off
echo Compiling APK...
cd /d "%~dp0"
call gradlew.bat assembleDebug
echo APK compilation completed!
echo The APK can be found in app\build\outputs\apk\debug\
pause