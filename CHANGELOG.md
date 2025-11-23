# Changelog

All notable changes to Heimdal will be documented in this file.

## [Unreleased]

### Added - Device Identification & Enrichment
- **OUI Vendor Lookup**: Embedded IEEE OUI database with 38,400+ vendors for automatic manufacturer identification
- **Device Classification**: Smart device type detection (Phone, Computer, IoT, Printer, TV, Router, etc.) using vendor, hostname, and service patterns
- **Hostname Resolution**: Multi-method hostname discovery (mDNS, reverse DNS) with async enrichment
- **Service Tracking**: mDNS service discovery for improved device classification
- **Rich Device Profiles**: Extended device data with manufacturer, type, hostname, and services

### Added - Enhanced Anomaly Detection
- **Statistical Baselines**: Rolling baseline metrics using exponential moving averages
- **Z-Score Detection**: Statistical anomaly detection for traffic spikes
- **Protocol Shift Detection**: Identify changes in protocol distribution patterns
- **Destination Anomalies**: Detect unusual changes in communication partners
- **Baseline Persistence**: Store and track baselines across restarts

### Added - Cloud Integration
- **Cloud Payload Schemas**: Standardized message formats for Asgard platform integration
- **Privacy Controls**: Granular opt-in/opt-out for device info, profiles, anomalies, and diagnostics
- **Data Anonymization**: Hash sensitive fields before cloud transmission
- **Message Versioning**: Schema versioning for backward compatibility

### Added - Dashboard Improvements
- **Device Icons**: Visual device type indicators (📱💻🖨️📺🔌)
- **Device Type Badges**: Color-coded type labels
- **Manufacturer Display**: Show full manufacturer name alongside vendor
- **Filter & Search**: Filter by device type, search by name/IP/MAC/vendor
- **Anomaly Section**: Dedicated UI for viewing detected anomalies (planned)

### Added - Development & Deployment
- **Automated Packaging**: `build/package-all.sh` script for multi-platform builds
- **GitHub Actions CI**: Automated build, test, and release workflow
- **Device Identification Docs**: Comprehensive documentation on identification system

### Improved - Discovery System
- **Status Reporting**: Scanner health monitoring with system tray integration
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Permission Guidance**: Platform-specific guidance for packet capture permissions
- **Error Handling**: Improved error messages and recovery

### Technical Details
- OUI database: 6.2MB embedded, 38,400 entries, <1µs lookup time
- Classification: 100+ vendor rules, 50+ hostname patterns, 15+ service types
- Baselines: EMA with α=0.2, calculated every persist interval
- Anomaly detection: Z-score thresholds adjusted by sensitivity (0.0-1.0)

## [1.0.0] - 2024-01-15

### Initial Release
- Network device discovery via ARP and mDNS
- Traffic interception using ARP spoofing
- Behavioral profiling with BadgerDB storage
- Web dashboard for monitoring
- Hardware (Raspberry Pi) and Desktop (Windows/macOS/Linux) products
- Ansible deployment for hardware
- System tray integration for desktop
- Feature gate system for tiered licensing

---

## Version Numbering

Heimdal follows [Semantic Versioning](https://semver.org/):
- MAJOR: Incompatible API changes
- MINOR: New functionality (backward compatible)
- PATCH: Bug fixes (backward compatible)

## Release Process

1. Update version in `internal/version/version.go`
2. Update this CHANGELOG.md
3. Run full test suite: `make test`
4. Build all platforms: `./build/package-all.sh`
5. Create git tag: `git tag v1.1.0`
6. Push tag: `git push --tags`
7. GitHub Actions will create release automatically

