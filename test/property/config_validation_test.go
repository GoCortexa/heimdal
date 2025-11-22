// +build property

package property

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/config"
)

// Feature: monorepo-architecture, Property 17: Configuration Validation
// Validates: Requirements 13.5
func TestProperty_ConfigurationValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Invalid configurations are detected with clear error messages",
		prop.ForAll(
			func(invalidField string) bool {
				// Start with a valid default config
				cfg, err := config.DefaultConfig()
				if err != nil {
					t.Logf("Failed to create default config: %v", err)
					return false
				}

				// Introduce an invalid value based on the field
				switch invalidField {
				case "database_path":
					cfg.Database.Path = ""
				case "database_gc_interval":
					cfg.Database.GCInterval = 0
				case "scan_interval":
					cfg.Discovery.ScanInterval = 0
				case "inactive_timeout":
					cfg.Discovery.InactiveTimeout = 0
				case "detection_sensitivity_negative":
					cfg.Detection.Sensitivity = -0.5
				case "detection_sensitivity_high":
					cfg.Detection.Sensitivity = 1.5
				case "visualizer_port_zero":
					cfg.Visualizer.Port = 0
				case "visualizer_port_high":
					cfg.Visualizer.Port = 70000
				case "tier":
					cfg.FeatureGate.Tier = "invalid_tier"
				case "log_level":
					cfg.Logging.Level = "invalid_level"
				case "log_file":
					cfg.Logging.File = ""
				default:
					// Unknown field, skip
					return true
				}

				// Validate should return an error
				err = cfg.Validate()
				if err == nil {
					t.Logf("Expected validation error for invalid %s, got nil", invalidField)
					return false
				}

				// Error message should be non-empty
				if err.Error() == "" {
					t.Log("Validation error message is empty")
					return false
				}

				return true
			},
			genInvalidConfigField(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 17: Configuration Validation (Valid Configs)
// Validates: Requirements 13.5
func TestProperty_ValidConfigurationAccepted(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Valid configurations pass validation",
		prop.ForAll(
			func(scanInterval, inactiveTimeout, gcInterval, port int, sensitivity float64, tier, logLevel string) bool {
				// Create a config with valid random values
				cfg, err := config.DefaultConfig()
				if err != nil {
					t.Logf("Failed to create default config: %v", err)
					return false
				}

				// Set valid random values
				cfg.Discovery.ScanInterval = scanInterval
				cfg.Discovery.InactiveTimeout = inactiveTimeout
				cfg.Database.GCInterval = gcInterval
				cfg.Visualizer.Port = port
				cfg.Detection.Sensitivity = sensitivity
				cfg.FeatureGate.Tier = tier
				cfg.Logging.Level = logLevel

				// Validate should succeed
				err = cfg.Validate()
				if err != nil {
					t.Logf("Expected valid config to pass validation, got error: %v", err)
					return false
				}

				return true
			},
			gen.IntRange(1, 300),      // scanInterval
			gen.IntRange(1, 60),       // inactiveTimeout
			gen.IntRange(1, 60),       // gcInterval
			gen.IntRange(1024, 65535), // port
			gen.Float64Range(0.0, 1.0), // sensitivity
			genValidTier(),            // tier
			genValidLogLevel(),        // logLevel
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genInvalidConfigField generates field names that can be made invalid
func genInvalidConfigField() gopter.Gen {
	return gen.OneConstOf(
		"database_path",
		"database_gc_interval",
		"scan_interval",
		"inactive_timeout",
		"detection_sensitivity_negative",
		"detection_sensitivity_high",
		"visualizer_port_zero",
		"visualizer_port_high",
		"tier",
		"log_level",
		"log_file",
	)
}

// genValidTier generates valid tier values
func genValidTier() gopter.Gen {
	return gen.OneConstOf("free", "pro", "enterprise")
}

// genValidLogLevel generates valid log level values
func genValidLogLevel() gopter.Gen {
	return gen.OneConstOf("debug", "info", "warn", "error")
}
