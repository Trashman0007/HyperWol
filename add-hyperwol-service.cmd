@echo off
:: Check for admin privileges
net session >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo ERROR: This script must be run as Administrator.
    echo Right-click and select "Run as administrator".
    pause
    exit /b 1
)

echo Installing HyperWol service...
:: Set working directory to script's location
cd /d %~dp0
:: Clean any existing service
echo Removing existing HyperWol service (if any)...
start /wait sc delete HyperWol >nul 2>&1

:: Install service using NSSM
echo Installing service...
start /wait nssm install HyperWol "%CD%\hyperwol.exe"


:: Configure service
echo Configuring service...
nssm set HyperWol DisplayName "HyperWol Wake-on-LAN Service"
nssm set HyperWol Description "Listens for WoL packets to start Hyper-V VMs"
nssm set HyperWol Start SERVICE_AUTO_START

net start HyperWol

echo.
echo Installation complete. Press any key to exit.
pause
exit /b 0