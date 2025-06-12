@echo off
echo This script must be run as Administrator.
echo Right-click and select "Run as administrator" if not already elevated.
echo.
pause
echo Removing HyperWol service...
cd /d "%~dp0"
sc delete HyperWol
if %ERRORLEVEL% equ 0 (
    echo Service removed successfully.
) else (
    echo Failed to remove service.
)
pause