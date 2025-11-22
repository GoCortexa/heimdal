# Feature Gate Module

The feature gate module implements tier-based feature access control for the Heimdal Desktop product, supporting Free, Pro, and Enterprise subscription tiers.

## Components

### 1. Feature Gate Core (`feature_gate.go`)

The main feature gate implementation that enforces tier-based access control.

**Key Types:**
- `Tier`: Subscription levels (Free, Pro, Enterprise)
- `Feature`: Gated features (NetworkVisibility, TrafficBlocking, etc.)
- `FeatureGate`: Main struct that manages feature access

**Key Methods:**
- `CanAccess(feature Feature) bool`: Check if current tier can access a feature
- `CheckAccess(feature Feature) error`: Check access and return detailed error if denied
- `GetTier() Tier`: Get current subscription tier
- `UpgradeTier(licenseKey string) error`: Upgrade tier with license validation

**Feature-to-Tier Mapping:**
- Free: NetworkVisibility
- Pro: TrafficBlocking, AdvancedFiltering, CloudSync
- Enterprise: MultiDevice, APIAccess

### 2. License Validation (`license.go`)

Implements license key validation for tier upgrades.

**Validators:**

#### LocalLicenseValidator
Validates license keys locally using SHA256 hashing.

**Format:** `TIER-HASH` where HASH = sha256(TIER + secret)

**Example:**
```go
validator := NewLocalLicenseValidator("my-secret")
tier, err := validator.ValidateLicense("pro-abc123...")
```

#### CloudLicenseValidator
Validates license keys against a cloud service (placeholder for future implementation).

### 3. Configuration Management (`config.go`)

Handles loading and saving feature gate configuration.

**Configuration Format:**
```json
{
  "tier": "pro",
  "license_key": "pro-abc123..."
}
```

**Key Functions:**
- `LoadConfig(configPath string) (*Config, error)`: Load config from file
- `SaveConfig(configPath string, config *Config) error`: Save config to file
- `InitializeFromConfig(configPath string, validator LicenseValidator) (*FeatureGate, error)`: Create FeatureGate from config

**Validation:**
- Tier must be "free", "pro", or "enterprise"
- License key required for Pro and Enterprise tiers
- License key validated on initialization

## Usage Example

```go
// Initialize with local validator
validator := featuregate.NewLocalLicenseValidator("secret")
fg, err := featuregate.InitializeFromConfig("/path/to/config.json", validator)
if err != nil {
    log.Fatal(err)
}

// Check feature access
if fg.CanAccess(featuregate.FeatureTrafficBlocking) {
    // Enable traffic blocking
} else {
    // Show upgrade prompt
}

// Check with detailed error
if err := fg.CheckAccess(featuregate.FeatureAPIAccess); err != nil {
    // err contains tier information and upgrade message
    log.Printf("Access denied: %v", err)
}
```

## Testing

### Property-Based Tests (`test/property/feature_gate_test.go`)

Two property tests validate correctness across all tier/feature combinations:

1. **Property 7: Feature Gate Access Control**
   - Validates: Requirements 5.4
   - Tests: Access control logic for all tier/feature combinations
   - Iterations: 100

2. **Property 8: Feature Gate Error Messages**
   - Validates: Requirements 5.5
   - Tests: Error messages for denied access
   - Iterations: 100

### Unit Tests (`config_test.go`)

- Configuration loading and validation
- Configuration saving and round-trip
- Local license validator functionality

## Requirements Coverage

This implementation satisfies the following requirements:

- **5.1**: Define tier levels (Free, Pro, Enterprise)
- **5.2**: Free tier enables read-only network visibility
- **5.3**: Pro tier enables active traffic blocking
- **5.4**: Check tier permissions before protected operations
- **5.5**: Provide clear error messages for denied access
- **5.6**: Support tier configuration through local files and license validation
- **13.2**: Configuration format includes feature gate settings
