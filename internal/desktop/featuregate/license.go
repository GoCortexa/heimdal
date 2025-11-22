package featuregate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// LicenseValidator defines the interface for license validation
type LicenseValidator interface {
	// ValidateLicense validates a license key and returns the associated tier
	ValidateLicense(licenseKey string) (Tier, error)
}

// LocalLicenseValidator validates license keys locally using a simple format
// Format: TIER-HASH where HASH is sha256(TIER + secret)
type LocalLicenseValidator struct {
	secret string
}

// NewLocalLicenseValidator creates a new local license validator
func NewLocalLicenseValidator(secret string) *LocalLicenseValidator {
	return &LocalLicenseValidator{
		secret: secret,
	}
}

// ValidateLicense validates a license key locally
func (v *LocalLicenseValidator) ValidateLicense(licenseKey string) (Tier, error) {
	if licenseKey == "" {
		return "", fmt.Errorf("license key is empty")
	}

	// Parse license key format: TIER-HASH
	parts := strings.Split(licenseKey, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid license key format")
	}

	tierStr := strings.ToLower(parts[0])
	providedHash := parts[1]

	// Validate tier
	var tier Tier
	switch tierStr {
	case "free":
		tier = TierFree
	case "pro":
		tier = TierPro
	case "enterprise":
		tier = TierEnterprise
	default:
		return "", fmt.Errorf("invalid tier in license key: %s", tierStr)
	}

	// Compute expected hash
	expectedHash := v.computeHash(string(tier))

	// Compare hashes
	if providedHash != expectedHash {
		return "", fmt.Errorf("invalid license key signature")
	}

	return tier, nil
}

// computeHash computes the hash for a tier
func (v *LocalLicenseValidator) computeHash(tier string) string {
	data := tier + v.secret
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CloudLicenseValidator validates license keys against a cloud service
type CloudLicenseValidator struct {
	apiEndpoint string
	apiKey      string
}

// NewCloudLicenseValidator creates a new cloud-based license validator
func NewCloudLicenseValidator(apiEndpoint, apiKey string) *CloudLicenseValidator {
	return &CloudLicenseValidator{
		apiEndpoint: apiEndpoint,
		apiKey:      apiKey,
	}
}

// ValidateLicense validates a license key against the cloud service
func (v *CloudLicenseValidator) ValidateLicense(licenseKey string) (Tier, error) {
	if licenseKey == "" {
		return "", fmt.Errorf("license key is empty")
	}

	// TODO: Implement actual cloud validation
	// For now, this is a placeholder that would make HTTP requests to the cloud service
	// The cloud service would verify the license key and return the tier

	return "", fmt.Errorf("cloud license validation not yet implemented")
}
