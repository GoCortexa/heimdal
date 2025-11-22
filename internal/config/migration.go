// Package config provides configuration migration utilities for backward compatibility.
//
// This module handles migration from the legacy hardware-focused configuration format
// to the new monorepo configuration format. It detects old configuration files,
// converts them to the new format, and logs migration warnings.
//
// Legacy Format (Hardware):
//   - Used by the original Heimdal Hardware product
//   - Located at /etc/heimdal/config.json
//   - Contains: database, network, discovery, interceptor, profiler, api, cloud, logging
//
// New Format (Monorepo):
//   - Used by both Hardware and Desktop products
//   - Hardware: /etc/heimdal/config.json
//   - Desktop: Platform-specific locations
//   - Contains: All legacy fields plus desktop-specific fields (detection, visualizer, system_tray, feature_gate)
//
// Migration Process:
//   1. Detect if configuration is in legacy format (missing new fields)
//   2. Convert legacy fields to new format
//   3. Add default values for new fields
//   4. Log migration warnings
//   5. Save migrated configuration
//
// The migration is automatic and transparent to the user. The original configuration
// file is backed up before migration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LegacyConfig represents the old hardware-focused configuration format
type LegacyConfig struct {
	Database    DatabaseConfig    `json:"database"`
	Network     NetworkConfig     `json:"network"`
	Discovery   LegacyDiscoveryConfig   `json:"discovery"`
	Interceptor LegacyInterceptorConfig `json:"interceptor"`
	Profiler    ProfilerConfig    `json:"profiler"`
	API         APIConfig         `json:"api"`
	Cloud       CloudConfig       `json:"cloud"`
	Logging     LoggingConfig     `json:"logging"`
}

// LegacyDiscoveryConfig represents the old discovery configuration
type LegacyDiscoveryConfig struct {
	ARPScanInterval int  `json:"arp_scan_interval_seconds"`
	MDNSEnabled     bool `json:"mdns_enabled"`
	InactiveTimeout int  `json:"inactive_timeout_minutes"`
}

// LegacyInterceptorConfig represents the old interceptor configuration
type LegacyInterceptorConfig struct {
	Enabled       bool     `json:"enabled"`
	SpoofInterval int      `json:"spoof_interval_seconds"`
	TargetMACs    []string `json:"target_macs"`
}

// Note: ProfilerConfig and APIConfig are already defined in config.go
// and are unchanged between versions, so we don't redefine them here

// MigrationResult contains information about the migration process
type MigrationResult struct {
	WasMigrated     bool
	BackupPath      string
	MigrationErrors []string
	Warnings        []string
}

// IsLegacyFormat detects if a configuration file is in the legacy format
func IsLegacyFormat(data []byte) (bool, error) {
	// Try to unmarshal as a generic map to check for presence of new fields
	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		return false, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Check for new fields that don't exist in legacy format
	// If any of these are missing, it's likely a legacy config
	newFields := []string{"detection", "visualizer", "system_tray", "feature_gate"}
	missingCount := 0

	for _, field := range newFields {
		if _, exists := configMap[field]; !exists {
			missingCount++
		}
	}

	// If all new fields are missing, it's definitely legacy format
	// If some are missing, it might be a partial config or legacy
	return missingCount >= len(newFields)-1, nil
}

// MigrateLegacyConfig converts a legacy configuration to the new format
func MigrateLegacyConfig(legacyPath string) (*Config, *MigrationResult, error) {
	result := &MigrationResult{
		WasMigrated:     false,
		MigrationErrors: make([]string, 0),
		Warnings:        make([]string, 0),
	}

	// Read the legacy configuration file
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return nil, result, fmt.Errorf("failed to read legacy config: %w", err)
	}

	// Check if it's actually a legacy format
	isLegacy, err := IsLegacyFormat(data)
	if err != nil {
		return nil, result, fmt.Errorf("failed to detect config format: %w", err)
	}

	if !isLegacy {
		// Not a legacy config, return nil to indicate no migration needed
		return nil, result, nil
	}

	// Parse as legacy config
	var legacy LegacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, result, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	// Create backup of original file
	backupPath := legacyPath + ".backup." + time.Now().Format("20060102-150405")
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create backup: %v", err))
	} else {
		result.BackupPath = backupPath
	}

	// Convert to new format
	newConfig := &Config{
		Database: legacy.Database,
		Network:  legacy.Network,
		Discovery: DiscoveryConfig{
			ARPScanInterval: legacy.Discovery.ARPScanInterval,
			MDNSEnabled:     legacy.Discovery.MDNSEnabled,
			InactiveTimeout: legacy.Discovery.InactiveTimeout,
		},
		Interceptor: InterceptorConfig{
			Enabled:       legacy.Interceptor.Enabled,
			SpoofInterval: legacy.Interceptor.SpoofInterval,
			TargetMACs:    legacy.Interceptor.TargetMACs,
		},
		Profiler: legacy.Profiler,
		API:      legacy.API,
		Cloud:    legacy.Cloud,
		Logging:  legacy.Logging,
	}

	// Validate the migrated configuration
	if err := newConfig.Validate(); err != nil {
		result.MigrationErrors = append(result.MigrationErrors, fmt.Sprintf("Validation error: %v", err))
		return nil, result, fmt.Errorf("migrated config validation failed: %w", err)
	}

	result.WasMigrated = true
	result.Warnings = append(result.Warnings, "Configuration migrated from legacy format")
	result.Warnings = append(result.Warnings, fmt.Sprintf("Original configuration backed up to: %s", backupPath))

	return newConfig, result, nil
}

