# Heimdal Build Configuration

This document describes the build configuration, CGO setup, and cross-compilation details for the Heimdal monorepo.

## Build Architecture

The Heimdal monorepo produces two distinct product lines:

1. **Hardware Product**: Raspberry Pi sensor (ARM64 Linux)
2. **Desktop Product**: Cross-platform software agent (Windows/macOS/Linux)

## CGO Configuration

### Why CGO is Required

Heimdal requires CGO for the following dependencies:

1. **libpcap**: Packet capture library (C library)
2. **BadgerDB**: Embedded database with CGO optimizations
3. **Platform-specific APIs**: System integration on Windows/macOS/Linux

### CGO Environment Variables

```bash
# Enable CGO (required for all builds)
export CGO_ENABLED=1

# Set cross-compiler (platform-specific)
export CC=<cross-compiler>

# Additional CGO flags (optional)
export CGO_CFLAGS="-O2 -g"
export CGO_LDFLAGS="-L/usr/local/lib"
```

## Platform-Specific Build Configuration

### Hardware (ARM64 Linux)

**Target**: Raspberry Pi 4 (ARM64 Linux)

**Build Configuration**:
```bash
GOOS=linux
GOARCH=arm64
CGO_ENABLED=1
CC=aarch64-linux-gnu-gcc
```

**LDFLAGS**:
```
-s                          # Strip symbol table
-w                          # Strip DWARF debugging info
-extldflags '-static'       # Static linking
```

**Static Linking**:
The hardware binary is statically linked to eliminate runtime dependencies. This ensures the binary runs on any ARM64 Linux system without requiring specific library versions.

**Dependencies**:
- libpcap (statically linked)
- glibc (statically linked)
- All Go dependencies (embedded)

**Binary Size**: ~15-25 MB (due to static linking)

**Deployment**: Single binary with no external dependencies

---

### Desktop Windows (x86_64)

**Target**: Windows 10/11 (x86_64)

**Build Configuration**:
```bash
GOOS=windows
GOARCH=amd64
CGO_ENABLED=1
CC=x86_64-w64-mingw32-gcc
```

**LDFLAGS**:
```
-s                          # Strip symbol table
-w                          # Strip DWARF debugging info
-H windowsgui               # GUI subsystem (no console)
```

**GUI Subsystem**:
The `-H windowsgui` flag builds the binary as a Windows GUI application, preventing a console window from appearing when the application starts.

**Dependencies**:
- Npcap (runtime dependency, bundled in installer)
- Windows API (system-provided)

**Binary Size**: ~10-20 MB

**Deployment**: Installer with Npcap bundled

---

### Desktop macOS (x86_64 and ARM64)

**Target**: macOS 10.15+ (Intel and Apple Silicon)

**Build Configuration (Intel)**:
```bash
GOOS=darwin
GOARCH=amd64
CGO_ENABLED=1
# CC not required when building on macOS
```

**Build Configuration (Apple Silicon)**:
```bash
GOOS=darwin
GOARCH=arm64
CGO_ENABLED=1
# CC not required when building on macOS
```

**LDFLAGS**:
```
-s                          # Strip symbol table
-w                          # Strip DWARF debugging info
```

**Dependencies**:
- libpcap (system-provided)
- macOS frameworks (system-provided)

**Binary Size**: ~10-20 MB per architecture

**Universal Binary**:
To create a universal binary (Intel + Apple Silicon):
```bash
lipo -create \
  bin/heimdal-desktop-macos-amd64 \
  bin/heimdal-desktop-macos-arm64 \
  -output bin/heimdal-desktop-macos-universal
```

**Code Signing**:
For distribution, the binary must be signed:
```bash
codesign --sign "Developer ID Application: Your Name" \
  --timestamp \
  --options runtime \
  bin/heimdal-desktop-macos-universal
```

**Deployment**: DMG or PKG installer

---

### Desktop Linux (x86_64)

**Target**: Ubuntu 20.04+, Fedora 35+, Debian 11+ (x86_64)

**Build Configuration**:
```bash
GOOS=linux
GOARCH=amd64
CGO_ENABLED=1
# CC not required when building on Linux
```

**LDFLAGS**:
```
-s                          # Strip symbol table
-w                          # Strip DWARF debugging info
```

**Dependencies**:
- libpcap-dev (package dependency)
- glibc (system-provided)

**Binary Size**: ~10-20 MB

**Deployment**: .deb or .rpm package with dependencies

---

## Cross-Compilation Matrix

| Target Platform | Build Platform | Cross-Compiler Required | Notes |
|----------------|----------------|------------------------|-------|
| ARM64 Linux | Linux x86_64 | `aarch64-linux-gnu-gcc` | Hardware product |
| ARM64 Linux | macOS | `aarch64-unknown-linux-gnu` | May not work on all macOS versions |
| Windows x86_64 | Linux | `x86_64-w64-mingw32-gcc` | Desktop product |
| Windows x86_64 | macOS | `x86_64-w64-mingw32-gcc` | Desktop product |
| macOS x86_64 | macOS | None | Native build |
| macOS ARM64 | macOS | None | Native build |
| macOS x86_64 | Linux | Not supported | Use macOS for macOS builds |
| Linux x86_64 | Linux | None | Native build |
| Linux x86_64 | macOS | Not supported | Use Linux for Linux builds |

