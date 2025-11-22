package featuregate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
	}{
		{
			name:        "Valid free tier config",
			configJSON:  `{"tier": "free", "license_key": ""}`,
			expectError: false,
		},
		{
			name:        "Valid pro tier config",
			configJSON:  `{"tier": "pro", "license_key": "test-key"}`,
			expectError: false,
		},
		{
			name:        "Valid enterprise tier config",
			configJSON:  `{"tier": "enterprise", "license_key": "test-key"}`,
			expectError: false,
		},
		{
			name:        "Invalid tier",
			configJSON:  `{"tier": "invalid", "license_key": ""}`,
			expectError: true,
		},
		{
			name:        "Pro tier without license key",
			configJSON:  `{"tier": "pro", "license_key": ""}`,
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			configJSON:  `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			err := os.WriteFile(configPath, []byte(tt.configJSON), 0600)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Load config
			config, err := LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if config == nil {
					t.Errorf("Expected config but got nil")
				}
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &Config{
		Tier:       "pro",
		LicenseKey: "test-key",
	}

	// Save config
	err := SaveConfig(configPath, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load it back
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Tier != config.Tier {
		t.Errorf("Expected tier %s, got %s", config.Tier, loadedConfig.Tier)
	}

	if loadedConfig.LicenseKey != config.LicenseKey {
		t.Errorf("Expected license key %s, got %s", config.LicenseKey, loadedConfig.LicenseKey)
	}
}

func TestLocalLicenseValidator(t *testing.T) {
	secret := "test-secret"
	validator := NewLocalLicenseValidator(secret)

	// Generate a valid license key
	tier := TierPro
	hash := validator.computeHash(string(tier))
	validKey := string(tier) + "-" + hash

	// Test valid license
	validatedTier, err := validator.ValidateLicense(validKey)
	if err != nil {
		t.Errorf("Valid license failed validation: %v", err)
	}
	if validatedTier != tier {
		t.Errorf("Expected tier %s, got %s", tier, validatedTier)
	}

	// Test invalid license
	invalidKey := "pro-invalidhash"
	_, err = validator.ValidateLicense(invalidKey)
	if err == nil {
		t.Errorf("Invalid license passed validation")
	}

	// Test empty license
	_, err = validator.ValidateLicense("")
	if err == nil {
		t.Errorf("Empty license passed validation")
	}

	// Test malformed license
	_, err = validator.ValidateLicense("malformed")
	if err == nil {
		t.Errorf("Malformed license passed validation")
	}
}
