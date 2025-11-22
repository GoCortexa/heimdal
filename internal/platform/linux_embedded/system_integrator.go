package linux_embedded

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

// LinuxEmbeddedSystemIntegrator implements SystemIntegrator for embedded Linux systems
// using systemd for service management.
type LinuxEmbeddedSystemIntegrator struct {
	serviceName string
	serviceFile string
}

// NewLinuxEmbeddedSystemIntegrator creates a new system integrator for embedded Linux
func NewLinuxEmbeddedSystemIntegrator() *LinuxEmbeddedSystemIntegrator {
	return &LinuxEmbeddedSystemIntegrator{}
}

// Install registers the application with systemd as a system service
func (l *LinuxEmbeddedSystemIntegrator) Install(config *platform.InstallConfig) error {
	if config == nil {
		return fmt.Errorf("install config is required")
	}
	if config.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if config.ExecutablePath == "" {
		return fmt.Errorf("executable path is required")
	}

	l.serviceName = config.ServiceName
	l.serviceFile = fmt.Sprintf("/etc/systemd/system/%s.service", config.ServiceName)

	// Check if service already exists
	if _, err := os.Stat(l.serviceFile); err == nil {
		return fmt.Errorf("service %s already installed", config.ServiceName)
	}

	// Create systemd service file content
	serviceContent := l.generateServiceFile(config)

	// Write service file
	if err := os.WriteFile(l.serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd daemon to recognize new service
	if err := l.runSystemctl("daemon-reload"); err != nil {
		// Clean up service file on failure
		os.Remove(l.serviceFile)
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

// generateServiceFile creates the systemd service file content
func (l *LinuxEmbeddedSystemIntegrator) generateServiceFile(config *platform.InstallConfig) string {
	var sb strings.Builder

	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=%s\n", config.Description))
	if config.DisplayName != "" {
		sb.WriteString(fmt.Sprintf("# Display Name: %s\n", config.DisplayName))
	}
	sb.WriteString("After=network.target\n")
	sb.WriteString("Wants=network-online.target\n")
	sb.WriteString("\n")

	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")
	sb.WriteString("Restart=on-failure\n")
	sb.WriteString("RestartSec=5s\n")

	// User
	if config.User != "" {
		sb.WriteString(fmt.Sprintf("User=%s\n", config.User))
	}

	// Working directory
	if config.WorkingDir != "" {
		sb.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", config.WorkingDir))
	}

	// Executable and arguments
	execStart := config.ExecutablePath
	if len(config.Arguments) > 0 {
		execStart += " " + strings.Join(config.Arguments, " ")
	}
	sb.WriteString(fmt.Sprintf("ExecStart=%s\n", execStart))

	// Capabilities for packet capture
	sb.WriteString("AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN\n")
	sb.WriteString("CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN\n")

	// Security hardening
	sb.WriteString("NoNewPrivileges=true\n")
	sb.WriteString("PrivateTmp=true\n")
	sb.WriteString("ProtectSystem=strict\n")
	sb.WriteString("ProtectHome=true\n")

	// Allow writing to specific directories
	if config.WorkingDir != "" {
		sb.WriteString(fmt.Sprintf("ReadWritePaths=%s\n", config.WorkingDir))
	}
	sb.WriteString("ReadWritePaths=/var/lib/heimdal\n")
	sb.WriteString("ReadWritePaths=/var/log/heimdal\n")

	sb.WriteString("\n")
	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")

	return sb.String()
}

// Uninstall removes the application from systemd
func (l *LinuxEmbeddedSystemIntegrator) Uninstall() error {
	if l.serviceName == "" {
		return fmt.Errorf("service not installed or service name not set")
	}

	// Stop the service if running
	_ = l.Stop() // Ignore error if service is not running

	// Disable the service
	_ = l.EnableAutoStart(false) // Ignore error if service is not enabled

	// Remove service file
	if l.serviceFile != "" {
		if err := os.Remove(l.serviceFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove service file: %w", err)
		}
	}

	// Reload systemd daemon
	if err := l.runSystemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

// Start begins the service
func (l *LinuxEmbeddedSystemIntegrator) Start() error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	if err := l.runSystemctl("start", l.serviceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// Stop halts the service
func (l *LinuxEmbeddedSystemIntegrator) Stop() error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	if err := l.runSystemctl("stop", l.serviceName); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

// Restart stops and starts the service
func (l *LinuxEmbeddedSystemIntegrator) Restart() error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	if err := l.runSystemctl("restart", l.serviceName); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	return nil
}

// GetStatus returns the current service status
func (l *LinuxEmbeddedSystemIntegrator) GetStatus() (*platform.ServiceStatus, error) {
	if l.serviceName == "" {
		return nil, fmt.Errorf("service name not set")
	}

	status := &platform.ServiceStatus{
		IsRunning:   false,
		IsInstalled: false,
		AutoStart:   false,
		PID:         0,
		Uptime:      0,
	}

	// Check if service file exists
	if l.serviceFile != "" {
		if _, err := os.Stat(l.serviceFile); err == nil {
			status.IsInstalled = true
		}
	}

	// Check if service is running
	cmd := exec.Command("systemctl", "is-active", l.serviceName)
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		status.IsRunning = true
	}

	// Check if service is enabled (auto-start)
	cmd = exec.Command("systemctl", "is-enabled", l.serviceName)
	output, err = cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "enabled" {
		status.AutoStart = true
	}

	// Get PID if running
	if status.IsRunning {
		cmd = exec.Command("systemctl", "show", l.serviceName, "--property=MainPID")
		output, err = cmd.Output()
		if err == nil {
			pidStr := strings.TrimSpace(string(output))
			pidStr = strings.TrimPrefix(pidStr, "MainPID=")
			if pid, err := strconv.Atoi(pidStr); err == nil {
				status.PID = pid
			}
		}

		// Get uptime if running
		cmd = exec.Command("systemctl", "show", l.serviceName, "--property=ActiveEnterTimestamp")
		output, err = cmd.Output()
		if err == nil {
			timestampStr := strings.TrimSpace(string(output))
			timestampStr = strings.TrimPrefix(timestampStr, "ActiveEnterTimestamp=")
			if timestampStr != "" && timestampStr != "0" {
				// Parse timestamp (format: "Mon 2006-01-02 15:04:05 MST")
				// For simplicity, we'll use systemctl show with a different property
				cmd = exec.Command("systemctl", "show", l.serviceName, "--property=ActiveEnterTimestampMonotonic")
				output, err = cmd.Output()
				if err == nil {
					microsecondsStr := strings.TrimSpace(string(output))
					microsecondsStr = strings.TrimPrefix(microsecondsStr, "ActiveEnterTimestampMonotonic=")
					if microseconds, err := strconv.ParseInt(microsecondsStr, 10, 64); err == nil && microseconds > 0 {
						// Get current monotonic time
						cmd = exec.Command("systemctl", "show", l.serviceName, "--property=ExecMainStartTimestampMonotonic")
						output, err = cmd.Output()
						if err == nil {
							currentStr := strings.TrimSpace(string(output))
							currentStr = strings.TrimPrefix(currentStr, "ExecMainStartTimestampMonotonic=")
							if current, err := strconv.ParseInt(currentStr, 10, 64); err == nil && current > 0 {
								uptimeMicroseconds := current - microseconds
								status.Uptime = time.Duration(uptimeMicroseconds) * time.Microsecond
							}
						}
					}
				}
			}
		}
	}

	return status, nil
}

// EnableAutoStart configures the service to start on boot
func (l *LinuxEmbeddedSystemIntegrator) EnableAutoStart(enabled bool) error {
	if l.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	var action string
	if enabled {
		action = "enable"
	} else {
		action = "disable"
	}

	if err := l.runSystemctl(action, l.serviceName); err != nil {
		return fmt.Errorf("failed to %s service: %w", action, err)
	}

	return nil
}

// runSystemctl executes a systemctl command
func (l *LinuxEmbeddedSystemIntegrator) runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w, output: %s", strings.Join(args, " "), err, string(output))
	}
	return nil
}

// SetServiceName sets the service name for an existing integrator instance
// This is useful when the integrator is created before installation
func (l *LinuxEmbeddedSystemIntegrator) SetServiceName(serviceName string) {
	l.serviceName = serviceName
	l.serviceFile = fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
}

// GetServiceFile returns the path to the systemd service file
func (l *LinuxEmbeddedSystemIntegrator) GetServiceFile() string {
	return l.serviceFile
}

// GetServiceName returns the service name
func (l *LinuxEmbeddedSystemIntegrator) GetServiceName() string {
	return l.serviceName
}

// ValidateSystemd checks if systemd is available on the system
func ValidateSystemd() error {
	// Check if systemctl command exists
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl not found, systemd may not be available: %w", err)
	}

	// Check if systemd is running
	cmd := exec.Command("systemctl", "is-system-running")
	if err := cmd.Run(); err != nil {
		// Note: is-system-running may return non-zero even when systemd is working
		// (e.g., in degraded state), so we just check if the command exists
	}

	// Check if /etc/systemd/system directory exists
	if _, err := os.Stat("/etc/systemd/system"); err != nil {
		return fmt.Errorf("systemd system directory not found: %w", err)
	}

	return nil
}

// GetServiceFilePath returns the expected path for a service file
func GetServiceFilePath(serviceName string) string {
	return filepath.Join("/etc/systemd/system", fmt.Sprintf("%s.service", serviceName))
}
