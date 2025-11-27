package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/getlantern/systray"
)

const (
	serviceName = "PinShareService"
	uiPort      = 8888 // Default UI port
)

type Tray struct {
	// Menu items
	menuOpenUI        *systray.MenuItem
	menuStatus        *systray.MenuItem
	menuIPFSStatus    *systray.MenuItem
	menuPinShareStatus *systray.MenuItem
	menuPeersStatus   *systray.MenuItem
	menuSeparator1    *systray.MenuItem
	menuStart         *systray.MenuItem
	menuStop          *systray.MenuItem
	menuRestart       *systray.MenuItem
	menuSeparator2    *systray.MenuItem
	menuSettings      *systray.MenuItem
	menuLogs          *systray.MenuItem
	menuAbout         *systray.MenuItem
	menuSeparator3    *systray.MenuItem
	menuExit          *systray.MenuItem

	// State
	serviceRunning bool
	lastError      error
}

func NewTray() *Tray {
	return &Tray{}
}

// BuildMenu creates the tray menu
func (t *Tray) BuildMenu() {
	// Open UI
	t.menuOpenUI = systray.AddMenuItem("Open PinShare UI", "Open the PinShare web interface")

	systray.AddSeparator()

	// Status
	t.menuStatus = systray.AddMenuItem("Status: Checking...", "Service status")
	t.menuStatus.Disable()

	t.menuIPFSStatus = systray.AddMenuItem("  IPFS: Unknown", "IPFS daemon status")
	t.menuIPFSStatus.Disable()

	t.menuPinShareStatus = systray.AddMenuItem("  PinShare: Unknown", "PinShare backend status")
	t.menuPinShareStatus.Disable()

	t.menuPeersStatus = systray.AddMenuItem("  Peers: Unknown", "Connected peers")
	t.menuPeersStatus.Disable()

	systray.AddSeparator()

	// Service control
	t.menuStart = systray.AddMenuItem("Start Service", "Start the PinShare service")
	t.menuStop = systray.AddMenuItem("Stop Service", "Stop the PinShare service")
	t.menuRestart = systray.AddMenuItem("Restart Service", "Restart the PinShare service")

	systray.AddSeparator()

	// Settings and logs
	t.menuSettings = systray.AddMenuItem("Settings...", "Open settings")
	t.menuLogs = systray.AddMenuItem("View Logs...", "Open log directory")

	systray.AddSeparator()

	// About
	t.menuAbout = systray.AddMenuItem("About PinShare", "About this application")

	systray.AddSeparator()

	// Exit
	t.menuExit = systray.AddMenuItem("Exit", "Exit the PinShare tray application")

	// Handle menu clicks
	go t.handleMenuClicks()

	// Initial status check
	t.updateStatus()
}

