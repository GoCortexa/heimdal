# Heimdal Build Guide

This guide explains how to build the Heimdal monorepo for both Hardware and Desktop products.

## Quick Start

### Prerequisites

1. **Go 1.21+**: Install from [golang.org](https://golang.org/dl/)
2. **Cross-compilers**: Run `./build/cross-compile-setup.sh` (see below)
3. **Make**: Usually pre-installed on Linux/macOS

### Build All Binaries

```bash
# Install cross-compilation toolchains (first time only)
./build/cross-compile-setup.sh

# Build all binaries
make build-all
```

Binaries will be in the `bin/` directory:
- `heimdal-hardware-arm64` - Hardware product (Raspberry Pi)
- `heimdal-desktop-windows-amd64.exe` - Windows desktop
- `heimdal-desktop-macos-amd64` - macOS Intel
- `heimdal-desktop-macos-arm64` - macOS Apple Silicon
- `heimdal-desktop-linux-amd64` - Linux desktop

## Build Targets

### Hardware Product

Build the Raspberry Pi sensor:

```bash
make build-hardware
```

Output: `bin/heimdal-hardware-arm64`

### Desktop Products

Build all desktop binaries:

```bash
make build-desktop-all
```

Or build for specific platforms:

```bash
make build-desktop-windows    # Windows
make build-desktop-macos      # macOS (both architectures)
make build-desktop-linux      # Linux
```

### Native Build

Build for your current platform (development):

```bash
make build-native
```

Output: `bin/heimdal`

## Testing

### Run All Tests

```bash
make test
```

### Run Specific Test Suites

```bash
make test-unit          # Unit tests only
make test-property      # Property-based tests
make test-integration   # Integration tests
```

### Platform-Specific Tests

```bash
make test-platform-windows    # Windows tests
make test-platform-macos      # macOS tests
make test-platform-linux      # Linux tests
```

### Coverage Report

```bash
make test-coverage
```

This generates `coverage.html` that you can open in a browser.

## Development

### Format Code

```bash
make fmt
```

### Run Linter

```bash
make lint
```

Note: Requires [golangci-lint](https://golangci-lint.run/usage/install/)

### Run Vet

```bash
make vet
```

### Tidy Dependencies

```bash
make tidy
```

## Cross-Compilation Setup

### Automatic Setup

Run the setup script to install all required cross-compilers:

```bash
./build/cross-compile-setup.sh
```

This script detects your OS and installs:
- ARM64 Linux cross-compiler (`aarch64-linux-gnu-gcc`)
- Windows cross-compiler (`x86_64-w64-mingw32-gcc`)
- Required libraries (libpcap)

### Manual Setup

#### Ubuntu/Debian

```bash
sudo apt-get update
sudo apt-get install -y \
  gcc-aarch64-linux-gnu \
  g++-aarch64-linux-gnu \
  gcc-mingw-w64-x86-64 \
  g++-mingw-w64-x86-64 \
  libpcap-dev
```

#### macOS

```bash
# Install Xcode command line tools
xcode-select --install

# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install cross-compilers
brew install mingw-w64
brew tap messense/macos-cross-toolchains
brew install aarch64-unknown-linux-gnu
```

#### Fedora/RHEL

```bash
sudo dnf install -y \
  gcc-aarch64-linux-gnu \
  mingw64-gcc \
  mingw64-gcc-c++ \
  libpcap-devel
```

## Packaging

### Windows Installer

```bash
make package-windows
```

Note: Requires NSIS or WiX. See `build/installers/windows/`

### macOS Package

```bash
make package-macos
```

Note: Requires `create-dmg` or `pkgbuild`. See `build/package/macos/`

### Linux Packages

```bash
make package-linux
```

Note: Requires `dpkg-deb` and `rpmbuild`. See `build/package/linux/`

## Clean

### Remove Build Artifacts

```bash
make clean
```

### Remove All Generated Files

```bash
make clean-all
```

This also clears Go caches.

## Troubleshooting

### Cross-compiler not found

**Error**: `aarch64-linux-gnu-gcc: command not found`

**Solution**: Run `./build/cross-compile-setup.sh` or install manually (see above)

### CGO linking errors

**Error**: `undefined reference to 'pcap_open_live'`

**Solution**: Install libpcap development files:
```bash
# Ubuntu/Debian
sudo apt-get install libpcap-dev

# macOS (usually pre-installed)
# Check: ls /usr/lib/libpcap.dylib

# Fedora/RHEL
sudo dnf install libpcap-devel
```

### Build fails on macOS

**Error**: `ld: library not found for -lpcap`

**Solution**: Ensure Xcode command line tools are installed:
```bash
xcode-select --install
```

### Windows build creates console window

**Issue**: Console window appears when running Windows binary

**Solution**: The Makefile already includes `-H windowsgui` flag. If you're building manually, ensure you include it:
```bash
go build -ldflags="-H windowsgui" ./cmd/heimdal-desktop
```

## Build Configuration

For detailed build configuration, see:
- `build/README.md` - Build system overview
- `build/BUILD_CONFIG.md` - Detailed build configuration
- `Makefile` - Build targets and commands

## CI/CD

The build system is designed for CI/CD pipelines. Example GitHub Actions workflow:

```yaml
name: Build and Test

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install cross-compilers
        run: ./build/cross-compile-setup.sh
      
      - name: Build all binaries
        run: make build-all
      
      - name: Run tests
        run: make test-coverage
      
      - name: Upload binaries
        uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: bin/
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Help

For a complete list of available targets:

```bash
make help
```

## Additional Resources

- [Go Build Documentation](https://golang.org/cmd/go/#hdr-Compile_packages_and_dependencies)
- [CGO Documentation](https://golang.org/cmd/cgo/)
- [Cross Compilation Guide](https://golang.org/doc/install/source#environment)
- [libpcap Documentation](https://www.tcpdump.org/manpages/pcap.3pcap.html)
