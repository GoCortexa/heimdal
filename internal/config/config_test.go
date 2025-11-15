package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify database defaults
	if cfg.Database.Path != "/var/lib/heimdal/db" {
		t.Errorf("expected database path '/var/lib/heimdal/db', got '%s'", cfg.Database.Path)
	}
	if cfg.Database.GCInterval != 5 {
		t.Errorf("expected GC interval 5, got %d", cfg.Database.GCInterval)
	}

	// Verify network defaults
	if cfg.Network.Interface != "" {
		t.Errorf("expected empty interface, got '%s'", cfg.Network.Interface)
	}
	if !cfg.Network.AutoDetect {
		t.Error("expected auto-detect to be true")
	}

	// Verify discovery defaults
	if cfg.Discovery.ARPScanInterval != 60 {
		t.Errorf("expected ARP scan interval 60, got %d", cfg.Discovery.ARPScanInterval)
	}
	if !cfg.Discovery.MDNSEnabled {
		t.Error("expected mDNS to be enabled")
	}

	// Verify API defaults
	if cfg.API.Port != 8080 {
		t.Errorf("expected API port 8080, got %d", cfg.API.Port)
	}

	// Verify cloud is disabled by default
	if cfg.Cloud.Enabled {
		t.Error("expected cloud to be disabled by default")
	}

	// Verify logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("expected log level 'info', got '%s'", cfg.Logging.Level)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create test configuration
	testConfig := DefaultConfig()
	testConfig.API.Port = 9090
	testConfig.Logging.Level = "debug"
	testConfig.Logging.File = filepath.Join(tmpDir, "test.log")

	// Write config to file
	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load configuration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.API.Port != 9090 {
		t.Errorf("expected API port 9090, got %d", cfg.API.Port)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected log level 'debug', got '%s'", cfg.Logging.Level)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		expectErr bool
	}{
		{
			name:      "valid default config",
			modify:    func(c *Config) {},
			expectErr: false,
		},
		{
			name: "empty database path",
			modify: func(c *Config) {
				c.Database.Path = ""
			},
			expectErr: true,
		},
		{
			name: "invalid GC interval",
			modify: func(c *Config) {
				c.Database.GCInterval = 0
			},
			expectErr: true,
		},
		{
			name: "invalid ARP scan interval",
			modify: func(c *Config) {
				c.Discovery.ARPScanInterval = 0
			},
			expectErr: true,
		},
		{
			name: "invalid API port (too low)",
			modify: func(c *Config) {
				c.API.Port = 0
			},
			expectErr: true,
		},
		{
			name: "invalid API port (too high)",
			modify: func(c *Config) {
				c.API.Port = 70000
			},
			expectErr: true,
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			expectErr: true,
		},
		{
			name: "cloud enabled without provider config",
			modify: func(c *Config) {
				c.Cloud.Enabled = true
				c.Cloud.Provider = "aws"
				c.Cloud.AWS.Endpoint = ""
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Logging.File = filepath.Join(t.TempDir(), "test.log")
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.expectErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}