// MigrateLegacyConfigInPlace migrates a legacy configuration file in place
func MigrateLegacyConfigInPlace(configPath string) (*MigrationResult, error) {
	newConfig, result, err := MigrateLegacyConfig(configPath)
	if err != nil {
		return result, err
	}

	// If no migration was needed, return early
	if newConfig == nil {
		return result, nil
	}

	// Write the migrated configuration back to the original path
	data, err := json.MarshalIndent(newConfig, "", "  ")
	if err != nil {
		return result, fmt.Errorf("failed to marshal migrated config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return result, fmt.Errorf("failed to write migrated config: %w", err)
	}

	return result, nil
}

// LoadConfigWithMigration loads a configuration file and automatically migrates if needed
func LoadConfigWithMigration(path string) (*Config, *MigrationResult, error) {
	result := &MigrationResult{
		WasMigrated:     false,
		MigrationErrors: make([]string, 0),
		Warnings:        make([]string, 0),
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, result, fmt.Errorf("configuration file not found: %s", path)
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, result, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Check if it's legacy format
	isLegacy, err := IsLegacyFormat(data)
	if err != nil {
		return nil, result, fmt.Errorf("failed to detect config format: %w", err)
	}

	if isLegacy {
		// Migrate the configuration
		newConfig, migResult, err := MigrateLegacyConfig(path)
		if err != nil {
			return nil, migResult, err
		}

		// Write the migrated config back
		migratedData, err := json.MarshalIndent(newConfig, "", "  ")
		if err != nil {
			return nil, migResult, fmt.Errorf("failed to marshal migrated config: %w", err)
		}

		if err := os.WriteFile(path, migratedData, 0644); err != nil {
			return nil, migResult, fmt.Errorf("failed to write migrated config: %w", err)
		}

		// Log migration warnings
		for _, warning := range migResult.Warnings {
			fmt.Fprintf(os.Stderr, "MIGRATION WARNING: %s\n", warning)
		}

		return newConfig, migResult, nil
	}

	// Not legacy format, load normally
	config, err := LoadConfig(path)
	if err != nil {
		return nil, result, err
	}

	return config, result, nil
}

// ConvertLegacyToDesktopConfig converts a legacy hardware config to desktop config format
// This is useful when users want to migrate from hardware to desktop deployment
func ConvertLegacyToDesktopConfig(legacyPath string, desktopPath string) error {
	// Read and migrate legacy config
	newConfig, result, err := MigrateLegacyConfig(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to migrate legacy config: %w", err)
	}

	// If no migration was needed (not legacy format), return error
	if newConfig == nil {
		return fmt.Errorf("source configuration is not in legacy format")
	}

	// Get default desktop config to fill in desktop-specific fields
	desktopConfig, err := DefaultDesktopConfig()
	if err != nil {
		return fmt.Errorf("failed to create default desktop config: %w", err)
	}

	// Copy over the migrated hardware settings
	desktopConfig.Database = newConfig.Database
	desktopConfig.Network = newConfig.Network
	// Map hardware discovery config to desktop format
	desktopConfig.Discovery = DesktopDiscoveryConfig{
		ScanInterval:    newConfig.Discovery.ARPScanInterval,
		MDNSEnabled:     newConfig.Discovery.MDNSEnabled,
		InactiveTimeout: newConfig.Discovery.InactiveTimeout,
	}
	// Map hardware interceptor config to desktop format
	desktopConfig.Interceptor = DesktopInterceptorConfig{
		Enabled:       newConfig.Interceptor.Enabled,
		TargetDevices: newConfig.Interceptor.TargetMACs,
	}
	desktopConfig.Cloud = newConfig.Cloud
	desktopConfig.Logging = newConfig.Logging

	// Adjust paths for desktop platform
	if dbPath, err := GetDefaultDatabasePath(); err == nil {
		desktopConfig.Database.Path = dbPath
	}
	if logPath, err := GetDefaultLogPath(); err == nil {
		desktopConfig.Logging.File = logPath
	}

	// Ensure desktop config directory exists
	desktopDir := filepath.Dir(desktopPath)
	if err := os.MkdirAll(desktopDir, 0755); err != nil {
		return fmt.Errorf("failed to create desktop config directory: %w", err)
	}

	// Write desktop config
	data, err := json.MarshalIndent(desktopConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal desktop config: %w", err)
	}

	if err := os.WriteFile(desktopPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write desktop config: %w", err)
	}

	// Log migration info
	fmt.Fprintf(os.Stderr, "Successfully converted legacy config to desktop format\n")
	fmt.Fprintf(os.Stderr, "Source: %s\n", legacyPath)
	fmt.Fprintf(os.Stderr, "Destination: %s\n", desktopPath)
	if result.BackupPath != "" {
		fmt.Fprintf(os.Stderr, "Backup: %s\n", result.BackupPath)
	}

	return nil
}

// DefaultDesktopConfig returns a desktop configuration with default values
// This is a helper function for migration purposes
func DefaultDesktopConfig() (*DesktopConfig, error) {
	dbPath, err := GetDefaultDatabasePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get default database path: %w", err)
	}

	logPath, err := GetDefaultLogPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get default log path: %w", err)
	}

	return &DesktopConfig{
		Database: DatabaseConfig{
			Path:       dbPath,
			GCInterval: 5,
		},
		Network: NetworkConfig{
			Interface:  "",
			AutoDetect: true,
		},
		Discovery: DesktopDiscoveryConfig{
			ScanInterval:    60,
			MDNSEnabled:     true,
			InactiveTimeout: 5,
		},
		Interceptor: DesktopInterceptorConfig{
			Enabled:       false,
			TargetDevices: []string{},
		},
		Detection: DetectionConfig{
			Enabled:     true,
			Sensitivity: 0.7,
		},
		Visualizer: VisualizerConfig{
			Enabled: true,
			Port:    8080,
		},
		SystemTray: SystemTrayConfig{
			Enabled:   true,
			AutoStart: false,
		},
		FeatureGate: FeatureGateConfig{
			Tier:       "free",
			LicenseKey: "",
		},
		Cloud: CloudConfig{
			Enabled:  false,
			Provider: "aws",
			AWS: AWSConfig{
				Endpoint: "",
				ClientID: "",
				CertPath: "",
				KeyPath:  "",
			},
			GCP: GCPConfig{
				ProjectID: "",
				TopicID:   "",
			},
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  logPath,
		},
	}, nil
}

