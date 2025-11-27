package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Config represents the PinShare configuration
type Config struct {
	InstallDirectory string `json:"install_directory"`
	DataDirectory    string `json:"data_directory"`
	IPFSBinary       string `json:"ipfs_binary"`
	PinShareBinary   string `json:"pinshare_binary"`

	IPFSAPIPort     int `json:"ipfs_api_port"`
	IPFSGatewayPort int `json:"ipfs_gateway_port"`
	IPFSSwarmPort   int `json:"ipfs_swarm_port"`
	PinShareAPIPort int `json:"pinshare_api_port"`
	PinShareP2PPort int `json:"pinshare_p2p_port"`
	UIPort          int `json:"ui_port"`

	OrgName   string `json:"org_name"`
	GroupName string `json:"group_name"`

	SkipVirusTotal bool `json:"skip_virus_total"`
	EnableCache    bool `json:"enable_cache"`
	ArchiveNode    bool `json:"archive_node"`

	EncryptionKey string `json:"encryption_key"`
	LogLevel      string `json:"log_level"`
	LogFilePath   string `json:"log_file_path"`
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "PinShare", "config.json")
}

// loadConfig loads configuration from file
func loadConfig() (*Config, error) {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// saveConfig saves configuration to file
func saveConfig(config *Config) error {
	configPath := getConfigPath()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// showSettingsDialog displays a PowerShell-based settings dialog
func showSettingsDialog() {
	config, err := loadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		showMessage("Error", fmt.Sprintf("Failed to load configuration: %v", err))
		return
	}

	// Create PowerShell script for settings dialog
	psScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$form = New-Object System.Windows.Forms.Form
$form.Text = 'PinShare Settings'
$form.Size = New-Object System.Drawing.Size(450, 520)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false

$y = 20

# Organization Settings Header
$lblHeader1 = New-Object System.Windows.Forms.Label
$lblHeader1.Location = New-Object System.Drawing.Point(10, $y)
$lblHeader1.Size = New-Object System.Drawing.Size(200, 20)
$lblHeader1.Text = 'Organization Settings'
$lblHeader1.Font = New-Object System.Drawing.Font('Segoe UI', 9, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($lblHeader1)
$y += 25

# Organization Name
$lblOrg = New-Object System.Windows.Forms.Label
$lblOrg.Location = New-Object System.Drawing.Point(10, $y)
$lblOrg.Size = New-Object System.Drawing.Size(120, 20)
$lblOrg.Text = 'Organization:'
$form.Controls.Add($lblOrg)

$txtOrg = New-Object System.Windows.Forms.TextBox
$txtOrg.Location = New-Object System.Drawing.Point(140, $y)
$txtOrg.Size = New-Object System.Drawing.Size(280, 20)
$txtOrg.Text = '%s'
$form.Controls.Add($txtOrg)
$y += 30

# Group Name
$lblGroup = New-Object System.Windows.Forms.Label
$lblGroup.Location = New-Object System.Drawing.Point(10, $y)
$lblGroup.Size = New-Object System.Drawing.Size(120, 20)
$lblGroup.Text = 'Group:'
$form.Controls.Add($lblGroup)

$txtGroup = New-Object System.Windows.Forms.TextBox
$txtGroup.Location = New-Object System.Drawing.Point(140, $y)
$txtGroup.Size = New-Object System.Drawing.Size(280, 20)
$txtGroup.Text = '%s'
$form.Controls.Add($txtGroup)
$y += 35

# Ports Header
$lblHeader2 = New-Object System.Windows.Forms.Label
$lblHeader2.Location = New-Object System.Drawing.Point(10, $y)
$lblHeader2.Size = New-Object System.Drawing.Size(200, 20)
$lblHeader2.Text = 'Port Configuration'
$lblHeader2.Font = New-Object System.Drawing.Font('Segoe UI', 9, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($lblHeader2)
$y += 25

# UI Port
$lblUIPort = New-Object System.Windows.Forms.Label
$lblUIPort.Location = New-Object System.Drawing.Point(10, $y)
$lblUIPort.Size = New-Object System.Drawing.Size(80, 20)
$lblUIPort.Text = 'UI Port:'
$form.Controls.Add($lblUIPort)

$txtUIPort = New-Object System.Windows.Forms.TextBox
$txtUIPort.Location = New-Object System.Drawing.Point(100, $y)
$txtUIPort.Size = New-Object System.Drawing.Size(60, 20)
$txtUIPort.Text = '%d'
$form.Controls.Add($txtUIPort)

# API Port
$lblAPIPort = New-Object System.Windows.Forms.Label
$lblAPIPort.Location = New-Object System.Drawing.Point(170, $y)
$lblAPIPort.Size = New-Object System.Drawing.Size(80, 20)
$lblAPIPort.Text = 'API Port:'
$form.Controls.Add($lblAPIPort)

$txtAPIPort = New-Object System.Windows.Forms.TextBox
$txtAPIPort.Location = New-Object System.Drawing.Point(260, $y)
$txtAPIPort.Size = New-Object System.Drawing.Size(60, 20)
$txtAPIPort.Text = '%d'
$form.Controls.Add($txtAPIPort)

# P2P Port
$lblP2PPort = New-Object System.Windows.Forms.Label
$lblP2PPort.Location = New-Object System.Drawing.Point(330, $y)
$lblP2PPort.Size = New-Object System.Drawing.Size(50, 20)
$lblP2PPort.Text = 'P2P:'
$form.Controls.Add($lblP2PPort)

$txtP2PPort = New-Object System.Windows.Forms.TextBox
$txtP2PPort.Location = New-Object System.Drawing.Point(360, $y)
$txtP2PPort.Size = New-Object System.Drawing.Size(60, 20)
$txtP2PPort.Text = '%d'
$form.Controls.Add($txtP2PPort)
$y += 35

# Features Header
$lblHeader3 = New-Object System.Windows.Forms.Label
$lblHeader3.Location = New-Object System.Drawing.Point(10, $y)
$lblHeader3.Size = New-Object System.Drawing.Size(200, 20)
$lblHeader3.Text = 'Features'
$lblHeader3.Font = New-Object System.Drawing.Font('Segoe UI', 9, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($lblHeader3)
$y += 25

# Skip VirusTotal
$chkSkipVT = New-Object System.Windows.Forms.CheckBox
$chkSkipVT.Location = New-Object System.Drawing.Point(10, $y)
$chkSkipVT.Size = New-Object System.Drawing.Size(400, 20)
$chkSkipVT.Text = 'Skip VirusTotal scanning (for private networks)'
$chkSkipVT.Checked = $%s
$form.Controls.Add($chkSkipVT)
$y += 25

# Enable Cache
$chkCache = New-Object System.Windows.Forms.CheckBox
$chkCache.Location = New-Object System.Drawing.Point(10, $y)
$chkCache.Size = New-Object System.Drawing.Size(200, 20)
$chkCache.Text = 'Enable file caching'
$chkCache.Checked = $%s
$form.Controls.Add($chkCache)

# Archive Node
$chkArchive = New-Object System.Windows.Forms.CheckBox
$chkArchive.Location = New-Object System.Drawing.Point(220, $y)
$chkArchive.Size = New-Object System.Drawing.Size(200, 20)
$chkArchive.Text = 'Run as archive node'
$chkArchive.Checked = $%s
$form.Controls.Add($chkArchive)
$y += 35

# Paths Header
$lblHeader4 = New-Object System.Windows.Forms.Label
$lblHeader4.Location = New-Object System.Drawing.Point(10, $y)
$lblHeader4.Size = New-Object System.Drawing.Size(200, 20)
$lblHeader4.Text = 'Paths'
$lblHeader4.Font = New-Object System.Drawing.Font('Segoe UI', 9, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($lblHeader4)
$y += 25

# Data Directory
$lblDataDir = New-Object System.Windows.Forms.Label
$lblDataDir.Location = New-Object System.Drawing.Point(10, $y)
$lblDataDir.Size = New-Object System.Drawing.Size(120, 20)
$lblDataDir.Text = 'Data Directory:'
$form.Controls.Add($lblDataDir)

$txtDataDir = New-Object System.Windows.Forms.TextBox
$txtDataDir.Location = New-Object System.Drawing.Point(140, $y)
$txtDataDir.Size = New-Object System.Drawing.Size(280, 20)
$txtDataDir.Text = '%s'
$form.Controls.Add($txtDataDir)
$y += 30

# Log Level
$lblLogLevel = New-Object System.Windows.Forms.Label
$lblLogLevel.Location = New-Object System.Drawing.Point(10, $y)
$lblLogLevel.Size = New-Object System.Drawing.Size(120, 20)
$lblLogLevel.Text = 'Log Level:'
$form.Controls.Add($lblLogLevel)

$cmbLogLevel = New-Object System.Windows.Forms.ComboBox
$cmbLogLevel.Location = New-Object System.Drawing.Point(140, $y)
$cmbLogLevel.Size = New-Object System.Drawing.Size(100, 20)
$cmbLogLevel.DropDownStyle = 'DropDownList'
$cmbLogLevel.Items.AddRange(@('debug', 'info', 'warn', 'error'))
$cmbLogLevel.SelectedItem = '%s'
$form.Controls.Add($cmbLogLevel)
$y += 45

# Note about restart
$lblNote = New-Object System.Windows.Forms.Label
$lblNote.Location = New-Object System.Drawing.Point(10, $y)
$lblNote.Size = New-Object System.Drawing.Size(400, 40)
$lblNote.Text = 'Note: Changes to ports and paths require a service restart to take effect.'
$lblNote.ForeColor = [System.Drawing.Color]::Gray
$form.Controls.Add($lblNote)
$y += 50

# Buttons
$btnSave = New-Object System.Windows.Forms.Button
$btnSave.Location = New-Object System.Drawing.Point(250, $y)
$btnSave.Size = New-Object System.Drawing.Size(80, 30)
$btnSave.Text = 'Save'
$btnSave.DialogResult = [System.Windows.Forms.DialogResult]::OK
$form.Controls.Add($btnSave)

$btnCancel = New-Object System.Windows.Forms.Button
$btnCancel.Location = New-Object System.Drawing.Point(340, $y)
$btnCancel.Size = New-Object System.Drawing.Size(80, 30)
$btnCancel.Text = 'Cancel'
$btnCancel.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
$form.Controls.Add($btnCancel)

$form.AcceptButton = $btnSave
$form.CancelButton = $btnCancel

$result = $form.ShowDialog()

if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
    # Output the values for Go to parse
    Write-Output "ORG=$($txtOrg.Text)"
    Write-Output "GROUP=$($txtGroup.Text)"
    Write-Output "UI_PORT=$($txtUIPort.Text)"
    Write-Output "API_PORT=$($txtAPIPort.Text)"
    Write-Output "P2P_PORT=$($txtP2PPort.Text)"
    Write-Output "SKIP_VT=$($chkSkipVT.Checked)"
    Write-Output "CACHE=$($chkCache.Checked)"
    Write-Output "ARCHIVE=$($chkArchive.Checked)"
    Write-Output "DATA_DIR=$($txtDataDir.Text)"
    Write-Output "LOG_LEVEL=$($cmbLogLevel.SelectedItem)"
    Write-Output "SAVED=true"
} else {
    Write-Output "SAVED=false"
}
`,
		escapeForPS(config.OrgName),
		escapeForPS(config.GroupName),
		config.UIPort,
		config.PinShareAPIPort,
		config.PinShareP2PPort,
		boolToPS(config.SkipVirusTotal),
		boolToPS(config.EnableCache),
		boolToPS(config.ArchiveNode),
		escapeForPS(config.DataDirectory),
		config.LogLevel,
	)

	// Run PowerShell with hidden window
	cmd := exec.Command("powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-Command", psScript,
	)

	// Don't hide window - we need to show the dialog
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: false,
	}

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Settings dialog error: %v", err)
		return
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	values := make(map[string]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			value := line[idx+1:]
			values[key] = value
		}
	}

	// Check if user clicked Save
	if values["SAVED"] != "True" {
		log.Println("Settings dialog cancelled")
		return
	}

	// Update config
	config.OrgName = values["ORG"]
	config.GroupName = values["GROUP"]
	config.DataDirectory = values["DATA_DIR"]
	config.LogLevel = values["LOG_LEVEL"]

	if port, err := strconv.Atoi(values["UI_PORT"]); err == nil {
		config.UIPort = port
	}
	if port, err := strconv.Atoi(values["API_PORT"]); err == nil {
		config.PinShareAPIPort = port
	}
	if port, err := strconv.Atoi(values["P2P_PORT"]); err == nil {
		config.PinShareP2PPort = port
	}

	config.SkipVirusTotal = values["SKIP_VT"] == "True"
	config.EnableCache = values["CACHE"] == "True"
	config.ArchiveNode = values["ARCHIVE"] == "True"

	// Save config
	if err := saveConfig(config); err != nil {
		log.Printf("Failed to save config: %v", err)
		showMessage("Error", fmt.Sprintf("Failed to save configuration: %v", err))
		return
	}

	log.Println("Settings saved successfully")
	showMessage("Settings", "Configuration saved. Restart the service for changes to take effect.")
}

// escapeForPS escapes a string for use in PowerShell
func escapeForPS(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "`", "``")
	return s
}

// boolToPS converts a bool to PowerShell boolean string
func boolToPS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
