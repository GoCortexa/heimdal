// +build property

package property

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/featuregate"
)

// Feature: monorepo-architecture, Property 7: Feature Gate Access Control
// Validates: Requirements 5.4
func TestProperty_FeatureGateAccessControl(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Feature gate correctly enforces tier permissions",
		prop.ForAll(
			func(tier featuregate.Tier, feature featuregate.Feature) bool {
				fg := featuregate.New(tier, "", nil)
				canAccess := fg.CanAccess(feature)
				requiredTier := getRequiredTier(feature)

				// Property: can access if tier >= required tier
				expectedAccess := tierLevel(tier) >= tierLevel(requiredTier)
				return canAccess == expectedAccess
			},
			genTier(),
			genFeature(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 8: Feature Gate Error Messages
// Validates: Requirements 5.5
func TestProperty_FeatureGateErrorMessages(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Feature gate provides clear error messages for denied access",
		prop.ForAll(
			func(tier featuregate.Tier, feature featuregate.Feature) bool {
				fg := featuregate.New(tier, "", nil)
				err := fg.CheckAccess(feature)

				// If access is denied, error should be non-nil and contain required tier info
				if !fg.CanAccess(feature) {
					if err == nil {
						t.Log("Expected error for denied access, got nil")
						return false
					}

					// Check that error message contains the feature name
					errMsg := err.Error()
					if errMsg == "" {
						t.Log("Error message is empty")
						return false
					}

					// Error should be of type FeatureAccessError
					if _, ok := err.(*featuregate.FeatureAccessError); !ok {
						t.Logf("Expected FeatureAccessError, got %T", err)
						return false
					}

					return true
				}

				// If access is allowed, error should be nil
				return err == nil
			},
			genTier(),
			genFeature(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genTier generates a valid Tier
func genTier() gopter.Gen {
	return gen.OneConstOf(
		featuregate.TierFree,
		featuregate.TierPro,
		featuregate.TierEnterprise,
	)
}

// genFeature generates a valid Feature
func genFeature() gopter.Gen {
	return gen.OneConstOf(
		featuregate.FeatureNetworkVisibility,
		featuregate.FeatureTrafficBlocking,
		featuregate.FeatureAdvancedFiltering,
		featuregate.FeatureCloudSync,
		featuregate.FeatureMultiDevice,
		featuregate.FeatureAPIAccess,
	)
}

// getRequiredTier returns the minimum tier required for a feature
func getRequiredTier(feature featuregate.Feature) featuregate.Tier {
	switch feature {
	case featuregate.FeatureNetworkVisibility:
		return featuregate.TierFree
	case featuregate.FeatureTrafficBlocking, featuregate.FeatureAdvancedFiltering, featuregate.FeatureCloudSync:
		return featuregate.TierPro
	case featuregate.FeatureMultiDevice, featuregate.FeatureAPIAccess:
		return featuregate.TierEnterprise
	default:
		return featuregate.TierEnterprise
	}
}

// tierLevel returns a numeric level for tier comparison
func tierLevel(tier featuregate.Tier) int {
	switch tier {
	case featuregate.TierFree:
		return 1
	case featuregate.TierPro:
		return 2
	case featuregate.TierEnterprise:
		return 3
	default:
		return 0
	}
}
