package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "golang.org/x/sys/windows/svc/eventlog"
)

// Config holds the JSON configuration
type Config struct {
    Port    int64             `json:"port"`
    VMs     map[string]string `json:"vms"`
    Host    string            `json:"host"`
    Timeout string            `json:"timeout"`
}

// VMInfo holds VM name and MAC address
type VMInfo struct {
    Name string
    MAC  string
}

var (
    configMutex *sync.Mutex = &sync.Mutex{} // Protects config.json access
    configPath  string
    elog        *eventlog.Log
    configReady chan struct{} // Signals config initialization complete
)

func main() {
    // Initialize event log
    var err error
    elog, err = eventlog.Open("HyperWol")
    if err != nil {
        log.Printf("Error opening Hyper-V event log: %s", err.Error())
        os.Exit(1)
    }
    defer func() {
        if err := elog.Close(); err != nil {
            log.Printf("Error closing event log: %s", err.Error())
        }
    }()

    // Initialize config ready channel
    configReady = make(chan struct{})

    // Set config.json path to %ProgramData%\HyperWol
    programData := os.Getenv("ProgramData")
    if programData == "" {
        programData = `C:\ProgramData`
    }
    configDir := filepath.Join(programData, "HyperWol")
    if err := os.MkdirAll(configDir, 0755); err != nil {
        elog.Error(1, fmt.Sprintf("Failed to create config directory %s: %s", configDir, err.Error()))
        os.Exit(1)
    }
    configPath = filepath.Join(configDir, "config.json")
    elog.Info(1, fmt.Sprintf("Config path set to %s", configPath))

    // Initialize config.json
    if err := initializeConfig(); err != nil {
        elog.Error(1, fmt.Sprintf("Failed to initialize config: %s", err.Error()))
        if _, err := os.Stat(configPath); os.IsNotExist(err) {
            elog.Error(1, "No existing config.json found")
            os.Exit(1)
        }
        elog.Warning(1, "Using existing config.json due to initialization failure")
    }
    close(configReady) // Signal config is ready

    // Start listener
    elog.Info(1, "Running hyperwol")
    if err := runListener(); err != nil {
        elog.Error(1, fmt.Sprintf("Listener failed: %s", err.Error()))
        os.Exit(2)
    }
}

// initializeConfig creates config.json with all Hyper-V VMs on startup
func initializeConfig() error {
    configMutex.Lock()
    defer configMutex.Unlock()

    elog.Info(1, "Initializing config...")
    // Attempt to remove existing config.json
    if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
        elog.Warning(1, fmt.Sprintf("Cannot remove config.json: %s", err.Error()))
    }

    // List VMs
    vms, err := listVMs()
    if err != nil {
        elog.Error(1, fmt.Sprintf("Failed to list VMs: %s", err.Error()))
        return fmt.Errorf("failed to list VMs: %s", err.Error())
    }

    // Create new config
    config := Config{
        Port:    7,
        VMs:     make(map[string]string),
        Host:    "0.0.0.0",
        Timeout: "60",
    }
    for _, vm := range vms {
        config.VMs[strings.ToUpper(vm.MAC)] = vm.Name
    }

    // Write config.json
    configData, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        elog.Error(1, fmt.Sprintf("Failed to serialize config: %s", err.Error()))
        return fmt.Errorf("failed to serialize config: %s", err.Error())
    }
    if err := os.WriteFile(configPath, configData, 0644); err != nil {
        elog.Warning(1, fmt.Sprintf("Failed to write config.json: %s", err.Error()))
        if _, err := os.Stat(configPath); err == nil {
            if _, err := os.ReadFile(configPath); err == nil {
                elog.Info(1, "Using existing config.json")
                return nil
            }
        }
        return fmt.Errorf("failed to write config.json: %s", err.Error())
    }
    elog.Info(1, fmt.Sprintf("Created config.json with %d VMs", len(config.VMs)))
    return nil
}

