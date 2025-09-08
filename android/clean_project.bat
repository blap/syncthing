@echo off
echo Cleaning project...
cd /d "%~dp0"
call gradlew.bat clean
echo Project cleaned successfully!
pause