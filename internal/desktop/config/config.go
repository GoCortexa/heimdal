// Package config provides configuration management for the Heimdal Desktop agent.
//
// The configuration is loaded from platform-specific locations:
//   - Windows: %APPDATA%\Heimdal\config.json
//   - macOS: ~/Library/Application Support/Heimdal/config.json
//   - Linux: ~/.config/heimdal/config.json
//
// Configuration Structure:
//   - Database: Storage path and settings
//   - Network: Interface selection and auto-detection
//   - Discovery: Device discovery settings
//   - Interceptor: Traffic interception settings (Pro tier)
//   - Detection: Anomaly detection sensitivity
//   - Visualizer: Local dashboard settings
//   - SystemTray: System tray integration settings
//   - FeatureGate: Tier and license configuration
//   - Cloud: Cloud connectivity settings (optional)
//   - Logging: Log level and file path
//
// The LoadConfig function reads and parses the configuration file from the
// platform-specific location, validates required fields, and returns a
// DesktopConfig struct. Default values are provided for optional fields.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// DesktopConfig represents the complete desktop agent configuration
type DesktopConfig struct {
	Database    DatabaseConfig    `json:"database"`
	Network     NetworkConfig     `json:"network"`
	Discovery   DiscoveryConfig   `json:"discovery"`
	Interceptor InterceptorConfig `json:"interceptor"`
	Detection   DetectionConfig   `json:"detection"`
	Visualizer  VisualizerConfig  `json:"visualizer"`
	SystemTray  SystemTrayConfig  `json:"system_tray"`
	FeatureGate FeatureGateConfig `json:"feature_gate"`
	Cloud       CloudConfig       `json:"cloud"`
	Logging     LoggingConfig     `json:"logging"`

	// Internal fields
	configPath string
	mu         sync.RWMutex
	watchers   []ConfigWatcher
}

// DatabaseConfig contains database-related settings
type DatabaseConfig struct {
	Path       string `json:"path"`
	GCInterval int    `json:"gc_interval_minutes"`
}

// NetworkConfig contains network interface settings
type NetworkConfig struct {
	Interface  string `json:"interface"`
	AutoDetect bool   `json:"auto_detect"`
}

// DiscoveryConfig contains device discovery settings
type DiscoveryConfig struct {
	ScanInterval    int  `json:"scan_interval_seconds"`
	MDNSEnabled     bool `json:"mdns_enabled"`
	InactiveTimeout int  `json:"inactive_timeout_minutes"`
}

// InterceptorConfig contains traffic interception settings
type InterceptorConfig struct {
	Enabled       bool     `json:"enabled"`
	TargetDevices []string `json:"target_devices"` // MAC addresses to intercept
}

// DetectionConfig contains anomaly detection settings
type DetectionConfig struct {
	Enabled     bool    `json:"enabled"`
	Sensitivity float64 `json:"sensitivity"` // 0.0 to 1.0
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
	Tier       string `json:"tier"`        // "free", "pro", "enterprise"
	LicenseKey string `json:"license_key"` // Optional license key
}

// CloudConfig contains cloud connectivity settings
type CloudConfig struct {
	Enabled  bool      `json:"enabled"`
	Provider string    `json:"provider"` // "aws" or "gcp"
	AWS      AWSConfig `json:"aws"`
	GCP      GCPConfig `json:"gcp"`
	
	// Privacy controls
	SendDeviceInfo    bool `json:"send_device_info"`     // Send device discovery data
	SendProfiles      bool `json:"send_profiles"`        // Send behavioral profiles
	SendAnomalies     bool `json:"send_anomalies"`       // Send detected anomalies
	AnonymizeData     bool `json:"anonymize_data"`       // Hash sensitive fields
	SendDiagnostics   bool `json:"send_diagnostics"`     // Send diagnostic telemetry
}

// AWSConfig contains AWS IoT Core settings
type AWSConfig struct {
	Endpoint string `json:"endpoint"`
	ClientID string `json:"client_id"`
	CertPath string `json:"cert_path"`
	KeyPath  string `json:"key_path"`
}

