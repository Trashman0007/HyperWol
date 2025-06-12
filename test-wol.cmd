@echo off
setlocal EnableDelayedExpansion

:: Set the path to wolcmd (assumed to be in the same directory as the script)
set "WOLCMD=%~dp0wolcmd.exe"

:: Check if wolcmd exists
if not exist "%WOLCMD%" (
    echo Error: wolcmd.exe not found in the script directory.
    pause
    exit /b 1
)

:: Get list of Hyper-V VMs and their MAC addresses
echo Querying Hyper-V virtual machines...
set "VM_COUNT=0"
:: Use a temporary file to store PowerShell output to avoid parsing issues
set "TEMP_FILE=%TEMP%\vm_list.txt"
powershell -Command "Get-VM | ForEach-Object { $vmName = $_.Name; $mac = (Get-VMNetworkAdapter -VMName $vmName).MacAddress -join ''; \"$vmName,$mac\" }" > "%TEMP_FILE%"

:: Check if PowerShell command was successful
if errorlevel 1 (
    echo Error: Failed to retrieve Hyper-V VM information. Ensure Hyper-V is installed and you have sufficient permissions.
    if exist "%TEMP_FILE%" del "%TEMP_FILE%"
    pause
    exit /b 1
)

:: Read the temporary file to populate VM list
for /f "tokens=1,2 delims=," %%a in (%TEMP_FILE%) do (
    set /a VM_COUNT+=1
    set "VM_NAME[!VM_COUNT!]=%%a"
    set "VM_MAC[!VM_COUNT!]=%%b"
)

:: Clean up temporary file
if exist "%TEMP_FILE%" del "%TEMP_FILE%"

:: Check if any VMs were found
if %VM_COUNT%==0 (
    echo No Hyper-V virtual machines found.
    pause
    exit /b 1
)

:: Display list of VMs
echo Available Virtual Machines:
echo.
for /l %%i in (1,1,%VM_COUNT%) do (
    echo %%i. !VM_NAME[%%i]! (MAC: !VM_MAC[%%i]!)
)
echo.

:: Prompt user to select a VM
set /p USER_CHOICE="Select a VM to start (1-%VM_COUNT%): "

:: Validate user input
if "!USER_CHOICE!"=="" (
    echo No selection made.
    pause
    exit /b 1
)
set /a USER_CHOICE_NUM=USER_CHOICE
if !USER_CHOICE_NUM! lss 1 (
    echo Invalid selection. Please choose a number between 1 and %VM_COUNT%.
    pause
    exit /b 1
)
if !USER_CHOICE_NUM! gtr %VM_COUNT% (
    echo Invalid selection. Please choose a number between 1 and %VM_COUNT%.
    pause
    exit /b 1
)

:: Get the selected VM's MAC address
set "SELECTED_MAC=!VM_MAC[%USER_CHOICE%]!"
:: Remove any colons or spaces from the MAC address for wolcmd
set "SELECTED_MAC=%SELECTED_MAC::=%"
set "SELECTED_MAC=%SELECTED_MAC: =%"

:: Change to script directory
cd /d "%~dp0"

:: Run wolcmd with the selected MAC address
echo Sending Wake-on-LAN packet to !VM_NAME[%USER_CHOICE%]! (MAC: %SELECTED_MAC%)...
"%WOLCMD%" %SELECTED_MAC% 127.0.0.1 255.255.255.0 7

:: Pause to allow user to see the result
pause