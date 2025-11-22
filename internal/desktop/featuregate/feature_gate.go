package featuregate

import (
	"fmt"
)

// Tier represents subscription levels
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
)

// Feature represents a gated feature
type Feature string

const (
	FeatureNetworkVisibility Feature = "network_visibility" // Free+
	FeatureTrafficBlocking   Feature = "traffic_blocking"   // Pro+
	FeatureAdvancedFiltering Feature = "advanced_filtering" // Pro+
	FeatureCloudSync         Feature = "cloud_sync"         // Pro+
	FeatureMultiDevice       Feature = "multi_device"       // Enterprise
	FeatureAPIAccess         Feature = "api_access"         // Enterprise
)

// FeatureGate enforces tier-based feature access
type FeatureGate struct {
	currentTier Tier
	licenseKey  string
	validator   LicenseValidator
}

// FeatureAccessError represents an error when a feature is not accessible
type FeatureAccessError struct {
	Feature      Feature
	CurrentTier  Tier
	RequiredTier Tier
	Message      string
}

func (e *FeatureAccessError) Error() string {
	return e.Message
}

// New creates a new FeatureGate with the specified tier
func New(tier Tier, licenseKey string, validator LicenseValidator) *FeatureGate {
	return &FeatureGate{
		currentTier: tier,
		licenseKey:  licenseKey,
		validator:   validator,
	}
}

// CanAccess checks if the current tier can access a feature
func (fg *FeatureGate) CanAccess(feature Feature) bool {
	requiredTier := fg.getRequiredTier(feature)
	return fg.tierLevel(fg.currentTier) >= fg.tierLevel(requiredTier)
}

// CheckAccess checks if the current tier can access a feature and returns an error if not
func (fg *FeatureGate) CheckAccess(feature Feature) error {
	if !fg.CanAccess(feature) {
		return &FeatureAccessError{
			Feature:      feature,
			CurrentTier:  fg.currentTier,
			RequiredTier: fg.getRequiredTier(feature),
			Message: fmt.Sprintf("Feature '%s' requires %s tier or higher. Your current tier is %s.",
				feature, fg.getRequiredTier(feature), fg.currentTier),
		}
	}
	return nil
}

// GetTier returns the current subscription tier
func (fg *FeatureGate) GetTier() Tier {
	return fg.currentTier
}

// UpgradeTier attempts to upgrade to a new tier with license validation
func (fg *FeatureGate) UpgradeTier(licenseKey string) error {
	if fg.validator == nil {
		return fmt.Errorf("no license validator configured")
	}

	tier, err := fg.validator.ValidateLicense(licenseKey)
	if err != nil {
		return fmt.Errorf("license validation failed: %w", err)
	}

	fg.currentTier = tier
	fg.licenseKey = licenseKey
	return nil
}

// getRequiredTier returns the minimum tier required for a feature
func (fg *FeatureGate) getRequiredTier(feature Feature) Tier {
	switch feature {
	case FeatureNetworkVisibility:
		return TierFree
	case FeatureTrafficBlocking, FeatureAdvancedFiltering, FeatureCloudSync:
		return TierPro
	case FeatureMultiDevice, FeatureAPIAccess:
		return TierEnterprise
	default:
		return TierEnterprise // Unknown features require highest tier
	}
}

// tierLevel returns a numeric level for tier comparison
func (fg *FeatureGate) tierLevel(tier Tier) int {
	switch tier {
	case TierFree:
		return 1
	case TierPro:
		return 2
	case TierEnterprise:
		return 3
	default:
		return 0 // Unknown tiers have no access
	}
}