// GCPConfig contains Google Cloud IoT Core settings
type GCPConfig struct {
	ProjectID string `json:"project_id"`
	TopicID   string `json:"topic_id"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level string `json:"level"` // "debug", "info", "warn", "error"
	File  string `json:"file"`
}

// ConfigWatcher is called when configuration changes
type ConfigWatcher func(*DesktopConfig) error

// GetDefaultConfigPath returns the platform-specific default configuration path
func GetDefaultConfigPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, "Heimdal")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(home, "Library", "Application Support", "Heimdal")
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config", "heimdal")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// GetDefaultDatabasePath returns the platform-specific default database path
func GetDefaultDatabasePath() (string, error) {
	var dbDir string

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		dbDir = filepath.Join(appData, "Heimdal", "db")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dbDir = filepath.Join(home, "Library", "Application Support", "Heimdal", "db")
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dbDir = filepath.Join(home, ".local", "share", "heimdal", "db")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return dbDir, nil
}

// GetDefaultLogPath returns the platform-specific default log path
func GetDefaultLogPath() (string, error) {
	var logDir string

	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
		}
		logDir = filepath.Join(localAppData, "Heimdal", "logs")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		logDir = filepath.Join(home, "Library", "Logs", "Heimdal")
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		logDir = filepath.Join(home, ".local", "share", "heimdal", "logs")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	return filepath.Join(logDir, "heimdal.log"), nil
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() (*DesktopConfig, error) {
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
		Discovery: DiscoveryConfig{
			ScanInterval:    60,
			MDNSEnabled:     true,
			InactiveTimeout: 5,
		},
		Interceptor: InterceptorConfig{
			Enabled:       false, // Disabled by default (requires Pro tier)
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
			// Privacy defaults: send basic telemetry for free tier
			SendDeviceInfo:  true,  // Send device discovery (helps build device database)
			SendProfiles:    true,  // Send behavioral profiles (helps ML models)
			SendAnomalies:   false, // Don't send anomalies by default (privacy)
			AnonymizeData:   true,  // Anonymize by default
			SendDiagnostics: false, // Opt-in for diagnostics
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  logPath,
		},
		watchers: make([]ConfigWatcher, 0),
	}, nil
}

// LoadConfig loads configuration from the default platform-specific path
func LoadConfig() (*DesktopConfig, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get default config path: %w", err)
	}

	return LoadConfigFromPath(configPath)
}

// LoadConfigFromPath loads configuration from a specific path
func LoadConfigFromPath(path string) (*DesktopConfig, error) {
	// Start with default configuration
	config, err := DefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	config.configPath = path

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Config file doesn't exist, create it with defaults
		if err := config.Save(); err != nil {
			return nil, fmt.Errorf("failed to create default config file: %w", err)
		}
		return config, nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Save writes the configuration to disk
func (c *DesktopConfig) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.configPath == "" {
		return fmt.Errorf("config path not set")
	}

	// Ensure directory exists
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// Reload reloads the configuration from disk and notifies watchers
func (c *DesktopConfig) Reload() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configPath == "" {
		return fmt.Errorf("config path not set")
	}

	// Read file
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Create a temporary config to parse into
	tempConfig := &DesktopConfig{}
	if err := json.Unmarshal(data, tempConfig); err != nil {
		return fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Validate new configuration
	if err := tempConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update current config with new values
	c.Database = tempConfig.Database
	c.Network = tempConfig.Network
	c.Discovery = tempConfig.Discovery
	c.Interceptor = tempConfig.Interceptor
	c.Detection = tempConfig.Detection
	c.Visualizer = tempConfig.Visualizer
	c.SystemTray = tempConfig.SystemTray
	c.FeatureGate = tempConfig.FeatureGate
	c.Cloud = tempConfig.Cloud
	c.Logging = tempConfig.Logging

	// Notify watchers
	for _, watcher := range c.watchers {
		if err := watcher(c); err != nil {
			return fmt.Errorf("config watcher error: %w", err)
		}
	}

	return nil
}

// AddWatcher adds a configuration change watcher
func (c *DesktopConfig) AddWatcher(watcher ConfigWatcher) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.watchers = append(c.watchers, watcher)
}

// StartWatching starts watching the configuration file for changes
func (c *DesktopConfig) StartWatching(interval time.Duration) chan struct{} {
	stopCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var lastModTime time.Time
		if stat, err := os.Stat(c.configPath); err == nil {
			lastModTime = stat.ModTime()
		}

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				stat, err := os.Stat(c.configPath)
				if err != nil {
					continue
				}

				if stat.ModTime().After(lastModTime) {
					lastModTime = stat.ModTime()
					if err := c.Reload(); err != nil {
						// Log error but continue watching
						fmt.Fprintf(os.Stderr, "Failed to reload config: %v\n", err)
					}
				}
			}
		}
	}()

	return stopCh
}

// Validate checks if the configuration is valid
func (c *DesktopConfig) Validate() error {
	// Validate database configuration
	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}
	if c.Database.GCInterval < 1 {
		return fmt.Errorf("database GC interval must be at least 1 minute")
	}

	// Validate discovery configuration
	if c.Discovery.ScanInterval < 1 {
		return fmt.Errorf("scan interval must be at least 1 second")
	}
	if c.Discovery.InactiveTimeout < 1 {
		return fmt.Errorf("inactive timeout must be at least 1 minute")
	}

	// Validate detection configuration
	if c.Detection.Sensitivity < 0.0 || c.Detection.Sensitivity > 1.0 {
		return fmt.Errorf("detection sensitivity must be between 0.0 and 1.0")
	}

	// Validate visualizer configuration
	if c.Visualizer.Port < 1 || c.Visualizer.Port > 65535 {
		return fmt.Errorf("visualizer port must be between 1 and 65535")
	}

	// Validate feature gate configuration
	validTiers := map[string]bool{
		"free":       true,
		"pro":        true,
		"enterprise": true,
	}
	if !validTiers[c.FeatureGate.Tier] {
		return fmt.Errorf("invalid tier: %s (must be free, pro, or enterprise)", c.FeatureGate.Tier)
	}

	// Validate cloud configuration if enabled
	if c.Cloud.Enabled {
		if c.Cloud.Provider != "aws" && c.Cloud.Provider != "gcp" {
			return fmt.Errorf("cloud provider must be 'aws' or 'gcp'")
		}

		if c.Cloud.Provider == "aws" {
			if c.Cloud.AWS.Endpoint == "" {
				return fmt.Errorf("AWS endpoint cannot be empty when cloud is enabled")
			}
			if c.Cloud.AWS.ClientID == "" {
				return fmt.Errorf("AWS client ID cannot be empty when cloud is enabled")
			}
			if c.Cloud.AWS.CertPath == "" {
				return fmt.Errorf("AWS certificate path cannot be empty when cloud is enabled")
			}
			if c.Cloud.AWS.KeyPath == "" {
				return fmt.Errorf("AWS key path cannot be empty when cloud is enabled")
			}
			// Verify certificate files exist
			if _, err := os.Stat(c.Cloud.AWS.CertPath); os.IsNotExist(err) {
				return fmt.Errorf("AWS certificate file not found: %s", c.Cloud.AWS.CertPath)
			}
			if _, err := os.Stat(c.Cloud.AWS.KeyPath); os.IsNotExist(err) {
				return fmt.Errorf("AWS key file not found: %s", c.Cloud.AWS.KeyPath)
			}
		}

		if c.Cloud.Provider == "gcp" {
			if c.Cloud.GCP.ProjectID == "" {
				return fmt.Errorf("GCP project ID cannot be empty when cloud is enabled")
			}
			if c.Cloud.GCP.TopicID == "" {
				return fmt.Errorf("GCP topic ID cannot be empty when cloud is enabled")
			}
		}
	}

	// Validate logging configuration
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}
	if c.Logging.File == "" {
		return fmt.Errorf("log file path cannot be empty")
	}

	// Validate log file directory exists or can be created
	logDir := filepath.Dir(c.Logging.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("cannot create log directory %s: %w", logDir, err)
	}

	return nil
}
