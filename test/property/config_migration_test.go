// +build property

package property

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/config"
)

// Feature: monorepo-architecture, Property 19: Backward Configuration Compatibility
// Validates: Requirements 15.3
func TestProperty_BackwardConfigurationCompatibility(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Legacy configuration files are successfully migrated to new format",
		prop.ForAll(
			func(dbPath, logFile, iface, provider string, arpInterval, inactiveTimeout, spoofInterval, persistInterval, maxDest, apiPort, gcInterval int, autoDetect, mdnsEnabled, interceptorEnabled, cloudEnabled bool) bool {
				// Create a temporary directory for test files
				tmpDir, err := os.MkdirTemp("", "heimdal-migration-test-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tmpDir)

				// Create dummy certificate files if cloud is enabled
				var certPath, keyPath string
				if cloudEnabled {
					certPath = filepath.Join(tmpDir, "cert.pem")
					keyPath = filepath.Join(tmpDir, "key.pem")
					// Create dummy cert files
					if err := os.WriteFile(certPath, []byte("dummy cert"), 0644); err != nil {
						t.Logf("Failed to create dummy cert: %v", err)
						return false
					}
					if err := os.WriteFile(keyPath, []byte("dummy key"), 0644); err != nil {
						t.Logf("Failed to create dummy key: %v", err)
						return false
					}
				} else {
					// Use empty paths when cloud is disabled
					certPath = ""
					keyPath = ""
				}

				// Create a legacy configuration
				legacyConfig := config.LegacyConfig{
					Database: config.DatabaseConfig{
						Path:       dbPath,
						GCInterval: gcInterval,
					},
					Network: config.NetworkConfig{
						Interface:  iface,
						AutoDetect: autoDetect,
					},
					Discovery: config.LegacyDiscoveryConfig{
						ARPScanInterval: arpInterval,
						MDNSEnabled:     mdnsEnabled,
						InactiveTimeout: inactiveTimeout,
					},
					Interceptor: config.LegacyInterceptorConfig{
						Enabled:       interceptorEnabled,
						SpoofInterval: spoofInterval,
						TargetMACs:    []string{},
					},
					Profiler: config.ProfilerConfig{
						PersistInterval: persistInterval,
						MaxDestinations: maxDest,
					},
					API: config.APIConfig{
						Port:               apiPort,
						Host:               "0.0.0.0",
						RateLimitPerMinute: 100,
					},
					Cloud: config.CloudConfig{
						Enabled:  cloudEnabled,
						Provider: provider,
						AWS: config.AWSConfig{
							Endpoint: "test-endpoint.iot.us-east-1.amazonaws.com",
							ClientID: "test-client",
							CertPath: certPath,
							KeyPath:  keyPath,
						},
						GCP: config.GCPConfig{
							ProjectID: "test-project",
							TopicID:   "test-topic",
						},
					},
					Logging: config.LoggingConfig{
						Level: "info",
						File:  logFile,
					},
				}

				// Write legacy config to file
				legacyPath := filepath.Join(tmpDir, "config.json")
				legacyData, err := json.MarshalIndent(legacyConfig, "", "  ")
				if err != nil {
					t.Logf("Failed to marshal legacy config: %v", err)
					return false
				}

				if err := os.WriteFile(legacyPath, legacyData, 0644); err != nil {
					t.Logf("Failed to write legacy config: %v", err)
					return false
				}

				// Verify it's detected as legacy format
				isLegacy, err := config.IsLegacyFormat(legacyData)
				if err != nil {
					t.Logf("Failed to detect legacy format: %v", err)
					return false
				}
				if !isLegacy {
					t.Log("Legacy config not detected as legacy format")
					return false
				}

				// Migrate the configuration
				migratedConfig, result, err := config.MigrateLegacyConfig(legacyPath)
				if err != nil {
					t.Logf("Migration failed: %v", err)
					return false
				}

				// Verify migration occurred
				if !result.WasMigrated {
					t.Log("Migration did not occur")
					return false
				}

				// Verify backup was created
				if result.BackupPath == "" {
					t.Log("No backup path in migration result")
					return false
				}

				// Verify migrated config has correct values
				if migratedConfig.Database.Path != dbPath {
					t.Logf("Database path mismatch: expected %s, got %s", dbPath, migratedConfig.Database.Path)
					return false
				}

				if migratedConfig.Network.Interface != iface {
					t.Logf("Network interface mismatch: expected %s, got %s", iface, migratedConfig.Network.Interface)
					return false
				}

				if migratedConfig.Discovery.ARPScanInterval != arpInterval {
					t.Logf("ARP scan interval mismatch: expected %d, got %d", arpInterval, migratedConfig.Discovery.ARPScanInterval)
					return false
				}

				if migratedConfig.Interceptor.Enabled != interceptorEnabled {
					t.Logf("Interceptor enabled mismatch: expected %v, got %v", interceptorEnabled, migratedConfig.Interceptor.Enabled)
					return false
				}

				if migratedConfig.Cloud.Enabled != cloudEnabled {
					t.Logf("Cloud enabled mismatch: expected %v, got %v", cloudEnabled, migratedConfig.Cloud.Enabled)
					return false
				}

				return true
			},
			genValidPath(),                // dbPath
			genValidPath(),                // logFile
			gen.AlphaString(),             // iface
			gen.OneConstOf("aws", "gcp"),  // provider
			gen.IntRange(1, 300),          // arpInterval
			gen.IntRange(1, 60),           // inactiveTimeout
			gen.IntRange(1, 10),           // spoofInterval
			gen.IntRange(1, 300),          // persistInterval
			gen.IntRange(1, 1000),         // maxDest
			gen.IntRange(1024, 65535),     // apiPort
			gen.IntRange(1, 60),           // gcInterval
			gen.Bool(),                    // autoDetect
			gen.Bool(),                    // mdnsEnabled
			gen.Bool(),                    // interceptorEnabled
			gen.Bool(),                    // cloudEnabled
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 19: Backward Configuration Compatibility (In-Place Migration)
// Validates: Requirements 15.3
func TestProperty_InPlaceMigration(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("In-place migration preserves all settings and creates backup",
		prop.ForAll(
			func(arpInterval, inactiveTimeout, spoofInterval int) bool {
				// Create a temporary directory for test files
				tmpDir, err := os.MkdirTemp("", "heimdal-inplace-test-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tmpDir)

				// Create a minimal valid legacy configuration
				legacyConfig := config.LegacyConfig{
					Database: config.DatabaseConfig{
						Path:       filepath.Join(tmpDir, "db"),
						GCInterval: 5,
					},
					Network: config.NetworkConfig{
						Interface:  "eth0",
						AutoDetect: true,
					},
					Discovery: config.LegacyDiscoveryConfig{
						ARPScanInterval: arpInterval,
						MDNSEnabled:     true,
						InactiveTimeout: inactiveTimeout,
					},
					Interceptor: config.LegacyInterceptorConfig{
						Enabled:       true,
						SpoofInterval: spoofInterval,
						TargetMACs:    []string{},
					},
					Profiler: config.ProfilerConfig{
						PersistInterval: 60,
						MaxDestinations: 100,
					},
					API: config.APIConfig{
						Port:               8080,
						Host:               "0.0.0.0",
						RateLimitPerMinute: 100,
					},
					Cloud: config.CloudConfig{
						Enabled:  false,
						Provider: "aws",
					},
					Logging: config.LoggingConfig{
						Level: "info",
						File:  filepath.Join(tmpDir, "heimdal.log"),
					},
				}

				// Write legacy config to file
				configPath := filepath.Join(tmpDir, "config.json")
				legacyData, err := json.MarshalIndent(legacyConfig, "", "  ")
				if err != nil {
					t.Logf("Failed to marshal legacy config: %v", err)
					return false
				}

				if err := os.WriteFile(configPath, legacyData, 0644); err != nil {
					t.Logf("Failed to write legacy config: %v", err)
					return false
				}

				// Perform in-place migration
				result, err := config.MigrateLegacyConfigInPlace(configPath)
				if err != nil {
					t.Logf("In-place migration failed: %v", err)
					return false
				}

				// Verify migration occurred
				if !result.WasMigrated {
					t.Log("Migration did not occur")
					return false
				}

				// Verify backup exists
				if result.BackupPath == "" {
					t.Log("No backup path in migration result")
					return false
				}

				if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
					t.Logf("Backup file does not exist: %s", result.BackupPath)
					return false
				}

				// Read the migrated config
				migratedData, err := os.ReadFile(configPath)
				if err != nil {
					t.Logf("Failed to read migrated config: %v", err)
					return false
				}

				// Parse migrated config
				var migratedConfig config.Config
				if err := json.Unmarshal(migratedData, &migratedConfig); err != nil {
					t.Logf("Failed to parse migrated config: %v", err)
					return false
				}

				// Verify values were preserved
				if migratedConfig.Discovery.ARPScanInterval != arpInterval {
					t.Logf("ARP scan interval not preserved: expected %d, got %d", arpInterval, migratedConfig.Discovery.ARPScanInterval)
					return false
				}

				if migratedConfig.Discovery.InactiveTimeout != inactiveTimeout {
					t.Logf("Inactive timeout not preserved: expected %d, got %d", inactiveTimeout, migratedConfig.Discovery.InactiveTimeout)
					return false
				}

				if migratedConfig.Interceptor.SpoofInterval != spoofInterval {
					t.Logf("Spoof interval not preserved: expected %d, got %d", spoofInterval, migratedConfig.Interceptor.SpoofInterval)
					return false
				}

				return true
			},
			gen.IntRange(1, 300), // arpInterval
			gen.IntRange(1, 60),  // inactiveTimeout
			gen.IntRange(1, 10),  // spoofInterval
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 19: Backward Configuration Compatibility (Non-Legacy Detection)
// Validates: Requirements 15.3
func TestProperty_NonLegacyConfigNotMigrated(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("New format configurations are not migrated",
		prop.ForAll(
			func(scanInterval, inactiveTimeout int) bool {
				// Create a temporary directory for test files
				tmpDir, err := os.MkdirTemp("", "heimdal-nonlegacy-test-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tmpDir)

				// Create a new format configuration (hardware config is same as legacy but we'll add a marker)
				// The key difference is that new configs will have been processed through the new system
				// For this test, we'll create a config with a desktop-specific field to mark it as new
				newConfigMap := map[string]interface{}{
					"database": map[string]interface{}{
						"path":                filepath.Join(tmpDir, "db"),
						"gc_interval_minutes": 5,
					},
					"network": map[string]interface{}{
						"interface":   "eth0",
						"auto_detect": true,
					},
					"discovery": map[string]interface{}{
						"arp_scan_interval_seconds": scanInterval,
						"mdns_enabled":              true,
						"inactive_timeout_minutes":  inactiveTimeout,
					},
					"interceptor": map[string]interface{}{
						"enabled":                true,
						"spoof_interval_seconds": 2,
						"target_macs":            []string{},
					},
					"profiler": map[string]interface{}{
						"persist_interval_seconds": 60,
						"max_destinations":         100,
					},
					"api": map[string]interface{}{
						"port":                  8080,
						"host":                  "0.0.0.0",
						"rate_limit_per_minute": 100,
					},
					"cloud": map[string]interface{}{
						"enabled":  false,
						"provider": "aws",
					},
					"logging": map[string]interface{}{
						"level": "info",
						"file":  filepath.Join(tmpDir, "heimdal.log"),
					},
					// Add desktop-specific fields to mark this as new format
					// Need at least 2 of the 4 new fields to not be detected as legacy
					"detection": map[string]interface{}{
						"enabled":     true,
						"sensitivity": 0.7,
					},
					"visualizer": map[string]interface{}{
						"enabled": true,
						"port":    8080,
					},
				}

				// Write new format config to file
				configPath := filepath.Join(tmpDir, "config.json")
				newData, err := json.MarshalIndent(newConfigMap, "", "  ")
				if err != nil {
					t.Logf("Failed to marshal new config: %v", err)
					return false
				}

				if err := os.WriteFile(configPath, newData, 0644); err != nil {
					t.Logf("Failed to write new config: %v", err)
					return false
				}

				// Verify it's NOT detected as legacy format
				isLegacy, err := config.IsLegacyFormat(newData)
				if err != nil {
					t.Logf("Failed to detect format: %v", err)
					return false
				}
				if isLegacy {
					t.Log("New format config incorrectly detected as legacy")
					return false
				}

				// Attempt migration
				migratedConfig, result, err := config.MigrateLegacyConfig(configPath)
				if err != nil {
					t.Logf("Migration check failed: %v", err)
					return false
				}

				// Verify no migration occurred
				if result.WasMigrated {
					t.Log("New format config was incorrectly migrated")
					return false
				}

				// Verify nil config returned (indicating no migration needed)
				if migratedConfig != nil {
					t.Log("Expected nil config for non-legacy format")
					return false
				}

				return true
			},
			gen.IntRange(1, 300),   // scanInterval
			gen.IntRange(1, 60),    // inactiveTimeout
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genValidPath generates a valid file path
func genValidPath() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	}).Map(func(s string) string {
		return "/tmp/" + s
	})
}
