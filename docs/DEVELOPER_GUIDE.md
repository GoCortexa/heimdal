# Heimdal Developer Guide

This guide is for developers who want to contribute to Heimdal, extend its functionality, or understand its internal architecture.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Development Setup](#development-setup)
3. [Project Structure](#project-structure)
4. [Core Interfaces](#core-interfaces)
5. [Adding Platform Implementations](#adding-platform-implementations)
6. [Testing Strategy](#testing-strategy)
7. [Building and Packaging](#building-and-packaging)
8. [Contributing Guidelines](#contributing-guidelines)
9. [Code Style and Standards](#code-style-and-standards)
10. [Debugging and Troubleshooting](#debugging-and-troubleshooting)

## Architecture Overview

Heimdal uses a monorepo architecture that maximizes code reuse between hardware and desktop products while maintaining clean separation of platform-specific concerns.

### Design Principles

1. **Interface-Driven Design**: Core functionality depends on interfaces, not concrete implementations
2. **Platform Abstraction**: Platform-specific code is isolated behind well-defined interfaces
3. **Shared Core Logic**: Packet analysis, cloud communication, and anomaly detection are shared
4. **Dependency Injection**: Components receive their dependencies through constructors
5. **Concurrent by Default**: Heavy use of goroutines and channels for concurrent operations

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Entry Points                             │
│  cmd/heimdal-hardware/        cmd/heimdal-desktop/          │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                  Product-Specific Layer                      │
│  Hardware Orchestrator        Desktop Orchestrator           │
│                              Feature Gate                    │
│                              Visualizer                      │
│                              System Tray                     │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                  Core Business Logic (Shared)                │
│  Packet Analysis  │  Cloud Comm  │  Detection  │  Profiler  │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│              Platform Abstraction Layer                      │
│  PacketCaptureProvider │ SystemIntegrator │ StorageProvider │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│              Platform Implementations                        │
│  linux_embedded  │  desktop_windows  │  desktop_macos  │    │
│  desktop_linux                                               │
└─────────────────────────────────────────────────────────────┘
```


## Development Setup

### Prerequisites

- **Go**: Version 1.21 or later
- **Git**: For version control
- **Make**: For build automation
- **Platform-specific tools**:
  - **Windows**: MinGW-w64 for CGO, Npcap SDK
  - **macOS**: Xcode Command Line Tools
  - **Linux**: GCC, libpcap-dev

### Initial Setup

1. **Clone the Repository**
   ```bash
   git clone https://github.com/your-org/heimdal.git
   cd heimdal
   ```

2. **Install Go Dependencies**
   ```bash
   go mod download
   ```

3. **Install Development Tools**
   ```bash
   # Install testing tools
   go install github.com/leanovate/gopter@latest
   
   # Install linting tools
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

4. **Verify Setup**
   ```bash
   make test
   ```

### IDE Setup

#### VS Code

Recommended extensions:
- Go (golang.go)
- Go Test Explorer
- Error Lens

Recommended settings (`.vscode/settings.json`):
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.testFlags": ["-v"],
  "go.coverOnSave": true
}
```

#### GoLand

- Enable Go Modules support
- Configure test runner to use `-v` flag
- Enable code coverage on test runs

### Running Locally

#### Hardware Product (requires Linux)

```bash
# Build
make build-hardware

# Run with test config
sudo ./bin/heimdal-hardware --config config/config.json
```

#### Desktop Product

```bash
# Build for your platform
make build-desktop-macos    # or build-desktop-windows, build-desktop-linux

# Run
./bin/heimdal-desktop-darwin-amd64
```

**Note**: Desktop product requires elevated permissions for packet capture.


## Project Structure

```
heimdal/
├── cmd/                          # Entry points
│   ├── heimdal-hardware/        # Hardware product main
│   │   └── main.go
│   └── heimdal-desktop/         # Desktop product main
│       ├── main.go
│       └── platform_*.go        # Platform-specific initialization
│
├── internal/                     # Internal packages (not importable externally)
│   ├── core/                    # Shared business logic
│   │   ├── packet/              # Packet analysis
│   │   │   └── analyzer.go
│   │   ├── cloud/               # Cloud communication
│   │   │   ├── connector.go
│   │   │   ├── aws_connector.go
│   │   │   └── gcp_connector.go
│   │   ├── detection/           # Anomaly detection
│   │   │   └── detector.go
│   │   └── profiler/            # Behavioral profiling
│   │       └── profiler.go
│   │
│   ├── platform/                # Platform abstraction layer
│   │   ├── interfaces.go        # Core interfaces
│   │   ├── linux_embedded/      # Raspberry Pi implementations
│   │   │   ├── packet_capture.go
│   │   │   └── system_integrator.go
│   │   ├── desktop_windows/     # Windows implementations
│   │   │   ├── packet_capture.go
│   │   │   ├── system_integrator.go
│   │   │   └── storage.go
│   │   ├── desktop_macos/       # macOS implementations
│   │   │   ├── packet_capture.go
│   │   │   ├── system_integrator.go
│   │   │   └── storage.go
│   │   └── desktop_linux/       # Linux desktop implementations
│   │       ├── packet_capture.go
│   │       ├── system_integrator.go
│   │       └── storage.go
│   │
│   ├── hardware/                # Hardware-specific logic
│   │   ├── orchestrator/        # Hardware orchestrator
│   │   │   └── orchestrator.go
│   │   └── config/              # Hardware configuration
│   │
│   └── desktop/                 # Desktop-specific logic
│       ├── orchestrator/        # Desktop orchestrator
│       │   └── orchestrator.go
│       ├── featuregate/         # Tier management
│       │   ├── feature_gate.go
│       │   ├── license.go
│       │   └── config.go
│       ├── visualizer/          # Local dashboard server
│       │   ├── server.go
│       │   ├── api.go
│       │   └── websocket.go
│       ├── systray/             # System tray integration
│       │   ├── systray.go
│       │   ├── systray_windows.go
│       │   ├── systray_darwin.go
│       │   ├── systray_linux.go
│       │   └── notifications.go
│       └── installer/           # Installation logic
│           └── onboarding.go
│
├── web/                         # Web assets
│   └── dashboard/               # Dashboard UI
│       ├── index.html
│       ├── app.js
│       └── styles.css
│
├── test/                        # Tests
│   ├── integration/             # Integration tests
│   ├── property/                # Property-based tests
│   │   ├── generators.go        # Test data generators
│   │   └── *_test.go
│   └── mocks/                   # Mock implementations
│       ├── mock_packet_capture.go
│       ├── mock_storage.go
│       └── mock_system_integrator.go
│
├── build/                       # Build system
│   ├── package/                 # Packaging scripts
│   └── installers/              # Installer configurations
│
├── ansible/                     # Ansible deployment (hardware)
│   ├── playbook.yml
│   ├── inventory.ini
│   └── roles/
│
├── docs/                        # Documentation
│   ├── DESKTOP_USER_GUIDE.md
│   └── DEVELOPER_GUIDE.md
│
├── config/                      # Example configurations
│   └── config.json
│
├── Makefile                     # Build targets
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── README.md                    # Project overview
├── ARCHITECTURE.md              # Detailed architecture
├── BUILD.md                     # Build instructions
└── CONFIG.md                    # Configuration reference
```

### Package Organization Rules

1. **cmd/**: Only contains `main` packages and minimal initialization code
2. **internal/core/**: Pure business logic with no platform dependencies
3. **internal/platform/**: Interface definitions and platform-specific implementations
4. **internal/hardware/**: Hardware product-specific logic
5. **internal/desktop/**: Desktop product-specific logic
6. **test/**: All test code, organized by test type


## Core Interfaces

The platform abstraction layer defines three core interfaces that enable platform-specific implementations.

### PacketCaptureProvider Interface

Abstracts packet capture mechanisms across platforms.

```go
// Location: internal/platform/interfaces.go

type PacketCaptureProvider interface {
    // Open initializes packet capture on the specified interface
    Open(interfaceName string, promiscuous bool, filter string) error
    
    // ReadPacket returns the next captured packet
    ReadPacket() (*Packet, error)
    
    // Close releases packet capture resources
    Close() error
    
    // GetStats returns capture statistics
    GetStats() (*CaptureStats, error)
}

type Packet struct {
    Timestamp    time.Time
    SrcMAC       net.HardwareAddr
    DstMAC       net.HardwareAddr
    SrcIP        net.IP
    DstIP        net.IP
    SrcPort      uint16
    DstPort      uint16
    Protocol     string
    PayloadSize  uint32
    RawData      []byte
}
```

**Implementations:**
- `linux_embedded`: Raw sockets / AF_PACKET
- `desktop_windows`: gopacket with Npcap
- `desktop_macos`: gopacket with libpcap
- `desktop_linux`: gopacket with libpcap

**Usage Example:**
```go
provider := &desktop_macos.PacketCapture{}
err := provider.Open("en0", true, "tcp or udp")
if err != nil {
    return err
}
defer provider.Close()

for {
    packet, err := provider.ReadPacket()
    if err != nil {
        break
    }
    // Process packet
}
```

### SystemIntegrator Interface

Abstracts OS-level service integration.

```go
type SystemIntegrator interface {
    // Install registers the application with the OS
    Install(config *InstallConfig) error
    
    // Uninstall removes the application from OS registration
    Uninstall() error
    
    // Start begins the service/daemon
    Start() error
    
    // Stop halts the service/daemon
    Stop() error
    
    // Restart stops and starts the service/daemon
    Restart() error
    
    // GetStatus returns the current service status
    GetStatus() (*ServiceStatus, error)
    
    // EnableAutoStart configures the service to start on boot
    EnableAutoStart(enabled bool) error
}
```

**Implementations:**
- `linux_embedded`: systemd service
- `desktop_windows`: Windows Service API
- `desktop_macos`: LaunchAgent
- `desktop_linux`: systemd user service

### StorageProvider Interface

Abstracts data persistence across platforms.

```go
type StorageProvider interface {
    // Open initializes the storage backend
    Open(path string, options *StorageOptions) error
    
    // Close releases storage resources
    Close() error
    
    // Get retrieves a value by key
    Get(key string) ([]byte, error)
    
    // Set stores a value with the given key
    Set(key string, value []byte) error
    
    // Delete removes a key-value pair
    Delete(key string) error
    
    // List returns all keys matching the prefix
    List(prefix string) ([]string, error)
    
    // Batch performs multiple operations atomically
    Batch(ops []BatchOp) error
}
```

**All platforms use BadgerDB** with platform-specific paths.


## Adding Platform Implementations

This section explains how to add support for a new platform (e.g., FreeBSD, Android).

### Step 1: Create Platform Directory

Create a new directory under `internal/platform/`:

```bash
mkdir -p internal/platform/platform_name
```

### Step 2: Implement PacketCaptureProvider

Create `internal/platform/platform_name/packet_capture.go`:

```go
// +build platform_name

package platform_name

import (
    "github.com/your-org/heimdal/internal/platform"
)

type PacketCapture struct {
    // Platform-specific fields
}

func (pc *PacketCapture) Open(interfaceName string, promiscuous bool, filter string) error {
    // Implementation
}

func (pc *PacketCapture) ReadPacket() (*platform.Packet, error) {
    // Implementation
}

func (pc *PacketCapture) Close() error {
    // Implementation
}

func (pc *PacketCapture) GetStats() (*platform.CaptureStats, error) {
    // Implementation
}
```

**Key Considerations:**
- Use build tags to ensure platform-specific code only compiles on target platform
- Handle platform-specific permissions and capabilities
- Implement proper error handling and resource cleanup
- Consider performance implications of packet capture method

### Step 3: Implement SystemIntegrator

Create `internal/platform/platform_name/system_integrator.go`:

```go
// +build platform_name

package platform_name

import (
    "github.com/your-org/heimdal/internal/platform"
)

type SystemIntegrator struct {
    // Platform-specific fields
}

func (si *SystemIntegrator) Install(config *platform.InstallConfig) error {
    // Implementation
}

func (si *SystemIntegrator) Uninstall() error {
    // Implementation
}

func (si *SystemIntegrator) Start() error {
    // Implementation
}

func (si *SystemIntegrator) Stop() error {
    // Implementation
}

func (si *SystemIntegrator) Restart() error {
    // Implementation
}

func (si *SystemIntegrator) GetStatus() (*platform.ServiceStatus, error) {
    // Implementation
}

func (si *SystemIntegrator) EnableAutoStart(enabled bool) error {
    // Implementation
}
```

**Key Considerations:**
- Research platform's service management system
- Handle platform-specific installation paths
- Implement proper privilege escalation if needed
- Provide clear error messages for common issues

### Step 4: Implement StorageProvider

Create `internal/platform/platform_name/storage.go`:

```go
// +build platform_name

package platform_name

import (
    "github.com/dgraph-io/badger/v3"
    "github.com/your-org/heimdal/internal/platform"
)

type Storage struct {
    db *badger.DB
}

func (s *Storage) Open(path string, options *platform.StorageOptions) error {
    // Determine platform-specific default path if path is empty
    // Open BadgerDB
}

// Implement other methods...
```

**Key Considerations:**
- Use platform-appropriate default paths
- Handle platform-specific file permissions
- Ensure proper cleanup on Close()

### Step 5: Add Platform-Specific Entry Point

If creating a desktop variant, add platform initialization in `cmd/heimdal-desktop/`:

Create `cmd/heimdal-desktop/platform_name.go`:

```go
// +build platform_name

package main

import (
    "github.com/your-org/heimdal/internal/platform/platform_name"
)

func initPlatform() (*platformProviders, error) {
    return &platformProviders{
        packetCapture:    &platform_name.PacketCapture{},
        systemIntegrator: &platform_name.SystemIntegrator{},
        storage:          &platform_name.Storage{},
    }, nil
}
```

### Step 6: Update Build System

Add build targets to `Makefile`:

```makefile
.PHONY: build-desktop-platform_name
build-desktop-platform_name:
	GOOS=platform_os GOARCH=platform_arch go build -o bin/heimdal-desktop-platform_name cmd/heimdal-desktop/*.go
```

### Step 7: Write Tests

Create platform-specific tests:

```go
// +build platform_name

package platform_name

import (
    "testing"
)

func TestPacketCapture_Open(t *testing.T) {
    // Test implementation
}

// More tests...
```

### Step 8: Document Platform Requirements

Update documentation with:
- Platform-specific prerequisites
- Installation instructions
- Known limitations
- Troubleshooting tips

### Example: Adding FreeBSD Support

Here's a concrete example of adding FreeBSD support:

1. **Create directory**: `internal/platform/desktop_freebsd/`

2. **Implement packet capture** using BPF (Berkeley Packet Filter):
   ```go
   // Use golang.org/x/net/bpf for BPF access
   ```

3. **Implement system integrator** using rc.d:
   ```go
   // Create rc.d script in /usr/local/etc/rc.d/
   ```

4. **Use standard storage** with path `~/.local/share/heimdal/`

5. **Add build target**:
   ```makefile
   build-desktop-freebsd:
       GOOS=freebsd GOARCH=amd64 go build -o bin/heimdal-desktop-freebsd-amd64 cmd/heimdal-desktop/*.go
   ```


## Testing Strategy

Heimdal uses a comprehensive testing strategy with multiple test types.

### Test Types

#### 1. Unit Tests

**Purpose**: Test individual functions and methods in isolation

**Location**: `*_test.go` files alongside source code

**Running**:
```bash
make test-unit
# or
go test ./internal/...
```

**Example**:
```go
func TestFeatureGate_CanAccess(t *testing.T) {
    tests := []struct {
        name     string
        tier     Tier
        feature  Feature
        expected bool
    }{
        {"Free can access visibility", TierFree, FeatureNetworkVisibility, true},
        {"Free cannot access blocking", TierFree, FeatureTrafficBlocking, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            fg := &FeatureGate{currentTier: tt.tier}
            result := fg.CanAccess(tt.feature)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### 2. Property-Based Tests

**Purpose**: Verify universal properties across many generated inputs

**Location**: `test/property/`

**Framework**: gopter (https://github.com/leanovate/gopter)

**Running**:
```bash
make test-property
# or
go test ./test/property/
```

**Configuration**: Each property test runs 100 iterations minimum

**Example**:
```go
// Feature: monorepo-architecture, Property 7: Feature Gate Access Control
// Validates: Requirements 5.4
func TestProperty_FeatureGateAccessControl(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("Feature gate correctly enforces tier permissions", 
        prop.ForAll(
            func(tier Tier, feature Feature) bool {
                fg := &FeatureGate{currentTier: tier}
                canAccess := fg.CanAccess(feature)
                requiredTier := getRequiredTier(feature)
                
                // Property: can access if tier >= required tier
                return canAccess == (tier >= requiredTier)
            },
            genTier(),
            genFeature(),
        ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

**Property Test Requirements**:
- Must include comment with feature name, property number, and requirements reference
- Must use custom generators for domain-specific types
- Must run at least 100 iterations
- Must not use mocks - test real functionality

**Custom Generators** (`test/property/generators.go`):
```go
func genTier() gopter.Gen {
    return gen.OneConstOf(TierFree, TierPro, TierEnterprise)
}

func genFeature() gopter.Gen {
    return gen.OneConstOf(
        FeatureNetworkVisibility,
        FeatureTrafficBlocking,
        FeatureAdvancedFiltering,
        // ...
    )
}

func genPacket() gopter.Gen {
    return gopter.CombineGens(
        gen.Identifier(),  // MAC address
        gen.Identifier(),  // IP address
        gen.UInt16(),      // Port
        gen.OneConstOf("TCP", "UDP", "ICMP"),
    ).Map(func(values []interface{}) *Packet {
        // Construct packet from generated values
    })
}
```

#### 3. Integration Tests

**Purpose**: Test component interactions and interface implementations

**Location**: `test/integration/`

**Running**:
```bash
make test-integration
# or
go test ./test/integration/
```

**Example**:
```go
func TestPacketCaptureIntegration(t *testing.T) {
    // Create real packet capture provider
    provider := &desktop_macos.PacketCapture{}
    
    // Create analyzer that uses the provider
    analyzer := packet.NewAnalyzer(provider)
    
    // Test that they work together
    err := analyzer.Start(context.Background())
    assert.NoError(t, err)
    
    // Verify packets are captured and analyzed
    // ...
}
```

#### 4. Platform-Specific Tests

**Purpose**: Test platform-specific implementations on their target platforms

**Location**: Platform directories with build tags

**Running**:
```bash
# Must run on target platform
make test-windows   # On Windows
make test-macos     # On macOS
make test-linux     # On Linux
```

**Example**:
```go
// +build darwin

package desktop_macos

func TestMacOSPacketCapture(t *testing.T) {
    // macOS-specific test
}
```

### Test Coverage

**Goals**:
- Core modules: 70% minimum
- Platform implementations: Best effort
- Integration tests: Critical paths

**Measuring Coverage**:
```bash
make test-coverage
# Opens coverage report in browser
```

**Coverage Report**:
```bash
go test -coverprofile=coverage.out ./internal/core/...
go tool cover -html=coverage.out
```

### Mock Implementations

**Location**: `test/mocks/`

**Purpose**: Enable testing core logic without platform dependencies

**Example**:
```go
// test/mocks/mock_packet_capture.go

type MockPacketCapture struct {
    packets []*platform.Packet
    index   int
}

func (m *MockPacketCapture) Open(interfaceName string, promiscuous bool, filter string) error {
    return nil
}

func (m *MockPacketCapture) ReadPacket() (*platform.Packet, error) {
    if m.index >= len(m.packets) {
        return nil, io.EOF
    }
    packet := m.packets[m.index]
    m.index++
    return packet, nil
}

// Usage in tests:
func TestAnalyzer(t *testing.T) {
    mockCapture := &MockPacketCapture{
        packets: []*platform.Packet{
            {SrcIP: net.ParseIP("192.168.1.1"), DstIP: net.ParseIP("192.168.1.2")},
        },
    }
    
    analyzer := packet.NewAnalyzer(mockCapture)
    // Test analyzer logic
}
```

### Testing Best Practices

1. **Test Behavior, Not Implementation**: Focus on what the code does, not how
2. **Use Table-Driven Tests**: For testing multiple scenarios
3. **Avoid Test Interdependence**: Each test should be independent
4. **Clean Up Resources**: Use `defer` for cleanup, especially in integration tests
5. **Use Meaningful Names**: Test names should describe what they test
6. **Test Error Cases**: Don't just test the happy path
7. **Keep Tests Fast**: Unit tests should run in milliseconds
8. **Use Subtests**: For better organization and parallel execution

### Continuous Integration

Tests run automatically on:
- Every pull request
- Every commit to main branch
- Nightly builds

**CI Configuration** (`.github/workflows/test.yml`):
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: make test
```


## Building and Packaging

### Build System Overview

Heimdal uses Make for build automation. All build targets are defined in the root `Makefile`.

### Common Build Targets

```bash
# Build all products for all platforms
make build-all

# Build specific products
make build-hardware              # ARM64 Linux
make build-desktop-windows       # Windows amd64
make build-desktop-macos         # macOS amd64 and arm64
make build-desktop-linux         # Linux amd64
make build-desktop-all           # All desktop platforms

# Clean build artifacts
make clean

# Run tests
make test
make test-unit
make test-property
make test-integration

# Create packages
make package-windows
make package-macos
make package-linux-deb
make package-linux-rpm
```

### Cross-Compilation

Heimdal supports cross-compilation for all target platforms.

#### Prerequisites

**For ARM64 Linux (Hardware)**:
```bash
# Install ARM64 cross-compiler
sudo apt-get install gcc-aarch64-linux-gnu
```

**For Windows (from Linux/macOS)**:
```bash
# Install MinGW-w64
sudo apt-get install mingw-w64
```

**For macOS (from Linux)**:
```bash
# Install osxcross
# See: https://github.com/tpoechtrager/osxcross
```

#### Build Flags

**Hardware (Static Linking)**:
```bash
CGO_ENABLED=1 \
GOOS=linux \
GOARCH=arm64 \
CC=aarch64-linux-gnu-gcc \
go build -ldflags="-linkmode external -extldflags -static" \
  -o bin/heimdal-hardware \
  cmd/heimdal-hardware/main.go
```

**Desktop (Dynamic Linking)**:
```bash
CGO_ENABLED=1 \
GOOS=darwin \
GOARCH=amd64 \
go build -o bin/heimdal-desktop-darwin-amd64 \
  cmd/heimdal-desktop/*.go
```

### Packaging

#### Windows Installer (NSIS)

**Prerequisites**: NSIS installed

**Script**: `build/installers/windows/installer.nsi`

**Build**:
```bash
make package-windows
```

**Output**: `build/installers/windows/heimdal-desktop-windows-installer.exe`

**Includes**:
- Heimdal Desktop binary
- Npcap installer
- Uninstaller
- Start menu shortcuts
- Registry entries for auto-start

#### macOS Installer (DMG)

**Prerequisites**: macOS with `hdiutil`

**Build**:
```bash
make package-macos
```

**Output**: `build/installers/macos/heimdal-desktop-macos.dmg`

**Includes**:
- Application bundle
- Background image
- Symbolic link to Applications folder

**Code Signing** (optional):
```bash
codesign --deep --force --verify --verbose \
  --sign "Developer ID Application: Your Name" \
  Heimdal\ Desktop.app
```

#### Linux Packages

**Debian/Ubuntu (.deb)**:
```bash
make package-linux-deb
```

**Output**: `build/package/linux/heimdal-desktop-linux-amd64.deb`

**Fedora/RHEL (.rpm)**:
```bash
make package-linux-rpm
```

**Output**: `build/package/linux/heimdal-desktop-linux-amd64.rpm`

### Version Management

Version is defined in `internal/version/version.go`:

```go
package version

const (
    Version = "1.0.0"
    Commit  = "dev"
    Date    = "unknown"
)
```

**Build with version info**:
```bash
go build -ldflags="-X github.com/your-org/heimdal/internal/version.Commit=$(git rev-parse HEAD) \
  -X github.com/your-org/heimdal/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  cmd/heimdal-desktop/*.go
```

### Release Process

1. **Update Version**: Edit `internal/version/version.go`
2. **Update Changelog**: Document changes in `CHANGELOG.md`
3. **Run Tests**: `make test`
4. **Build All Platforms**: `make build-all`
5. **Create Packages**: `make package-all`
6. **Tag Release**: `git tag v1.0.0 && git push --tags`
7. **Upload Artifacts**: Upload to release page
8. **Update Documentation**: Update docs with new version info


## Contributing Guidelines

### Getting Started

1. **Fork the Repository**: Create your own fork on GitHub
2. **Clone Your Fork**: `git clone https://github.com/your-username/heimdal.git`
3. **Create a Branch**: `git checkout -b feature/your-feature-name`
4. **Make Changes**: Implement your feature or fix
5. **Test**: Run all tests and ensure they pass
6. **Commit**: Write clear, descriptive commit messages
7. **Push**: Push your branch to your fork
8. **Pull Request**: Open a PR against the main repository

### Commit Message Format

Use conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples**:
```
feat(desktop): add system tray notifications

Implement desktop notifications for new devices and anomalies.
Notifications are configurable through the settings menu.

Closes #123
```

```
fix(packet): handle nil pointer in packet parser

Add nil check before dereferencing packet data to prevent panic.

Fixes #456
```

### Pull Request Guidelines

1. **One Feature Per PR**: Keep PRs focused on a single feature or fix
2. **Write Tests**: Include tests for new functionality
3. **Update Documentation**: Update relevant docs
4. **Pass CI**: Ensure all CI checks pass
5. **Request Review**: Request review from maintainers
6. **Address Feedback**: Respond to review comments promptly

**PR Template**:
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Property tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] Tests pass locally
```

### Code Review Process

1. **Automated Checks**: CI runs tests and linters
2. **Peer Review**: At least one maintainer reviews
3. **Feedback**: Reviewer provides constructive feedback
4. **Iteration**: Author addresses feedback
5. **Approval**: Reviewer approves changes
6. **Merge**: Maintainer merges PR

### What We Look For

- **Correctness**: Does the code work as intended?
- **Tests**: Are there adequate tests?
- **Style**: Does it follow project conventions?
- **Documentation**: Is it well-documented?
- **Performance**: Are there performance implications?
- **Security**: Are there security concerns?

### Areas for Contribution

**Good First Issues**:
- Documentation improvements
- Test coverage improvements
- Bug fixes
- UI/UX enhancements

**Advanced Contributions**:
- New platform implementations
- Performance optimizations
- New features
- Architecture improvements

### Communication

- **GitHub Issues**: For bug reports and feature requests
- **Pull Requests**: For code contributions
- **Discussions**: For questions and ideas
- **Email**: For security issues (security@heimdal.io)


## Code Style and Standards

### Go Style Guide

Follow the official Go style guide and these project-specific conventions:

#### Formatting

- Use `gofmt` for formatting (enforced by CI)
- Use `goimports` for import organization
- Line length: 120 characters maximum (soft limit)

#### Naming Conventions

**Packages**:
- Lowercase, single word if possible
- No underscores or camelCase
- Examples: `packet`, `cloud`, `detection`

**Files**:
- Lowercase with underscores
- Examples: `packet_capture.go`, `system_integrator.go`

**Types**:
- PascalCase for exported types
- camelCase for unexported types
- Examples: `PacketCapture`, `featureGate`

**Functions/Methods**:
- PascalCase for exported functions
- camelCase for unexported functions
- Examples: `NewAnalyzer()`, `processPacket()`

**Variables**:
- camelCase for local variables
- PascalCase for exported variables
- Short names for short scopes: `i`, `err`, `ctx`
- Descriptive names for larger scopes: `packetCount`, `deviceMAC`

**Constants**:
- PascalCase for exported constants
- camelCase for unexported constants
- Group related constants with `const ()`

#### Comments

**Package Comments**:
```go
// Package packet provides packet analysis functionality.
// It abstracts packet capture mechanisms and provides
// protocol parsing and metadata extraction.
package packet
```

**Type Comments**:
```go
// Analyzer processes packets from any capture provider.
// It extracts metadata and forwards it to downstream consumers.
type Analyzer struct {
    // ...
}
```

**Function Comments**:
```go
// NewAnalyzer creates a new packet analyzer with the given provider.
// The analyzer must be started with Start() before it will process packets.
func NewAnalyzer(provider PacketCaptureProvider) *Analyzer {
    // ...
}
```

**Exported Items**: All exported items must have comments

#### Error Handling

**Return Errors**:
```go
// Good
func processPacket(p *Packet) error {
    if p == nil {
        return fmt.Errorf("packet is nil")
    }
    // ...
}

// Bad - don't panic in library code
func processPacket(p *Packet) {
    if p == nil {
        panic("packet is nil")
    }
    // ...
}
```

**Wrap Errors**:
```go
// Good - provides context
if err := db.Save(device); err != nil {
    return fmt.Errorf("failed to save device %s: %w", device.MAC, err)
}

// Bad - loses context
if err := db.Save(device); err != nil {
    return err
}
```

**Check Errors**:
```go
// Good
data, err := ioutil.ReadFile(path)
if err != nil {
    return err
}

// Bad - ignoring errors
data, _ := ioutil.ReadFile(path)
```

#### Concurrency

**Use Contexts**:
```go
func (a *Analyzer) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            packet, err := a.provider.ReadPacket()
            // ...
        }
    }
}
```

**Protect Shared State**:
```go
type Profiler struct {
    mu       sync.RWMutex
    profiles map[string]*Profile
}

func (p *Profiler) GetProfile(mac string) *Profile {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.profiles[mac]
}
```

**Close Channels**:
```go
// Sender closes
func producer(ch chan<- int) {
    defer close(ch)
    for i := 0; i < 10; i++ {
        ch <- i
    }
}
```

#### Interfaces

**Small Interfaces**:
```go
// Good - focused interface
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Bad - too many methods
type DataManager interface {
    Read() error
    Write() error
    Delete() error
    Update() error
    List() error
    // ...
}
```

**Accept Interfaces, Return Structs**:
```go
// Good
func NewAnalyzer(provider PacketCaptureProvider) *Analyzer {
    return &Analyzer{provider: provider}
}

// Bad
func NewAnalyzer(provider PacketCaptureProvider) PacketAnalyzer {
    return &Analyzer{provider: provider}
}
```

### Project-Specific Conventions

#### Platform Abstraction

- Core logic must depend only on interfaces, never concrete implementations
- Platform-specific code must use build tags
- All platform implementations must be in `internal/platform/`

#### Dependency Injection

- Use constructor functions that accept dependencies
- Avoid global state
- Use interfaces for dependencies

```go
// Good
func NewOrchestrator(
    packetCapture platform.PacketCaptureProvider,
    storage platform.StorageProvider,
    cloud cloud.Connector,
) *Orchestrator {
    return &Orchestrator{
        packetCapture: packetCapture,
        storage:       storage,
        cloud:         cloud,
    }
}

// Bad - hardcoded dependencies
func NewOrchestrator() *Orchestrator {
    return &Orchestrator{
        packetCapture: &linux_embedded.PacketCapture{},
        storage:       &linux_embedded.Storage{},
        cloud:         &aws.Connector{},
    }
}
```

#### Configuration

- Use struct tags for JSON marshaling
- Provide sensible defaults
- Validate configuration on load

```go
type Config struct {
    Port     int    `json:"port"`
    Interval string `json:"interval"`
}

func (c *Config) Validate() error {
    if c.Port < 1 || c.Port > 65535 {
        return fmt.Errorf("invalid port: %d", c.Port)
    }
    // ...
}
```

### Linting

**golangci-lint Configuration** (`.golangci.yml`):
```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - deadcode
    - typecheck

linters-settings:
  gofmt:
    simplify: true
  goimports:
    local-prefixes: github.com/your-org/heimdal

issues:
  exclude-use-default: false
```

**Run Linter**:
```bash
golangci-lint run
```

### Pre-Commit Hooks

Install pre-commit hooks to catch issues early:

```bash
# .git/hooks/pre-commit
#!/bin/bash

# Format code
gofmt -w .
goimports -w .

# Run linter
golangci-lint run

# Run tests
go test ./...

# Check for errors
if [ $? -ne 0 ]; then
    echo "Pre-commit checks failed"
    exit 1
fi
```


## Debugging and Troubleshooting

### Debugging Tools

#### Delve Debugger

**Installation**:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**Usage**:
```bash
# Debug a test
dlv test ./internal/core/packet

# Debug the application
dlv exec ./bin/heimdal-desktop

# Attach to running process
dlv attach $(pgrep heimdal-desktop)
```

**Common Commands**:
- `break main.main` - Set breakpoint
- `continue` - Continue execution
- `next` - Step over
- `step` - Step into
- `print variable` - Print variable value
- `goroutines` - List goroutines
- `goroutine 1` - Switch to goroutine

#### VS Code Debugging

**Configuration** (`.vscode/launch.json`):
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Desktop",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/heimdal-desktop",
            "args": ["--config", "config/config.json"],
            "env": {},
            "showLog": true
        },
        {
            "name": "Debug Test",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/internal/core/packet",
            "args": ["-test.v"]
        }
    ]
}
```

### Logging

#### Log Levels

Heimdal uses structured logging with these levels:
- `DEBUG`: Detailed information for debugging
- `INFO`: General informational messages
- `WARN`: Warning messages
- `ERROR`: Error messages
- `FATAL`: Fatal errors (application exits)

#### Enabling Debug Logging

**Environment Variable**:
```bash
export HEIMDAL_LOG_LEVEL=debug
./bin/heimdal-desktop
```

**Configuration File**:
```json
{
  "logging": {
    "level": "debug",
    "output": "stdout"
  }
}
```

#### Log Output

**Console Output**:
```
2024-01-15T10:30:45Z INFO  [orchestrator] Starting Heimdal Desktop
2024-01-15T10:30:45Z DEBUG [packet] Opening packet capture on interface en0
2024-01-15T10:30:45Z INFO  [api] Starting API server on port 8080
```

**File Output**:
- Windows: `%APPDATA%\Heimdal\logs\heimdal.log`
- macOS: `~/Library/Logs/Heimdal/heimdal.log`
- Linux: `~/.local/share/heimdal/logs/heimdal.log`

### Common Issues

#### Build Failures

**Issue**: CGO errors during cross-compilation

**Solution**:
```bash
# Ensure cross-compiler is installed
sudo apt-get install gcc-aarch64-linux-gnu

# Set CC environment variable
export CC=aarch64-linux-gnu-gcc
```

**Issue**: Missing dependencies

**Solution**:
```bash
go mod download
go mod tidy
```

#### Runtime Errors

**Issue**: Permission denied for packet capture

**Solution**:
```bash
# Linux
sudo setcap cap_net_raw,cap_net_admin=eip ./bin/heimdal-desktop

# macOS
sudo ./bin/heimdal-desktop

# Windows
# Run as Administrator
```

**Issue**: Port already in use

**Solution**:
```bash
# Find process using port 8080
lsof -i :8080  # macOS/Linux
netstat -ano | findstr :8080  # Windows

# Kill the process or change port in config
```

#### Test Failures

**Issue**: Property test fails intermittently

**Solution**:
- Check for race conditions
- Increase iteration count to reproduce
- Use `-race` flag: `go test -race`
- Check test logs for patterns

**Issue**: Integration test fails on CI but passes locally

**Solution**:
- Check for environment-specific assumptions
- Verify test cleanup is working
- Check for timing issues
- Use `t.Parallel()` carefully

### Performance Profiling

#### CPU Profiling

```bash
# Run with CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Analyze profile
go tool pprof cpu.prof
```

**pprof Commands**:
- `top` - Show top functions by CPU time
- `list functionName` - Show source code with annotations
- `web` - Generate call graph (requires graphviz)

#### Memory Profiling

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=.

# Analyze profile
go tool pprof mem.prof
```

#### Live Profiling

Add pprof endpoint to application:

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // ...
}
```

Access profiles:
- CPU: `http://localhost:6060/debug/pprof/profile?seconds=30`
- Heap: `http://localhost:6060/debug/pprof/heap`
- Goroutines: `http://localhost:6060/debug/pprof/goroutine`

### Troubleshooting Checklist

When encountering issues:

1. **Check Logs**: Review application logs for errors
2. **Verify Configuration**: Ensure config file is valid
3. **Check Permissions**: Verify required permissions are granted
4. **Test Connectivity**: Ensure network connectivity
5. **Review Recent Changes**: Check git history for recent changes
6. **Reproduce Locally**: Try to reproduce the issue locally
7. **Isolate the Problem**: Use binary search to narrow down the cause
8. **Check Dependencies**: Verify all dependencies are up to date
9. **Search Issues**: Check GitHub issues for similar problems
10. **Ask for Help**: If stuck, ask on forums or open an issue

### Getting Help

- **Documentation**: Check docs/ directory
- **GitHub Issues**: Search existing issues
- **Community Forums**: Ask questions
- **Email**: Contact maintainers

When asking for help, include:
- Operating system and version
- Go version
- Heimdal version
- Steps to reproduce
- Error messages and logs
- What you've already tried

---

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [gopter Documentation](https://github.com/leanovate/gopter)
- [BadgerDB Documentation](https://dgraph.io/docs/badger/)

## Conclusion

Thank you for contributing to Heimdal! Your contributions help make network security more accessible and effective.

For questions or suggestions about this guide, please open an issue or submit a pull request.

---

**Last Updated**: 2024-01-15
**Version**: 1.0.0