// handleMenuClicks handles menu item clicks
func (t *Tray) handleMenuClicks() {
	for {
		select {
		case <-t.menuOpenUI.ClickedCh:
			t.handleOpenUI()

		case <-t.menuStart.ClickedCh:
			t.handleStartService()

		case <-t.menuStop.ClickedCh:
			t.handleStopService()

		case <-t.menuRestart.ClickedCh:
			t.handleRestartService()

		case <-t.menuSettings.ClickedCh:
			t.handleSettings()

		case <-t.menuLogs.ClickedCh:
			t.handleViewLogs()

		case <-t.menuAbout.ClickedCh:
			t.handleAbout()

		case <-t.menuExit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

// handleOpenUI opens the PinShare UI in browser
func (t *Tray) handleOpenUI() {
	url := fmt.Sprintf("http://localhost:%d", uiPort)
	if err := openBrowser(url); err != nil {
		log.Printf("Failed to open browser: %v", err)
		showMessage("Error", "Failed to open browser")
	}
}

// handleStartService starts the service
func (t *Tray) handleStartService() {
	if err := startService(); err != nil {
		log.Printf("Failed to start service: %v", err)
		showMessage("Error", fmt.Sprintf("Failed to start service: %v", err))
	} else {
		showMessage("Success", "PinShare service started")
		time.Sleep(1 * time.Second)
		t.updateStatus()
	}
}

// handleStopService stops the service
func (t *Tray) handleStopService() {
	if err := stopService(); err != nil {
		log.Printf("Failed to stop service: %v", err)
		showMessage("Error", fmt.Sprintf("Failed to stop service: %v", err))
	} else {
		showMessage("Success", "PinShare service stopped")
		time.Sleep(1 * time.Second)
		t.updateStatus()
	}
}

// handleRestartService restarts the service
func (t *Tray) handleRestartService() {
	// Stop first
	if err := stopService(); err != nil {
		log.Printf("Failed to stop service: %v", err)
		showMessage("Error", fmt.Sprintf("Failed to stop service: %v", err))
		return
	}

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Start again
	if err := startService(); err != nil {
		log.Printf("Failed to start service: %v", err)
		showMessage("Error", fmt.Sprintf("Failed to start service: %v", err))
	} else {
		showMessage("Success", "PinShare service restarted")
		time.Sleep(1 * time.Second)
		t.updateStatus()
	}
}

// handleSettings opens the settings dialog
func (t *Tray) handleSettings() {
	log.Println("Opening settings dialog...")
	go showSettingsDialog()
}

// handleViewLogs opens the log directory
func (t *Tray) handleViewLogs() {
	// Get data directory
	programData := "C:\\ProgramData"
	logDir := fmt.Sprintf("%s\\PinShare\\logs", programData)

	if err := openBrowser(logDir); err != nil {
		log.Printf("Failed to open log directory: %v", err)
		showMessage("Error", "Failed to open log directory")
	}
}

// handleAbout shows about information
func (t *Tray) handleAbout() {
	showMessage("About PinShare", "PinShare - Decentralized IPFS Pinning Service\nVersion 1.0")
}

// UpdateStatusLoop periodically updates the status
func (t *Tray) UpdateStatusLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		t.updateStatus()
	}
}

// updateStatus updates the service status
func (t *Tray) updateStatus() {
	log.Println("Updating service status...")

	// Check service state using sc query (doesn't require admin)
	serviceState := getServiceState()
	log.Printf("Service state: %s", serviceState)

	switch serviceState {
	case "RUNNING":
		t.serviceRunning = true
		t.menuStatus.SetTitle("Status: Running")

		// Update icon/tooltip
		systray.SetTooltip("PinShare - Running")

		// Enable stop/restart, disable start
		t.menuStart.Disable()
		t.menuStop.Enable()
		t.menuRestart.Enable()

		// Check actual health via HTTP endpoints
		uiHealthy := checkHTTPHealth(fmt.Sprintf("http://localhost:%d/api/health", uiPort))
		pinshareHealthy := checkHTTPHealth("http://localhost:9090/api/health")
		ipfsHealthy := checkHTTPHealth("http://localhost:5001/api/v0/version")

		if uiHealthy {
			t.menuStatus.SetTitle("Status: Running OK")
		}

		if ipfsHealthy {
			t.menuIPFSStatus.SetTitle("  IPFS: Online")
		} else {
			t.menuIPFSStatus.SetTitle("  IPFS: Starting...")
		}

		if pinshareHealthy {
			t.menuPinShareStatus.SetTitle("  PinShare: Online")
		} else {
			t.menuPinShareStatus.SetTitle("  PinShare: Starting...")
		}

		t.menuPeersStatus.SetTitle("  Peers: Connected")

	case "STOPPED":
		t.serviceRunning = false
		t.menuStatus.SetTitle("Status: Stopped")
		t.menuIPFSStatus.SetTitle("  IPFS: Offline")
		t.menuPinShareStatus.SetTitle("  PinShare: Offline")
		t.menuPeersStatus.SetTitle("  Peers: None")

		// Enable start, disable stop
		t.menuStart.Enable()
		t.menuStop.Disable()
		t.menuRestart.Disable()

		systray.SetTooltip("PinShare - Stopped")

	case "START_PENDING":
		t.menuStatus.SetTitle("Status: Starting...")
		t.menuStart.Disable()
		t.menuStop.Disable()
		t.menuRestart.Disable()
		systray.SetTooltip("PinShare - Starting...")

	case "STOP_PENDING":
		t.menuStatus.SetTitle("Status: Stopping...")
		t.menuStart.Disable()
		t.menuStop.Disable()
		t.menuRestart.Disable()
		systray.SetTooltip("PinShare - Stopping...")

	case "NOT_INSTALLED":
		t.serviceRunning = false
		t.menuStatus.SetTitle("Status: Not Installed")
		t.menuIPFSStatus.SetTitle("  IPFS: N/A")
		t.menuPinShareStatus.SetTitle("  PinShare: N/A")
		t.menuPeersStatus.SetTitle("  Peers: N/A")
		t.menuStart.Disable()
		t.menuStop.Disable()
		t.menuRestart.Disable()
		systray.SetTooltip("PinShare - Service not installed")

	default:
		t.menuStatus.SetTitle(fmt.Sprintf("Status: %s", serviceState))
		systray.SetTooltip("PinShare - Unknown status")
	}
}

// getServiceState uses 'sc query' to get service state (doesn't require admin)
func getServiceState() string {
	cmd := exec.Command("sc", "query", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Service might not be installed
		outputStr := string(output)
		if strings.Contains(outputStr, "1060") || strings.Contains(outputStr, "does not exist") {
			return "NOT_INSTALLED"
		}
		log.Printf("sc query failed: %v, output: %s", err, outputStr)
		return "UNKNOWN"
	}

	outputStr := string(output)

	// Parse STATE from output
	// Example: "        STATE              : 4  RUNNING"
	if strings.Contains(outputStr, "RUNNING") {
		return "RUNNING"
	} else if strings.Contains(outputStr, "STOPPED") {
		return "STOPPED"
	} else if strings.Contains(outputStr, "START_PENDING") {
		return "START_PENDING"
	} else if strings.Contains(outputStr, "STOP_PENDING") {
		return "STOP_PENDING"
	} else if strings.Contains(outputStr, "PAUSED") {
		return "PAUSED"
	}

	return "UNKNOWN"
}

// checkHTTPHealth checks if an HTTP endpoint responds successfully
func checkHTTPHealth(url string) bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// runElevatedServiceCommand runs a service control command with UAC elevation
// while minimizing visible windows
func runElevatedServiceCommand(action string) error {
	log.Printf("Running elevated service command: %s %s", action, serviceName)

	// Use PowerShell to run net.exe with elevation
	// -WindowStyle Hidden hides the elevated net.exe window
	// The outer PowerShell is also hidden via SysProcAttr
	psCmd := fmt.Sprintf(
		"Start-Process -FilePath 'net.exe' -ArgumentList '%s %s' -Verb RunAs -Wait -WindowStyle Hidden",
		action, serviceName,
	)

	cmd := exec.Command("powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle", "Hidden",
		"-Command", psCmd,
	)

	// Hide the console window for the outer PowerShell process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to %s service: %v, output: %s", action, err, string(output))
		return fmt.Errorf("failed to %s service: %w", action, err)
	}

	log.Printf("Service %s command completed successfully", action)
	return nil
}

// startService starts the service using 'net start' with elevation
func startService() error {
	return runElevatedServiceCommand("start")
}

// stopService stops the service using 'net stop' with elevation
func stopService() error {
	return runElevatedServiceCommand("stop")
}