## Build Flags Explained

### Go Build Flags

- `-trimpath`: Remove file system paths from binary (security)
- `-race`: Enable race detector (testing only)
- `-coverprofile`: Generate coverage report (testing only)

### Linker Flags (LDFLAGS)

- `-s`: Strip symbol table (reduces binary size)
- `-w`: Strip DWARF debugging information (reduces binary size)
- `-X main.Version=<version>`: Set version string at build time
- `-X main.BuildTime=<time>`: Set build time at build time
- `-extldflags '-static'`: Pass flags to external linker (static linking)
- `-H windowsgui`: Set Windows subsystem to GUI

### CGO Flags

- `CGO_ENABLED=1`: Enable CGO (required for C dependencies)
- `CGO_CFLAGS`: C compiler flags (optimization, debugging)
- `CGO_LDFLAGS`: C linker flags (library paths)

## Optimization Levels

### Development Builds

```bash
# Fast compilation, debugging enabled
go build -gcflags="all=-N -l" ./cmd/heimdal-desktop
```

### Release Builds

```bash
# Optimized, stripped, minimal size
go build -ldflags="-s -w" -trimpath ./cmd/heimdal-desktop
```

### Debug Builds

```bash
# Debugging symbols, race detector
go build -race -gcflags="all=-N -l" ./cmd/heimdal-desktop
```

## Static vs Dynamic Linking

### Static Linking (Hardware)

**Advantages**:
- No runtime dependencies
- Works on any ARM64 Linux system
- Predictable behavior

**Disadvantages**:
- Larger binary size
- Cannot use system security updates for libraries

**When to Use**: Hardware product (Raspberry Pi deployment)

### Dynamic Linking (Desktop)

**Advantages**:
- Smaller binary size
- Uses system libraries (security updates)
- Standard for desktop applications

**Disadvantages**:
- Requires runtime dependencies
- May have compatibility issues

**When to Use**: Desktop products (Windows/macOS/Linux)

## Troubleshooting

### CGO Linking Errors

**Problem**: `undefined reference to 'pcap_open_live'`

**Solution**: Ensure libpcap is installed and CGO can find it:
```bash
# Linux
sudo apt-get install libpcap-dev

# macOS
# libpcap is system-provided

# Verify
pkg-config --libs libpcap
```

### Cross-Compiler Not Found

**Problem**: `aarch64-linux-gnu-gcc: command not found`

**Solution**: Install the cross-compiler:
```bash
# Ubuntu/Debian
sudo apt-get install gcc-aarch64-linux-gnu

# macOS
brew tap messense/macos-cross-toolchains
brew install aarch64-unknown-linux-gnu
```

### Static Linking Fails

**Problem**: `cannot find -lpcap`

**Solution**: Install static libraries:
```bash
# Ubuntu/Debian
sudo apt-get install libpcap-dev:arm64

# Or build libpcap from source with static linking
```

### Windows GUI Flag Issues

**Problem**: Console window appears on Windows

**Solution**: Ensure `-H windowsgui` is in LDFLAGS:
```bash
go build -ldflags="-H windowsgui" ./cmd/heimdal-desktop
```

### macOS Code Signing

**Problem**: "App is damaged and can't be opened"

**Solution**: Sign the binary:
```bash
codesign --sign "Developer ID Application: Your Name" \
  --timestamp \
  --options runtime \
  bin/heimdal-desktop-macos-universal
```

## Performance Considerations

### Build Time

Typical build times on modern hardware (8-core CPU, 16GB RAM):

- Single binary: 10-30 seconds
- All binaries: 2-5 minutes
- With tests: 5-10 minutes

### Binary Size

Typical binary sizes:

- Hardware (static): 15-25 MB
- Desktop Windows: 10-20 MB
- Desktop macOS: 10-20 MB per architecture
- Desktop Linux: 10-20 MB

### Runtime Performance

CGO has minimal performance overhead:

- Function call overhead: ~10-50ns
- Packet processing: No measurable impact
- Database operations: Optimized by BadgerDB

## Security Considerations

### Binary Hardening

**Linux (Hardware)**:
```bash
# Enable RELRO (Relocation Read-Only)
go build -ldflags="-extldflags '-Wl,-z,relro,-z,now'" ./cmd/heimdal-hardware

# Enable stack canaries (enabled by default in Go)
```

**Windows**:
```bash
# Enable DEP (Data Execution Prevention) - enabled by default
# Enable ASLR (Address Space Layout Randomization) - enabled by default
```

**macOS**:
```bash
# Enable hardened runtime
codesign --sign "Developer ID" --options runtime --timestamp <binary>
```

### Dependency Security

- Use `go mod verify` to verify dependencies
- Use `go list -m all` to audit dependencies
- Use `govulncheck` to scan for vulnerabilities

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build

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
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: bin/
```

## References

- [Go Build Documentation](https://golang.org/cmd/go/#hdr-Compile_packages_and_dependencies)
- [CGO Documentation](https://golang.org/cmd/cgo/)
- [Cross Compilation](https://golang.org/doc/install/source#environment)
- [libpcap Documentation](https://www.tcpdump.org/manpages/pcap.3pcap.html)
- [Npcap Documentation](https://npcap.com/)
