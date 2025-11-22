//go:build linux
// +build linux

package desktop_linux

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

// LinuxSystemIntegrator implements SystemIntegrator for Linux using systemd user service
type LinuxSystemIntegrator struct {
	serviceName string
	unitPath    string
}

// NewLinuxSystemIntegrator creates a new Linux system integrator
func NewLinuxSystemIntegrator() *LinuxSystemIntegrator {
	return &LinuxSystemIntegrator{}
}

// Install registers the application as a systemd user service
func (l *LinuxSystemIntegrator) Install(config *platform.InstallConfig) error {
	if config == nil {
		return fmt.Errorf("install config cannot be nil")
	}
	if config.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if config.ExecutablePath == "" {
		return fmt.Errorf("executable path is required")
	}

	l.serviceName = config.ServiceName

	// Determine systemd user unit path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	l.unitPath = filepath.Join(homeDir, ".config", "systemd", "user", fmt.Sprintf("%s.service", config.ServiceName))

	// Check if unit file already exists
	if _, err := os.Stat(l.unitPath); err == nil {
		return fmt.Errorf("service %s already installed at %s", config.ServiceName, l.unitPath)
	}

	// Ensure directory exists
	unitDir := filepath.Dir(l.unitPath)
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	// Generate unit file content
	unitContent := l.generateUnitFile(config)

	// Write unit file
	if err := os.WriteFile(l.unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	// Reload systemd daemon
	if err := l.reloadDaemon(); err != nil {
		// Clean up unit file on failure
		os.Remove(l.unitPath)
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

// generateUnitFile creates the systemd unit file content
func (l *LinuxSystemIntegrator) generateUnitFile(config *platform.InstallConfig) string {
	var sb strings.Builder

	// [Unit] section
	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=%s\n", config.Description))
	sb.WriteString("After=network.target\n")
	sb.WriteString("\n")

	// [Service] section
	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")

	// Build ExecStart command
	execStart := config.ExecutablePath
	if len(config.Arguments) > 0 {
		execStart += " " + strings.Join(config.Arguments, " ")
	}
	sb.WriteString(fmt.Sprintf("ExecStart=%s\n", execStart))

	// Working directory
	if config.WorkingDir != "" {
		sb.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", config.WorkingDir))
	}

	// Restart policy
	sb.WriteString("Restart=on-failure\n")
	sb.WriteString("RestartSec=10s\n")

	// Standard output and error
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".local", "share", "heimdal", "logs")
	os.MkdirAll(logDir, 0755)

	sb.WriteString(fmt.Sprintf("StandardOutput=append:%s\n", filepath.Join(logDir, "stdout.log")))
	sb.WriteString(fmt.Sprintf("StandardError=append:%s\n", filepath.Join(logDir, "stderr.log")))

	// Environment
	sb.WriteString("Environment=\"PATH=/usr/local/bin:/usr/bin:/bin\"\n")

	sb.WriteString("\n")

	// [Install] section
	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=default.target\n")

	return sb.String()
}

// Uninstall removes the systemd user service
func (l *LinuxSystemIntegrator) Uninstall() error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Stop the service if running
	_ = l.Stop() // Ignore error if service is not running

	// Disable the service
	_ = l.EnableAutoStart(false) // Ignore error if service is not enabled

	// Remove unit file
	if l.unitPath != "" {
		if err := os.Remove(l.unitPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove unit file: %w", err)
		}
	}

	// Reload systemd daemon
	if err := l.reloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

// Start begins the systemd user service
func (l *LinuxSystemIntegrator) Start() error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	cmd := exec.Command("systemctl", "--user", "start", l.serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service: %w, output: %s", err, string(output))
	}

	return nil
}

// Stop halts the systemd user service
func (l *LinuxSystemIntegrator) Stop() error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	cmd := exec.Command("systemctl", "--user", "stop", l.serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop service: %w, output: %s", err, string(output))
	}

	return nil
}

// Restart stops and starts the systemd user service
func (l *LinuxSystemIntegrator) Restart() error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	cmd := exec.Command("systemctl", "--user", "restart", l.serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service: %w, output: %s", err, string(output))
	}

	return nil
}

// GetStatus returns the current service status
func (l *LinuxSystemIntegrator) GetStatus() (*platform.ServiceStatus, error) {
	if l.serviceName == "" {
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

	// Check if unit file exists
	if l.unitPath != "" {
		if _, err := os.Stat(l.unitPath); err == nil {
			status.IsInstalled = true
		}
	}

	// Get service status from systemctl
	cmd := exec.Command("systemctl", "--user", "show", l.serviceName,
		"--property=ActiveState",
		"--property=MainPID",
		"--property=UnitFileState",
		"--property=ActiveEnterTimestamp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Service might not be loaded
		return status, nil
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	var activeEnterTime time.Time

	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "ActiveState":
			status.IsRunning = (value == "active")

		case "MainPID":
			if pid, err := strconv.Atoi(value); err == nil {
				status.PID = pid
			}

		case "UnitFileState":
			status.AutoStart = (value == "enabled")

		case "ActiveEnterTimestamp":
			// Parse timestamp (format: "Mon 2006-01-02 15:04:05 MST")
			if value != "" && value != "n/a" {
				// Try multiple time formats
				formats := []string{
					"Mon 2006-01-02 15:04:05 MST",
					time.RFC3339,
					time.RFC1123,
				}
				for _, format := range formats {
					if t, err := time.Parse(format, value); err == nil {
						activeEnterTime = t
						break
					}
				}
			}
		}
	}

	// Calculate uptime
	if status.IsRunning && !activeEnterTime.IsZero() {
		status.Uptime = time.Since(activeEnterTime)
	}

	return status, nil
}

// EnableAutoStart configures the service to start on boot
func (l *LinuxSystemIntegrator) EnableAutoStart(enabled bool) error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	var cmd *exec.Cmd
	if enabled {
		cmd = exec.Command("systemctl", "--user", "enable", l.serviceName)
	} else {
		cmd = exec.Command("systemctl", "--user", "disable", l.serviceName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to change auto-start setting: %w, output: %s", err, string(output))
	}

	return nil
}

// reloadDaemon reloads the systemd user daemon
func (l *LinuxSystemIntegrator) reloadDaemon() error {
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload daemon: %w, output: %s", err, string(output))
	}

	return nil
}

// SetServiceName sets the service name for an existing integrator instance
func (l *LinuxSystemIntegrator) SetServiceName(serviceName string) {
	l.serviceName = serviceName

	// Update unit path
	if homeDir, err := os.UserHomeDir(); err == nil {
		l.unitPath = filepath.Join(homeDir, ".config", "systemd", "user", fmt.Sprintf("%s.service", serviceName))
	}
}

// GetUnitPath returns the path to the systemd unit file
func (l *LinuxSystemIntegrator) GetUnitPath() string {
	return l.unitPath
}

// GetServiceName returns the service name
func (l *LinuxSystemIntegrator) GetServiceName() string {
	return l.serviceName
}
