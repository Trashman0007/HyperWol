# HyperWol

HyperWol is a lightweight Windows application that listens for Wake-on-LAN (WoL) packets to start Hyper-V virtual machines (VMs) automatically. It generates a configuration file (`config.json`) listing all Hyper-V VMs and their MAC addresses, then monitors UDP port 7 for WoL packets to trigger VM startup via PowerShell. Designed for seamless integration with external scripts (e.g., PHP-based WoL triggers), HyperWol runs as a Windows service using NSSM (Non-Sucking Service Manager).

## Features
- **Automatic VM Discovery**: Detects all Hyper-V VMs and their MAC addresses, storing them in `C:\ProgramData\HyperWol\config.json`.
- **WoL Listener**: Listens on UDP port 7 for WoL magic packets and starts the corresponding VM.
- **Service Integration**: Runs as a Windows service under the Local System Account for continuous operation.
- **Event Logging**: Logs activity to Windows Event Viewer (under `HyperWol` source) for debugging.
- **Bundled Package**: Includes `hyperwol.exe`, NSSM, `wolcmd`, and scripts for easy setup and testing.

## Requirements
- Windows 10/11 or Windows Server with Hyper-V enabled.
- Administrative privileges for installation.
- Hyper-V virtual machines configured on the local computer.

## Installation
1. **Download the Package**:
   - Download `HyperWol.zip` from the [Releases](https://github.com/Trashman0007/HyperWol/releases) page.

2. **Extract the Package**:
   - Unzip `HyperWol.zip` to a directory (e.g., `C:\Program Files\HyperWol`).
   - The package contains:
     - `hyperwol.exe`: The main WoL listener application.
     - `nssm.exe`: NSSM service manager.
     - `wolcmd.exe`: WoL packet sender for testing.
     - `add-hyperwol-service.cmd`: Script to install the service.
     - `test_wol.cmd`: Script to test WoL locally.

3. **Install the Service**:
   - Navigate to the extraction directory (e.g., `C:\Program Files\HyperWol`).
   - Right-click `add-hyperwol-service.cmd` and select **Run as administrator**.
   - The script:
     - Installs the `HyperWol` service using NSSM.
     - Configures the service to auto-start under the Local System Account.
     - Starts the service.

4. **Verify Installation**:
   - Open Command Prompt as administrator and run:
     ```cmd
     sc query HyperWol
     ```
     - Ensure `STATE: 4 RUNNING`.
   - Check `C:\ProgramData\HyperWol\config.json` for VM details:
     ```cmd
     type C:\ProgramData\HyperWol\config.json
     ```

## Testing WoL Locally
Use the included `test_wol.bat` script to send a WoL packet to a Hyper-V VM on the local machine:

1. **Run the Test Script**:
   - Navigate to `C:\Program Files\HyperWol`.
   - Right-click `test_wol.cmd` and select **Run as administrator**:
     ```cmd
     test_wol.cmd
     ```
   - The script:
     - Lists all Hyper-V VMs and their MAC addresses.
     - Prompts you to select a VM (e.g., `Machine1`).
     - Sends a WoL packet to `127.0.0.1:7` using `wolcmd`.

2. **Verify VM Startup**:
   - Check the VM’s state in PowerShell:
     ```powershell
     Get-VM -Name "Machine1" | Select-Object Name, State
     ```
     - Expect `State: Running`.

## Usage
- **Trigger WoL Remotely**:
  - Use a WoL client (e.g., a PHP script) to send a magic packet to the server’s IP (e.g., `192.168.1.100`) on port 7 with the VM’s MAC address (e.g., `00:15:5D:01:64:11`).
  - Example using `wolcmd`:
    ```cmd
    wolcmd 00155D016411 192.168.1.100 255.255.255.0 7
    ```

- **Monitor Logs**:
  - Open Event Viewer (`eventvwr.msc`), navigate to Windows Logs > Application, and filter for `HyperWol`.
  - Look for messages like “Listening for WoL packets on 0.0.0.0:7” or “Successfully started VM Machine1”.

## Troubleshooting
- **Service Not Starting**:
  - Check service status:
    ```cmd
    sc query HyperWol
    ```
  - Ensure UDP port 7 is free:
    ```cmd
    netstat -ano | findstr :7
    ```
  - Check Event Viewer for `HyperWol` errors.

- **No VMs in `config.json`**:
  - Verify Hyper-V is enabled and VMs exist:
    ```powershell
    Get-VM
    ```
  - Ensure the Local System Account has Hyper-V access.

- **WoL Not Working**:
  - Confirm the VM’s network adapter supports WoL in Hyper-V settings.
  - Test locally with `test_wol.bat`.

## Uninstallation
1. Stop and remove the service:
   ```cmd
   net stop HyperWol
   sc delete HyperWol
   
## License
1. MIT License. See LICENSE for details.

## Acknowledgments

NSSM for service management.

wolcmd for WoL packet testing.


