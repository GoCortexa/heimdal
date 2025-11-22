package systray

import (
	"encoding/json"
	"fmt"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// AutoStartManager manages auto-start configuration
type AutoStartManager struct {
	integrator platform.SystemIntegrator
	storage    platform.StorageProvider
	configKey  string
}

// NewAutoStartManager creates a new auto-start manager
func NewAutoStartManager(integrator platform.SystemIntegrator, storage platform.StorageProvider) *AutoStartManager {
	return &AutoStartManager{
		integrator: integrator,
		storage:    storage,
		configKey:  "autostart_enabled",
	}
}

// IsEnabled returns whether auto-start is currently enabled
func (asm *AutoStartManager) IsEnabled() (bool, error) {
	// First check the system integrator status
	status, err := asm.integrator.GetStatus()
	if err != nil {
		return false, fmt.Errorf("failed to get service status: %w", err)
	}

	return status.AutoStart, nil
}

// Enable enables auto-start
func (asm *AutoStartManager) Enable() error {
	// Enable auto-start through the system integrator
	if err := asm.integrator.EnableAutoStart(true); err != nil {
		return fmt.Errorf("failed to enable auto-start: %w", err)
	}

	// Store the preference in configuration
	if err := asm.savePreference(true); err != nil {
		// Log warning but don't fail - the system integrator already enabled it
		fmt.Printf("Warning: failed to save auto-start preference: %v\n", err)
	}

	return nil
}

// Disable disables auto-start
func (asm *AutoStartManager) Disable() error {
	// Disable auto-start through the system integrator
	if err := asm.integrator.EnableAutoStart(false); err != nil {
		return fmt.Errorf("failed to disable auto-start: %w", err)
	}

	// Store the preference in configuration
	if err := asm.savePreference(false); err != nil {
		// Log warning but don't fail - the system integrator already disabled it
		fmt.Printf("Warning: failed to save auto-start preference: %v\n", err)
	}

	return nil
}

// Toggle toggles the auto-start setting
func (asm *AutoStartManager) Toggle() (bool, error) {
	enabled, err := asm.IsEnabled()
	if err != nil {
		return false, fmt.Errorf("failed to check auto-start status: %w", err)
	}

	if enabled {
		if err := asm.Disable(); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := asm.Enable(); err != nil {
		return false, err
	}
	return true, nil
}

// GetPreference returns the stored auto-start preference
func (asm *AutoStartManager) GetPreference() (bool, error) {
	if asm.storage == nil {
		// If no storage is available, check the system integrator
		return asm.IsEnabled()
	}

	data, err := asm.storage.Get(asm.configKey)
	if err != nil {
		// If key doesn't exist, default to false
		return false, nil
	}

	var enabled bool
	if err := json.Unmarshal(data, &enabled); err != nil {
		return false, fmt.Errorf("failed to unmarshal auto-start preference: %w", err)
	}

	return enabled, nil
}

// savePreference saves the auto-start preference to storage
func (asm *AutoStartManager) savePreference(enabled bool) error {
	if asm.storage == nil {
		return nil // No storage available, skip
	}

	data, err := json.Marshal(enabled)
	if err != nil {
		return fmt.Errorf("failed to marshal auto-start preference: %w", err)
	}

	if err := asm.storage.Set(asm.configKey, data); err != nil {
		return fmt.Errorf("failed to save auto-start preference: %w", err)
	}

	return nil
}

// SyncWithSystem synchronizes the stored preference with the actual system state
func (asm *AutoStartManager) SyncWithSystem() error {
	// Get the actual system state
	systemEnabled, err := asm.IsEnabled()
	if err != nil {
		return fmt.Errorf("failed to get system auto-start state: %w", err)
	}

	// Get the stored preference
	storedEnabled, err := asm.GetPreference()
	if err != nil {
		return fmt.Errorf("failed to get stored auto-start preference: %w", err)
	}

	// If they don't match, update the stored preference to match the system
	if systemEnabled != storedEnabled {
		if err := asm.savePreference(systemEnabled); err != nil {
			return fmt.Errorf("failed to sync auto-start preference: %w", err)
		}
	}

	return nil
}

// AutoStartConfig represents the auto-start configuration
type AutoStartConfig struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"` // Optional: reason for the setting
}

// LoadConfig loads the auto-start configuration
func (asm *AutoStartManager) LoadConfig() (*AutoStartConfig, error) {
	enabled, err := asm.GetPreference()
	if err != nil {
		return nil, err
	}

	return &AutoStartConfig{
		Enabled: enabled,
	}, nil
}

// SaveConfig saves the auto-start configuration
func (asm *AutoStartManager) SaveConfig(config *AutoStartConfig) error {
	if config.Enabled {
		return asm.Enable()
	}
	return asm.Disable()
}