// GetDefaultDatabasePath returns the platform-specific default database path
func GetDefaultDatabasePath() (string, error) {
	// This is imported from internal/desktop/config
	// For hardware, use the standard path
	return "/var/lib/heimdal/db", nil
}

// GetDefaultLogPath returns the platform-specific default log path
func GetDefaultLogPath() (string, error) {
	// This is imported from internal/desktop/config
	// For hardware, use the standard path
	return "/var/log/heimdal/heimdal.log", nil
}

// Desktop-specific config types for migration
// These mirror the types in internal/desktop/config but are defined here
// to avoid circular dependencies

// DesktopDiscoveryConfig contains device discovery settings for desktop
type DesktopDiscoveryConfig struct {
	ScanInterval    int  `json:"scan_interval_seconds"`
	MDNSEnabled     bool `json:"mdns_enabled"`
	InactiveTimeout int  `json:"inactive_timeout_minutes"`
}

// DesktopInterceptorConfig contains traffic interception settings for desktop
type DesktopInterceptorConfig struct {
	Enabled       bool     `json:"enabled"`
	TargetDevices []string `json:"target_devices"`
}

// DesktopConfig is the desktop configuration format
type DesktopConfig struct {
	Database    DatabaseConfig           `json:"database"`
	Network     NetworkConfig            `json:"network"`
	Discovery   DesktopDiscoveryConfig   `json:"discovery"`
	Interceptor DesktopInterceptorConfig `json:"interceptor"`
	Detection   DetectionConfig          `json:"detection"`
	Visualizer  VisualizerConfig         `json:"visualizer"`
	SystemTray  SystemTrayConfig         `json:"system_tray"`
	FeatureGate FeatureGateConfig        `json:"feature_gate"`
	Cloud       CloudConfig              `json:"cloud"`
	Logging     LoggingConfig            `json:"logging"`
}

// DetectionConfig contains anomaly detection settings
type DetectionConfig struct {
	Enabled     bool    `json:"enabled"`
	Sensitivity float64 `json:"sensitivity"`
}

// VisualizerConfig contains local dashboard settings
type VisualizerConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// SystemTrayConfig contains system tray integration settings
type SystemTrayConfig struct {
	Enabled   bool `json:"enabled"`
	AutoStart bool `json:"auto_start"`
}

// FeatureGateConfig contains tier and licensing settings
type FeatureGateConfig struct {
	Tier       string `json:"tier"`
	LicenseKey string `json:"license_key"`
}
