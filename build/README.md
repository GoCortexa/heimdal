# Heimdal Build System

This directory contains build configurations, packaging scripts, and installers for the Heimdal monorepo.

## Directory Structure

```
build/
├── README.md                    # This file
├── cross-compile-setup.sh       # Setup script for cross-compilation toolchains
├── installers/                  # Installer configurations
│   └── windows/                 # Windows installer (NSIS/WiX)
└── package/                     # Packaging scripts
    ├── linux/                   # Linux package scripts (deb/rpm)
    └── macos/                   # macOS package scripts (DMG/PKG)
```

## Cross-Compilation Setup

### Prerequisites

The Heimdal build system requires cross-compilation toolchains for building binaries for different platforms:

1. **ARM64 Linux (Hardware)**: `aarch64-linux-gnu-gcc`
2. **Windows (Desktop)**: `x86_64-w64-mingw32-gcc`
3. **macOS (Desktop)**: Xcode command line tools (when building on macOS)

### Installation

Run the setup script to install cross-compilation toolchains:

```bash
./build/cross-compile-setup.sh
```

Or install manually:

#### Ubuntu/Debian

```bash
# ARM64 Linux cross-compiler
sudo apt-get install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu

# Windows cross-compiler
sudo apt-get install gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64

# Additional dependencies
sudo apt-get install libpcap-dev
```

#### macOS

```bash
# Install Xcode command line tools
xcode-select --install

# Install cross-compilers via Homebrew
brew install mingw-w64

# For ARM64 Linux cross-compilation on macOS
brew tap messense/macos-cross-toolchains
brew install aarch64-unknown-linux-gnu
```

#### Fedora/RHEL

```bash
# ARM64 Linux cross-compiler
sudo dnf install gcc-aarch64-linux-gnu

# Windows cross-compiler
sudo dnf install mingw64-gcc mingw64-gcc-c++

# Additional dependencies
sudo dnf install libpcap-devel
```

## Build Flags

### Hardware Binary (ARM64 Linux)

The hardware binary is built with static linking to eliminate external dependencies:

```makefile
GOOS=linux GOARCH=arm64 CGO_ENABLED=1
CC=aarch64-linux-gnu-gcc
LDFLAGS="-s -w -extldflags '-static'"
```

**Flags explained:**
- `-s`: Strip symbol table
- `-w`: Strip DWARF debugging information
- `-extldflags '-static'`: Statically link C libraries
- `CGO_ENABLED=1`: Enable CGO for packet capture libraries

### Desktop Windows Binary

The Windows binary is built with GUI subsystem flags:

```makefile
GOOS=windows GOARCH=amd64 CGO_ENABLED=1
CC=x86_64-w64-mingw32-gcc
LDFLAGS="-s -w -H windowsgui"
```

**Flags explained:**
- `-H windowsgui`: Build as Windows GUI application (no console window)
- Requires Npcap at runtime (bundled in installer)

### Desktop macOS Binary

The macOS binary is built for both Intel and Apple Silicon:

```makefile
# Intel (amd64)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1
LDFLAGS="-s -w"

# Apple Silicon (arm64)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1
LDFLAGS="-s -w"
```

**Notes:**
- Requires libpcap (system-provided)
- May require code signing for distribution

### Desktop Linux Binary

The Linux binary is built for standard x86_64 systems:

```makefile
GOOS=linux GOARCH=amd64 CGO_ENABLED=1
LDFLAGS="-s -w"
```

**Notes:**
- Requires libpcap-dev at runtime
- Can be packaged as deb/rpm with dependencies

## CGO Dependencies

### libpcap

All binaries require libpcap for packet capture:

- **Hardware**: Statically linked
- **Windows**: Provided by Npcap
- **macOS**: System-provided
- **Linux**: Package dependency (libpcap-dev)

### BadgerDB

BadgerDB is a pure Go library with CGO dependencies for performance:

- Automatically handled by Go build system
- No special configuration required

## Troubleshooting

### Cross-compiler not found

If you see errors like `aarch64-linux-gnu-gcc: command not found`:

1. Verify the cross-compiler is installed: `which aarch64-linux-gnu-gcc`
2. Install the appropriate package (see Installation section)
3. Ensure the compiler is in your PATH

### CGO linking errors

If you see CGO linking errors:

1. Verify CGO is enabled: `go env CGO_ENABLED` should return `1`
2. Check that the cross-compiler is correctly set: `echo $CC`
3. For static linking, ensure static libraries are available

### macOS code signing

For macOS distribution, you may need to sign the binary:

```bash
codesign --sign "Developer ID Application: Your Name" \
  --timestamp \
  --options runtime \
  bin/heimdal-desktop-macos-amd64
```

### Windows Npcap dependency

The Windows binary requires Npcap at runtime. The installer should:

1. Check if Npcap is installed
2. If not, run the bundled Npcap installer
3. Verify installation before proceeding

## Build Verification

After building, verify the binaries:

```bash
# Check binary architecture
file bin/heimdal-hardware-arm64
# Expected: ELF 64-bit LSB executable, ARM aarch64

file bin/heimdal-desktop-windows-amd64.exe
# Expected: PE32+ executable (GUI) x86-64

file bin/heimdal-desktop-macos-amd64
# Expected: Mach-O 64-bit executable x86_64

file bin/heimdal-desktop-linux-amd64
# Expected: ELF 64-bit LSB executable, x86-64

# Check for dynamic dependencies (hardware should have minimal)
ldd bin/heimdal-hardware-arm64
# Expected: statically linked or minimal dependencies

# Check binary size (should be reasonable)
ls -lh bin/
```

## Performance Considerations

### Binary Size

Typical binary sizes:
- Hardware: 15-25 MB (statically linked)
- Desktop: 10-20 MB (dynamically linked)

To reduce binary size:
- Use `-ldflags="-s -w"` (already configured)
- Consider UPX compression for distribution (optional)

### Build Time

Typical build times on modern hardware:
- Single binary: 10-30 seconds
- All binaries: 2-5 minutes

To speed up builds:
- Use `go build -i` to install packages (cached)
- Use `GOCACHE` environment variable
- Consider distributed builds for CI/CD

## CI/CD Integration

The build system is designed for CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Setup cross-compilers
  run: ./build/cross-compile-setup.sh

- name: Build all binaries
  run: make build-all

- name: Run tests
  run: make test-coverage

- name: Upload artifacts
  uses: actions/upload-artifact@v3
  with:
    name: binaries
    path: bin/
```

## References

- [Go Cross Compilation](https://golang.org/doc/install/source#environment)
- [CGO Documentation](https://golang.org/cmd/cgo/)
- [libpcap Documentation](https://www.tcpdump.org/manpages/pcap.3pcap.html)
- [Npcap Documentation](https://npcap.com/)