// listVMs retrieves all Hyper-V VMs with names and MAC addresses
func listVMs() ([]VMInfo, error) {
    elog.Info(1, "Listing Hyper-V VMs...")
    // Get VM names
    cmd := exec.Command("powershell", "-Command", "(Get-VM | Select-Object -ExpandProperty Name) -join ','")
    output, err := cmd.CombinedOutput()
    if err != nil {
        elog.Error(1, fmt.Sprintf("Failed to list VMs: %s, error: %s", strings.TrimSpace(string(output)), err.Error()))
        return nil, fmt.Errorf("failed to list VMs: %s", strings.TrimSpace(string(output)))
    }
    vmNames := strings.Split(strings.TrimSpace(string(output)), ",")
    elog.Info(1, fmt.Sprintf("Found VM names: %v", vmNames))
    if len(vmNames) == 0 || vmNames[0] == "" {
        elog.Info(1, "No VMs found.")
        return nil, nil
    }

    // Get MAC addresses for each VM
    var vms []VMInfo
    for _, name := range vmNames {
        name = strings.TrimSpace(name)
        if name == "" {
            elog.Info(1, "Skipping empty VM name")
            continue
        }
        elog.Info(1, fmt.Sprintf("Querying MAC for VM: %s", name))
        mac, err := getVMMacAddress(name)
        if err != nil {
            elog.Warning(1, fmt.Sprintf("Skipping VM %s due to MAC error: %s", name, err.Error()))
            continue
        }
        elog.Info(1, fmt.Sprintf("VM %s MAC: %s", name, mac))
        vms = append(vms, VMInfo{Name: name, MAC: mac})
    }
    elog.Info(1, fmt.Sprintf("Total VMs found: %d", len(vms)))
    return vms, nil
}

// getVMMacAddress queries the MAC address of a VM
func getVMMacAddress(vmName string) (string, error) {
    cmd := exec.Command("powershell", "-Command", fmt.Sprintf("(Get-VMNetworkAdapter -VMName '%s').MacAddress", vmName))
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to get MAC address: %s", strings.TrimSpace(string(output)))
    }
    mac := strings.TrimSpace(string(output))
    if len(mac) != 12 {
        return "", fmt.Errorf("invalid MAC address format: %s", mac)
    }
    // Format as XX:XX:XX:XX:XX:XX
    return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
        mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12]), nil
}

// runListener listens for WoL packets and starts VMs
func runListener() error {
    // Wait for config to be ready
    elog.Info(1, "Waiting for config initialization")
    select {
    case <-configReady:
        elog.Info(1, "Config initialization complete")
    case <-time.After(10 * time.Second):
        elog.Error(1, "Timeout waiting for config initialization")
        return fmt.Errorf("timeout waiting for config initialization")
    }

    // Retry reading config.json
    var configFile []byte
    var err error
    for attempt := 1; attempt <= 5; attempt++ {
        elog.Info(1, fmt.Sprintf("Attempting to read config from %s (attempt %d)", configPath, attempt))
        configMutex.Lock()
        configFile, err = os.ReadFile(configPath)
        configMutex.Unlock()
        if err == nil {
            break
        }
        elog.Warning(1, fmt.Sprintf("Failed to read config.json (attempt %d): %s", attempt, err.Error()))
        if attempt == 5 {
            elog.Error(1, fmt.Sprintf("Failed to read config.json after %d attempts: %s", attempt, err.Error()))
            return fmt.Errorf("failed to read config.json: %s", err.Error())
        }
        time.Sleep(500 * time.Millisecond)
    }

    var config Config
    if err := json.Unmarshal(configFile, &config); err != nil {
        elog.Error(1, fmt.Sprintf("Failed to parse config.json: %s", err.Error()))
        return fmt.Errorf("failed to parse config.json: %s", err.Error())
    }

    addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
    elog.Info(1, fmt.Sprintf("Resolving UDP address %s", addr))
    udpAddr, err := net.ResolveUDPAddr("                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                