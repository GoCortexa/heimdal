// +build property

package property

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/config"
)

// Feature: monorepo-architecture, Property 18: Configuration Hot-Reload
// Validates: Requirements 13.6
func TestProperty_ConfigurationHotReload(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Configuration changes are applied without restart",
		prop.ForAll(
			func(newPort, newScanInterval, newGCInterval int, newSensitivity float64, newTier, newLogLevel string) bool {
				// Create a temporary config file
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.json")

				// Create initial config
				initialCfg, err := config.DefaultConfig()
				if err != nil {
					t.Logf("Failed to create default config: %v", err)
					return false
				}
				initialCfg.Visualizer.Port = 8080
				initialCfg.Discovery.ScanInterval = 60
				initialCfg.Database.GCInterval = 5
				initialCfg.Detection.Sensitivity = 0.5
				initialCfg.FeatureGate.Tier = "free"
				initialCfg.Logging.Level = "info"

				// Save initial config to file
				data, err := json.MarshalIndent(initialCfg, "", "  ")
				if err != nil {
					t.Logf("Failed to marshal initial config: %v", err)
					return false
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Logf("Failed to write initial config: %v", err)
					return false
				}

				// Load config from file
				cfg, err := config.LoadConfigFromPath(configPath)
				if err != nil {
					t.Logf("Failed to load config: %v", err)
					return false
				}

				// Verify initial values
				if cfg.Visualizer.Port != 8080 {
					t.Logf("Initial port mismatch: expected 8080, got %d", cfg.Visualizer.Port)
					return false
				}

				// Track if watcher was called
				watcherCalled := false
				cfg.AddWatcher(func(c *config.DesktopConfig) error {
					watcherCalled = true
					return nil
				})

				// Modify config file with new values
				cfg.Visualizer.Port = newPort
				cfg.Discovery.ScanInterval = newScanInterval
				cfg.Database.GCInterval = newGCInterval
				cfg.Detection.Sensitivity = newSensitivity
				cfg.FeatureGate.Tier = newTier
				cfg.Logging.Level = newLogLevel

				// Save modified config
				data, err = json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					t.Logf("Failed to marshal modified config: %v", err)
					return false
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Logf("Failed to write modified config: %v", err)
					return false
				}

				// Wait a bit to ensure file is written
				time.Sleep(10 * time.Millisecond)

				// Reload config
				if err := cfg.Reload(); err != nil {
					t.Logf("Failed to reload config: %v", err)
					return false
				}

				// Verify new values were applied
				if cfg.Visualizer.Port != newPort {
					t.Logf("Port not updated: expected %d, got %d", newPort, cfg.Visualizer.Port)
					return false
				}
				if cfg.Discovery.ScanInterval != newScanInterval {
					t.Logf("ScanInterval not updated: expected %d, got %d", newScanInterval, cfg.Discovery.ScanInterval)
					return false
				}
				if cfg.Database.GCInterval != newGCInterval {
					t.Logf("GCInterval not updated: expected %d, got %d", newGCInterval, cfg.Database.GCInterval)
					return false
				}
				if cfg.Detection.Sensitivity != newSensitivity {
					t.Logf("Sensitivity not updated: expected %f, got %f", newSensitivity, cfg.Detection.Sensitivity)
					return false
				}
				if cfg.FeatureGate.Tier != newTier {
					t.Logf("Tier not updated: expected %s, got %s", newTier, cfg.FeatureGate.Tier)
					return false
				}
				if cfg.Logging.Level != newLogLevel {
					t.Logf("LogLevel not updated: expected %s, got %s", newLogLevel, cfg.Logging.Level)
					return false
				}

				// Verify watcher was called
				if !watcherCalled {
					t.Log("Config watcher was not called after reload")
					return false
				}

				return true
			},
			gen.IntRange(1024, 65535),  // newPort
			gen.IntRange(1, 300),       // newScanInterval
			gen.IntRange(1, 60),        // newGCInterval
			gen.Float64Range(0.0, 1.0), // newSensitivity
			genValidTier(),             // newTier
			genValidLogLevel(),         // newLogLevel
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 18: Configuration Hot-Reload (Invalid Changes)
// Validates: Requirements 13.6
func TestProperty_ConfigurationHotReloadRejectsInvalid(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Invalid configuration changes are rejected during reload",
		prop.ForAll(
			func(invalidField string) bool {
				// Create a temporary config file
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.json")

				// Create initial valid config
				initialCfg, err := config.DefaultConfig()
				if err != nil {
					t.Logf("Failed to create default config: %v", err)
					return false
				}

				// Save initial config to file
				data, err := json.MarshalIndent(initialCfg, "", "  ")
				if err != nil {
					t.Logf("Failed to marshal initial config: %v", err)
					return false
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Logf("Failed to write initial config: %v", err)
					return false
				}

				// Load config from file
				cfg, err := config.LoadConfigFromPath(configPath)
				if err != nil {
					t.Logf("Failed to load config: %v", err)
					return false
				}

				// Store original values
				originalPort := cfg.Visualizer.Port
				originalScanInterval := cfg.Discovery.ScanInterval

				// Create an invalid config and write to file
				invalidCfg, _ := config.DefaultConfig()
				switch invalidField {
				case "port":
					invalidCfg.Visualizer.Port = 0
				case "scan_interval":
					invalidCfg.Discovery.ScanInterval = 0
				case "sensitivity":
					invalidCfg.Detection.Sensitivity = 2.0
				case "tier":
					invalidCfg.FeatureGate.Tier = "invalid"
				default:
					return true
				}

				// Write invalid config to file
				data, err = json.MarshalIndent(invalidCfg, "", "  ")
				if err != nil {
					t.Logf("Failed to marshal invalid config: %v", err)
					return false
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Logf("Failed to write invalid config: %v", err)
					return false
				}

				// Wait a bit to ensure file is written
				time.Sleep(10 * time.Millisecond)

				// Attempt to reload - should fail
				err = cfg.Reload()
				if err == nil {
					t.Logf("Expected reload to fail for invalid %s, but it succeeded", invalidField)
					return false
				}

				// Verify original values are preserved
				if cfg.Visualizer.Port != originalPort {
					t.Logf("Port changed after failed reload: expected %d, got %d", originalPort, cfg.Visualizer.Port)
					return false
				}
				if cfg.Discovery.ScanInterval != originalScanInterval {
					t.Logf("ScanInterval changed after failed reload: expected %d, got %d", originalScanInterval, cfg.Discovery.ScanInterval)
					return false
				}

				return true
			},
			gen.OneConstOf("port", "scan_interval", "sensitivity", "tier"),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
