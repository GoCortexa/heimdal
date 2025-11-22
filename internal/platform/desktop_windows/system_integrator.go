// +build windows

package desktop_windows

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// WindowsSystemIntegrator implements SystemIntegrator for Windows using Windows Service API
type WindowsSystemIntegrator struct {
	serviceName string
}

// NewWindowsSystemIntegrator creates a new Windows system integrator
func NewWindowsSystemIntegrator() *WindowsSystemIntegrator {
	return &WindowsSystemIntegrator{}
}

// Install registers the application as a Windows Service
func (w *WindowsSystemIntegrator) Install(config *platform.InstallConfig) error {
	if config == nil {
		return fmt.Errorf("install config cannot be nil")
	}

	w.serviceName = config.ServiceName

	// Connect to the service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Check if service already exists
	s, err := m.OpenService(config.ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", config.ServiceName)
	}

	// Prepare service configuration
	svcConfig := mgr.Config{
		DisplayName:      config.DisplayName,
		Description:      config.Description,
		StartType:        mgr.StartManual,
		ServiceStartName: config.User,
	}

	// Build command line with arguments
	exePath := config.ExecutablePath
	if len(config.Arguments) > 0 {
		// Note: Windows services receive arguments differently
		// We'll store them in the service config
	}

	// Create the service
	s, err = m.CreateService(
		config.ServiceName,
		exePath,
		svcConfig,
		config.Arguments...,
	)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Set recovery options (restart on failure)
	err = s.SetRecoveryActions(
		[]mgr.RecoveryAction{
			{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
			{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
			{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
		},
		300, // Reset failure count after 5 minutes
	)
	if err != nil {
		// Non-fatal error, log but continue
		fmt.Printf("Warning: failed to set recovery actions: %v\n", err)
	}

	return nil
}

// Uninstall removes the Windows Service
func (w *WindowsSystemIntegrator) Uninstall() error {
	if w.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Connect to the service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open the service
	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Stop the service if it's running
	status, err := s.Query()
	if err == nil && status.State == svc.Running {
		_, err = s.Control(svc.Stop)
		if err != nil {
			return fmt.Errorf("failed to stop service before uninstall: %w", err)
		}

		// Wait for service to stop
		timeout := time.Now().Add(30 * time.Second)
		for time.Now().Before(timeout) {
			status, err = s.Query()
			if err != nil || status.State == svc.Stopped {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Delete the service
	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// Start begins the Windows Service
func (w *WindowsSystemIntegrator) Start() error {
	if w.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Connect to the service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open the service
	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Start the service
	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// Stop halts the Windows Service
func (w *WindowsSystemIntegrator) Stop() error {
	if w.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Connect to the service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open the service
	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Send stop control
	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait for service to stop
	timeout := time.Now().Add(30 * time.Second)
	for time.Now().Before(timeout) {
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status: %w", err)
		}
		if status.State == svc.Stopped {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("service did not stop within timeout")
}

// Restart stops and starts the Windows Service
func (w *WindowsSystemIntegrator) Restart() error {
	if err := w.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait a moment before starting
	time.Sleep(1 * time.Second)

	if err := w.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// GetStatus returns the current service status
func (w *WindowsSystemIntegrator) GetStatus() (*platform.ServiceStatus, error) {
	if w.serviceName == "" {
		return &platform.ServiceStatus{
			IsRunning:   false,
			IsInstalled: false,
			AutoStart:   false,
			PID:         0,
			Uptime:      0,
		}, nil
	}

	// Connect to the service manager
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open the service
	s, err := m.OpenService(w.serviceName)
	if err != nil {
		// Service not installed
		return &platform.ServiceStatus{
			IsRunning:   false,
			IsInstalled: false,
			AutoStart:   false,
			PID:         0,
			Uptime:      0,
		}, nil
	}
	defer s.Close()

	// Query service status
	status, err := s.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query service status: %w", err)
	}

	// Get service configuration
	config, err := s.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get service config: %w", err)
	}

	// Get PID using sc query
	pid := w.getServicePID()

	// Calculate uptime (approximate)
	var uptime time.Duration
	if status.State == svc.Running {
		// We can't get exact start time from Windows API easily
		// This is an approximation
		uptime = 0 // Would need to track start time separately
	}

	return &platform.ServiceStatus{
		IsRunning:   status.State == svc.Running,
		IsInstalled: true,
		AutoStart:   config.StartType == mgr.StartAutomatic,
		PID:         pid,
		Uptime:      uptime,
	}, nil
}

// getServicePID attempts to get the PID of the service
func (w *WindowsSystemIntegrator) getServicePID() int {
	cmd := exec.Command("sc", "queryex", w.serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	// Parse output for PID
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "PID") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				pidStr := strings.TrimSpace(parts[1])
				pid, err := strconv.Atoi(pidStr)
				if err == nil {
					return pid
				}
			}
		}
	}

	return 0
}

// EnableAutoStart configures the service to start on boot
func (w *WindowsSystemIntegrator) EnableAutoStart(enabled bool) error {
	if w.serviceName == "" {
		return fmt.Errorf("service name not set")
	}

	// Connect to the service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open the service
	s, err := m.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Get current configuration
	config, err := s.Config()
	if err != nil {
		return fmt.Errorf("failed to get service config: %w", err)
	}

	// Update start type
	if enabled {
		config.StartType = mgr.StartAutomatic
	} else {
		config.StartType = mgr.StartManual
	}

	// Update the service configuration
	err = s.UpdateConfig(config)
	if err != nil {
		return fmt.Errorf("failed to update service config: %w", err)
	}

	return nil
}

// IsAdministrator checks if the current process has administrator privileges
func IsAdministrator() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}
