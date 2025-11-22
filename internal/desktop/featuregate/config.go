package featuregate

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the feature gate configuration
type Config struct {
	Tier       string `json:"tier"`        // "free", "pro", "enterprise"
	LicenseKey string `json:"license_key"` // License key for validation
}

// LoadConfig loads feature gate configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate tier
	tier := Tier(c.Tier)
	switch tier {
	case TierFree, TierPro, TierEnterprise:
		// Valid tier
	default:
		return fmt.Errorf("invalid tier: %s (must be 'free', 'pro', or 'enterprise')", c.Tier)
	}

	// License key is optional for free tier
	if tier != TierFree && c.LicenseKey == "" {
		return fmt.Errorf("license key is required for %s tier", tier)
	}

	return nil
}

// InitializeFromConfig creates a FeatureGate from configuration
func InitializeFromConfig(configPath string, validator LicenseValidator) (*FeatureGate, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	tier := Tier(config.Tier)

	// For non-free tiers, validate the license key
	if tier != TierFree && validator != nil {
		validatedTier, err := validator.ValidateLicense(config.LicenseKey)
		if err != nil {
			return nil, fmt.Errorf("license validation failed: %w", err)
		}

		// Ensure the validated tier matches the configured tier
		if validatedTier != tier {
			return nil, fmt.Errorf("license key tier (%s) does not match configured tier (%s)", validatedTier, tier)
		}
	}

	return New(tier, config.LicenseKey, validator), nil
}

// SaveConfig saves feature gate configuration to a file
func SaveConfig(configPath string, config *Config) error {
	if configPath == "" {
		return fmt.Errorf("config path is empty")
	}

	// Validate config before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
