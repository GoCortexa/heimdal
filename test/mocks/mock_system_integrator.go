package mocks

import (
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// MockSystemIntegrator is a mock implementation for testing system integration
type MockSystemIntegrator struct {
	mu              sync.Mutex
	isInstalled     bool
	isRunning       bool
	autoStart       bool
	pid             int
	startTime       time.Time
	installErr      error
	uninstallErr    error
	startErr        error
	stopErr         error
	restartErr      error
	getStatusErr    error
	autoStartErr    error
	installConfig   *platform.InstallConfig
	installCalled   bool
	uninstallCalled bool
	startCalled     bool
	stopCalled      bool
	restartCalled   bool
}

// NewMockSystemIntegrator creates a new mock system integrator
func NewMockSystemIntegrator() *MockSystemIntegrator {
	return &MockSystemIntegrator{
		isInstalled: false,
		isRunning:   false,
		autoStart:   false,
		pid:         0,
	}
}

// Install registers the application with the mock OS
func (m *MockSystemIntegrator) Install(config *platform.InstallConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.installCalled = true

	if m.installErr != nil {
		return m.installErr
	}

	m.installConfig = config
	m.isInstalled = true
	return nil
}

// Uninstall removes the application from mock OS registration
func (m *MockSystemIntegrator) Uninstall() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.uninstallCalled = true

	if m.uninstallErr != nil {
		return m.uninstallErr
	}

	m.isInstalled = false
	m.isRunning = false
	m.autoStart = false
	return nil
}

// Start begins the mock service
func (m *MockSystemIntegrator) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startCalled = true

	if m.startErr != nil {
		return m.startErr
	}

	if !m.isInstalled {
		return &MockError{Message: "service not installed"}
	}

	m.isRunning = true
	m.pid = 12345
	m.startTime = time.Now()
	return nil
}

// Stop halts the mock service
func (m *MockSystemIntegrator) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopCalled = true

	if m.stopErr != nil {
		return m.stopErr
	}

	m.isRunning = false
	m.pid = 0
	return nil
}

// Restart stops and starts the mock service
func (m *MockSystemIntegrator) Restart() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.restartCalled = true

	if m.restartErr != nil {
		return m.restartErr
	}

	if !m.isInstalled {
		return &MockError{Message: "service not installed"}
	}

	m.isRunning = true
	m.pid = 12345
	m.startTime = time.Now()
	return nil
}

// GetStatus returns the current mock service status
func (m *MockSystemIntegrator) GetStatus() (*platform.ServiceStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getStatusErr != nil {
		return nil, m.getStatusErr
	}

	var uptime time.Duration
	if m.isRunning {
		uptime = time.Since(m.startTime)
	}

	return &platform.ServiceStatus{
		IsRunning:   m.isRunning,
		IsInstalled: m.isInstalled,
		AutoStart:   m.autoStart,
		PID:         m.pid,
		Uptime:      uptime,
	}, nil
}

// EnableAutoStart configures the mock service to start on boot
func (m *MockSystemIntegrator) EnableAutoStart(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.autoStartErr != nil {
		return m.autoStartErr
	}

	m.autoStart = enabled
	return nil
}

// SetInstallError configures the mock to return an error on Install
func (m *MockSystemIntegrator) SetInstallError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installErr = err
}

// SetUninstallError configures the mock to return an error on Uninstall
func (m *MockSystemIntegrator) SetUninstallError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uninstallErr = err
}

// SetStartError configures the mock to return an error on Start
func (m *MockSystemIntegrator) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startErr = err
}

// SetStopError configures the mock to return an error on Stop
func (m *MockSystemIntegrator) SetStopError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopErr = err
}

// SetRestartError configures the mock to return an error on Restart
func (m *MockSystemIntegrator) SetRestartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartErr = err
}

// SetGetStatusError configures the mock to return an error on GetStatus
func (m *MockSystemIntegrator) SetGetStatusError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getStatusErr = err
}

// SetAutoStartError configures the mock to return an error on EnableAutoStart
func (m *MockSystemIntegrator) SetAutoStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoStartErr = err
}

// WasInstallCalled returns whether Install was called
func (m *MockSystemIntegrator) WasInstallCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.installCalled
}

// WasUninstallCalled returns whether Uninstall was called
func (m *MockSystemIntegrator) WasUninstallCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.uninstallCalled
}

// WasStartCalled returns whether Start was called
func (m *MockSystemIntegrator) WasStartCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalled
}

// WasStopCalled returns whether Stop was called
func (m *MockSystemIntegrator) WasStopCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopCalled
}

// WasRestartCalled returns whether Restart was called
func (m *MockSystemIntegrator) WasRestartCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartCalled
}

// GetInstallConfig returns the config passed to Install
func (m *MockSystemIntegrator) GetInstallConfig() *platform.InstallConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.installConfig
}

// Reset resets the mock to its initial state
func (m *MockSystemIntegrator) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isInstalled = false
	m.isRunning = false
	m.autoStart = false
	m.pid = 0
	m.installErr = nil
	m.uninstallErr = nil
	m.startErr = nil
	m.stopErr = nil
	m.restartErr = nil
	m.getStatusErr = nil
	m.autoStartErr = nil
	m.installConfig = nil
	m.installCalled = false
	m.uninstallCalled = false
	m.startCalled = false
	m.stopCalled = false
	m.restartCalled = false
}

// MockError is a simple error type for mock errors
type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}
