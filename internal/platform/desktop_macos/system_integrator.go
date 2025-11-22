//go:build darwin
// +build darwin

package desktop_macos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// MacOSSystemIntegrator implements SystemIntegrator for macOS using LaunchAgent
type MacOSSystemIntegrator struct {
	serviceName string
	plistPath   string
	isUserAgent bool // true for LaunchAgent, false for LaunchDaemon
}

// NewMacOSSystemIntegrator creates a new macOS system integrator
// By default, creates a LaunchAgent (user-level service)
func NewMacOSSystemIntegrator() *MacOSSystemIntegrator {
	return &MacOSSystemIntegrator{
		isUserAgent: true,
	}
}

// NewMacOSSystemIntegratorDaemon creates a system-level LaunchDaemon
// This requires root privileges and runs at system level
func NewMacOSSystemIntegratorDaemon() *MacOSSystemIntegrator {
	return &MacOSSystemIntegrator{
		isUserAgent: false,
	}
}

// Install registers the application as a LaunchAgent or LaunchDaemon
func (m *MacOSSystemIntegrator) Install(config *platform.InstallConfig) error {
	if config == nil {
		return fmt.Errorf("install config cannot be nil")
	}
	if config.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if config.ExecutablePath == "" {
		return fmt.Errorf("executable path is required")
	}

	m.serviceName = config.ServiceName

	// Determine plist path based on agent type
	if m.isUserAgent {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		m.plistPath = filepath.Join(homeDir, "Library", "LaunchAgents", fmt.Sprintf("%s.plist", config.ServiceName))
	} else {
		m.plistPath = filepath.Join("/Library", "LaunchDaemons", fmt.Sprintf("%s.plist", config.ServiceName))
	}

	// Check if plist already exists
	if _, err := os.Stat(m.plistPath); err == nil {
		return fmt.Errorf("service %s already installed at %s", config.ServiceName, m.plistPath)
	}

	// Ensure directory exists
	plistDir := filepath.Dir(m.plistPath)
	if err := os.MkdirAll(plistDir, 0755); err != nil {
		return fmt.Errorf("failed to create plist directory: %w", err)
	}

	// Generate plist content
	plistContent := m.generatePlist(config)

	// Write plist file
	if err := os.WriteFile(m.plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the service
	if err := m.loadService(); err != nil {
		// Clean up plist file on failure
		os.Remove(m.plistPath)
		return fmt.Errorf("failed to load service: %w", err)
	}

	return nil
}

// generatePlist creates the LaunchAgent/LaunchDaemon plist content
func (m *MacOSSystemIntegrator) generatePlist(config *platform.InstallConfig) string {
	var sb strings.Builder

	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	sb.WriteString("<plist version=\"1.0\">\n")
	sb.WriteString("<dict>\n")

	// Label (required)
	sb.WriteString("\t<key>Label</key>\n")
	sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", config.ServiceName))

	// Program and arguments
	sb.WriteString("\t<key>ProgramArguments</key>\n")
	sb.WriteString("\t<array>\n")
	sb.WriteString(fmt.Sprintf("\t\t<string>%s</string>\n", config.ExecutablePath))
	for _, arg := range config.Arguments {
		sb.WriteString(fmt.Sprintf("\t\t<string>%s</string>\n", arg))
	}
	sb.WriteString("\t</array>\n")

	// Working directory
	if config.WorkingDir != "" {
		sb.WriteString("\t<key>WorkingDirectory</key>\n")
		sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", config.WorkingDir))
	}

	// User (for LaunchDaemon)
	if !m.isUserAgent && config.User != "" {
		sb.WriteString("\t<key>UserName</key>\n")
		sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", config.User))
	}

	// Run at load (auto-start)
	sb.WriteString("\t<key>RunAtLoad</key>\n")
	sb.WriteString("\t<false/>\n") // Default to manual start

	// Keep alive (restart on crash)
	sb.WriteString("\t<key>KeepAlive</key>\n")
	sb.WriteString("\t<dict>\n")
	sb.WriteString("\t\t<key>SuccessfulExit</key>\n")
	sb.WriteString("\t\t<false/>\n") // Don't restart on successful exit
	sb.WriteString("\t\t<key>Crashed</key>\n")
	sb.WriteString("\t\t<true/>\n") // Restart on crash
	sb.WriteString("\t</dict>\n")

	// Standard output and error logs
	logDir := "/tmp"
	if m.isUserAgent {
		if homeDir, err := os.UserHomeDir(); err == nil {
			logDir = filepath.Join(homeDir, "Library", "Logs", "Heimdal")
			os.MkdirAll(logDir, 0755)
		}
	} else {
		logDir = "/var/log/heimdal"
		os.MkdirAll(logDir, 0755)
	}

	sb.WriteString("\t<key>StandardOutPath</key>\n")
	sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", filepath.Join(logDir, "stdout.log")))
	sb.WriteString("\t<key>StandardErrorPath</key>\n")
	sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", filepath.Join(logDir, "stderr.log")))

	// Process type (for LaunchAgent)
	if m.isUserAgent {
		sb.WriteString("\t<key>ProcessType</key>\n")
		sb.WriteString("\t<string>Interactive</string>\n")
	}

	// Throttle interval (prevent rapid restarts)
	sb.WriteString("\t<key>ThrottleInterval</key>\n")
	sb.WriteString("\t<integer>10</integer>\n")

	sb.WriteString("</dict>\n")
	sb.WriteString("</plist>\n")

	return sb.String()
}

// Uninstall removes the LaunchAgent/LaunchDaemon
func (m *MacOSSystemIntegrator) Uninstall() error {
	if m.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Stop the service if running
	_ = m.Stop() // Ignore error if service is not running

	// Unload the service
	_ = m.unloadService() // Ignore error if service is not loaded

	// Remove plist file
	if m.plistPath != "" {
		if err := os.Remove(m.plistPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove plist file: %w", err)
		}
	}

	return nil
}

// Start begins the service
func (m *MacOSSystemIntegrator) Start() error {
	if m.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Use launchctl to start the service
	var cmd *exec.Cmd
	if m.isUserAgent {
		cmd = exec.Command("launchctl", "start", m.serviceName)
	} else {
		cmd = exec.Command("sudo", "launchctl", "start", m.serviceName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service: %w, output: %s", err, string(output))
	}

	return nil
}

// Stop halts the service
func (m *MacOSSystemIntegrator) Stop() error {
	if m.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Use launchctl to stop the service
	var cmd *exec.Cmd
	if m.isUserAgent {
		cmd = exec.Command("launchctl", "stop", m.serviceName)
	} else {
		cmd = exec.Command("sudo", "launchctl", "stop", m.serviceName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop service: %w, output: %s", err, string(output))
	}

	return nil
}

// Restart stops and starts the service
func (m *MacOSSystemIntegrator) Restart() error {
	if err := m.Stop(); err != nil {
		// Continue even if stop fails (service might not be running)
	}

	// Wait a moment before starting
	time.Sleep(1 * time.Second)

	if err := m.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// GetStatus returns the current service status
func (m *MacOSSystemIntegrator) GetStatus() (*platform.ServiceStatus, error) {
	if m.serviceName == "" {
		return &platform.ServiceStatus{
			IsRunning:   false,
			IsInstalled: false,
			AutoStart:   false,
			PID:         0,
			Uptime:      0,
		}, nil
	}

	status := &platform.ServiceStatus{
		IsRunning:   false,
		IsInstalled: false,
		AutoStart:   false,
		PID:         0,
		Uptime:      0,
	}

	// Check if plist file exists
	if m.plistPath != "" {
		if _, err := os.Stat(m.plistPath); err == nil {
			status.IsInstalled = true
		}
	}

	// Check if service is loaded and running
	var cmd *exec.Cmd
	if m.isUserAgent {
		cmd = exec.Command("launchctl", "list", m.serviceName)
	} else {
		cmd = exec.Command("sudo", "launchctl", "list", m.serviceName)
	}

	output, err := cmd.CombinedOutput()
	if err == nil {
		// Parse output to get PID and status
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Look for PID
			if strings.HasPrefix(line, "\"PID\"") {
				parts := strings.Split(line, "=")
				if len(parts) >= 2 {
					pidStr := strings.Trim(strings.TrimSpace(parts[1]), ";")
					if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
						status.PID = pid
						status.IsRunning = true
					}
				}
			}
		}
	}

	// Check if RunAtLoad is enabled (auto-start)
	if status.IsInstalled && m.plistPath != "" {
		content, err := os.ReadFile(m.plistPath)
		if err == nil {
			// Simple check for RunAtLoad true
			if strings.Contains(string(content), "<key>RunAtLoad</key>") {
				// Look for the next line after RunAtLoad
				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					if strings.Contains(line, "<key>RunAtLoad</key>") && i+1 < len(lines) {
						nextLine := strings.TrimSpace(lines[i+1])
						if strings.Contains(nextLine, "<true/>") {
							status.AutoStart = true
						}
						break
					}
				}
			}
		}
	}

	// Get uptime (approximate - macOS doesn't provide easy access to start time)
	// We would need to track this separately or parse system logs
	status.Uptime = 0

	return status, nil
}

// EnableAutoStart configures the service to start on boot
func (m *MacOSSystemIntegrator) EnableAutoStart(enabled bool) error {
	if m.serviceName == "" {
		return fmt.Errorf("service name not set")
	}
	if m.plistPath == "" {
		return fmt.Errorf("plist path not set")
	}

	// Read current plist
	content, err := os.ReadFile(m.plistPath)
	if err != nil {
		return fmt.Errorf("failed to read plist file: %w", err)
	}

	// Modify RunAtLoad value
	contentStr := string(content)

	// Find and replace RunAtLoad value
	if strings.Contains(contentStr, "<key>RunAtLoad</key>") {
		// Replace the value after RunAtLoad
		lines := strings.Split(contentStr, "\n")
		for i, line := range lines {
			if strings.Contains(line, "<key>RunAtLoad</key>") && i+1 < len(lines) {
				if enabled {
					lines[i+1] = "\t<true/>"
				} else {
					lines[i+1] = "\t<false/>"
				}
				break
			}
		}
		contentStr = strings.Join(lines, "\n")
	} else {
		// Add RunAtLoad if it doesn't exist
		// Insert before </dict>
		insertValue := "\t<key>RunAtLoad</key>\n"
		if enabled {
			insertValue += "\t<true/>\n"
		} else {
			insertValue += "\t<false/>\n"
		}
		contentStr = strings.Replace(contentStr, "</dict>", insertValue+"</dict>", 1)
	}

	// Write updated plist
	if err := os.WriteFile(m.plistPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write updated plist: %w", err)
	}

	// Reload the service to apply changes
	if err := m.unloadService(); err != nil {
		// Continue even if unload fails
	}
	if err := m.loadService(); err != nil {
		return fmt.Errorf("failed to reload service: %w", err)
	}

	return nil
}

// loadService loads the service with launchctl
func (m *MacOSSystemIntegrator) loadService() error {
	if m.plistPath == "" {
		return fmt.Errorf("plist path not set")
	}

	var cmd *exec.Cmd
	if m.isUserAgent {
		cmd = exec.Command("launchctl", "load", m.plistPath)
	} else {
		cmd = exec.Command("sudo", "launchctl", "load", m.plistPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to load service: %w, output: %s", err, string(output))
	}

	return nil
}

// unloadService unloads the service with launchctl
func (m *MacOSSystemIntegrator) unloadService() error {
	if m.plistPath == "" {
		return fmt.Errorf("plist path not set")
	}

	var cmd *exec.Cmd
	if m.isUserAgent {
		cmd = exec.Command("launchctl", "unload", m.plistPath)
	} else {
		cmd = exec.Command("sudo", "launchctl", "unload", m.plistPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unload service: %w, output: %s", err, string(output))
	}

	return nil
}

// SetServiceName sets the service name for an existing integrator instance
func (m *MacOSSystemIntegrator) SetServiceName(serviceName string) {
	m.serviceName = serviceName

	// Update plist path
	if m.isUserAgent {
		if homeDir, err := os.UserHomeDir(); err == nil {
			m.plistPath = filepath.Join(homeDir, "Library", "LaunchAgents", fmt.Sprintf("%s.plist", serviceName))
		}
	} else {
		m.plistPath = filepath.Join("/Library", "LaunchDaemons", fmt.Sprintf("%s.plist", serviceName))
	}
}

// GetPlistPath returns the path to the plist file
func (m *MacOSSystemIntegrator) GetPlistPath() string {
	return m.plistPath
}

// GetServiceName returns the service name
func (m *MacOSSystemIntegrator) GetServiceName() string {
	return m.serviceName
}

// IsUserAgent returns true if this is a LaunchAgent (user-level)
func (m *MacOSSystemIntegrator) IsUserAgent() bool {
	return m.isUserAgent
}
