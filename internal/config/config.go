// Package config provides configuration management for the Heimdal sensor.
//
// The configuration is loaded from a JSON file (default: /etc/heimdal/config.json)
// and contains settings for all sensor components including database, network,
// discovery, interception, profiling, API, cloud connectivity, and logging.
//
// Configuration Structure:
//   - Database: BadgerDB path and garbage collection settings
//   - Network: Interface selection and auto-detection
//   - Discovery: ARP scan intervals, mDNS settings, inactive timeout
//   - Interceptor: ARP spoofing enable/disable, spoof interval, target MACs
//   - Profiler: Persistence interval, max destinations per profile
//   - API: Host, port, rate limiting
//   - Cloud: Provider selection, AWS IoT and Google Cloud settings
//   - Logging: Log level and file path
//
// The LoadConfig function reads and parses the configuration file, validates
// required fields, and returns a Config struct. Default values are provided
// for optional fields.
//
// Example configuration file location:
//   /etc/heimdal/config.json
//
// See CONFIG.md for complete configuration reference and examples.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the complete application configuration
type Config struct {
	Database    DatabaseConfig    `json:"database"`
	Network     NetworkConfig     `json:"network"`
	Discovery   DiscoveryConfig   `json:"discovery"`
	Interceptor InterceptorConfig `json:"interceptor"`
	Profiler    ProfilerConfig    `json:"profiler"`
	API         APIConfig         `json:"api"`
	Cloud       CloudConfig       `json:"cloud"`
	Logging     LoggingConfig     `json:"logging"`
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
	ARPScanInterval int  `json:"arp_scan_interval_seconds"`
	MDNSEnabled     bool `json:"mdns_enabled"`
	InactiveTimeout int  `json:"inactive_timeout_minutes"`
}

// InterceptorConfig contains traffic interception settings
type InterceptorConfig struct {
	Enabled       bool     `json:"enabled"`
	SpoofInterval int      `json:"spoof_interval_seconds"`
	TargetMACs    []string `json:"target_macs"`
}

// ProfilerConfig contains behavioral profiling settings
type ProfilerConfig struct {
	PersistInterval int `json:"persist_interval_seconds"`
	MaxDestinations int `json:"max_destinations"`
}

// APIConfig contains web API settings
type APIConfig struct {
	Port               int    `json:"port"`
	Host               string `json:"host"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
}

// CloudConfig contains cloud connectivity settings
type CloudConfig struct {
	Enabled  bool      `json:"enabled"`
	Provider string    `json:"provider"`
	AWS      AWSConfig `json:"aws"`
	GCP      GCPConfig `json:"gcp"`
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
	Level string `json:"level"`
	File  string `json:"file"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Path:       "/var/lib/heimdal/db",
			GCInterval: 5,
		},
		Network: NetworkConfig{
			Interface:  "",
			AutoDetect: true,
		},
		Discovery: DiscoveryConfig{
			ARPScanInterval: 60,
			MDNSEnabled:     true,
			InactiveTimeout: 5,
		},
		Interceptor: InterceptorConfig{
			Enabled:       true,
			SpoofInterval: 2,
			TargetMACs:    []string{},
		},
		Profiler: ProfilerConfig{
			PersistInterval: 60,
			MaxDestinations: 100,
		},
		API: APIConfig{
			Port:               8080,
			Host:               "0.0.0.0",
			RateLimitPerMinute: 100,
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
			File:  "/var/log/heimdal/heimdal.log",
		},
	}
}

// LoadConfig loads configuration from a JSON file
// If the file doesn't exist or is invalid, it returns the default configuration
func LoadConfig(path string) (*Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", path)
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

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate database configuration
	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}
	if c.Database.GCInterval < 1 {
		return fmt.Errorf("database GC interval must be at least 1 minute")
	}

	// Validate discovery configuration
	if c.Discovery.ARPScanInterval < 1 {
		return fmt.Errorf("ARP scan interval must be at least 1 second")
	}
	if c.Discovery.InactiveTimeout < 1 {
		return fmt.Errorf("inactive timeout must be at least 1 minute")
	}

	// Validate interceptor configuration
	if c.Interceptor.SpoofInterval < 1 {
		return fmt.Errorf("spoof interval must be at least 1 second")
	}

	// Validate profiler configuration
	if c.Profiler.PersistInterval < 1 {
		return fmt.Errorf("persist interval must be at least 1 second")
	}
	if c.Profiler.MaxDestinations < 1 {
		return fmt.Errorf("max destinations must be at least 1")
	}

	// Validate API configuration
	if c.API.Port < 1 || c.API.Port > 65535 {
		return fmt.Errorf("API port must be between 1 and 65535")
	}
	if c.API.Host == "" {
		return fmt.Errorf("API host cannot be empty")
	}
	if c.API.RateLimitPerMinute < 1 {
		return fmt.Errorf("rate limit must be at least 1 request per minute")
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
