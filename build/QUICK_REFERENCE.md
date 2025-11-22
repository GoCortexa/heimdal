# Heimdal Build System Quick Reference

## Setup (First Time Only)

```bash
./build/cross-compile-setup.sh
```

## Common Build Commands

| Command | Description |
|---------|-------------|
| `make build-all` | Build all binaries (hardware + desktop) |
| `make build-hardware` | Build Raspberry Pi binary |
| `make build-desktop-all` | Build all desktop binaries |
| `make build-native` | Build for current platform |
| `make clean` | Remove build artifacts |

## Platform-Specific Builds

| Command | Output |
|---------|--------|
| `make build-desktop-windows` | `bin/heimdal-desktop-windows-amd64.exe` |
| `make build-desktop-macos` | `bin/heimdal-desktop-macos-{amd64,arm64}` |
| `make build-desktop-linux` | `bin/heimdal-desktop-linux-amd64` |
| `make build-hardware` | `bin/heimdal-hardware-arm64` |

## Testing Commands

| Command | Description |
|---------|-------------|
| `make test` | Run all tests |
| `make test-unit` | Run unit tests only |
| `make test-property` | Run property-based tests |
| `make test-integration` | Run integration tests |
| `make test-coverage` | Generate coverage report |

## Development Commands

| Command | Description |
|---------|-------------|
| `make fmt` | Format code |
| `make vet` | Run go vet |
| `make lint` | Run linter (requires golangci-lint) |
| `make tidy` | Tidy go.mod |
| `make deps` | Download dependencies |

## Packaging Commands

| Command | Description |
|---------|-------------|
| `make package-windows` | Create Windows installer |
| `make package-macos` | Create macOS DMG |
| `make package-linux` | Create Linux packages |

## CI/CD Commands

| Command | Description |
|---------|-------------|
| `make ci` | Run CI pipeline (fmt, vet, test) |
| `make ci-full` | Run full CI (fmt, vet, lint, test, build) |

## Help

```bash
make help
```

## Cross-Compilation Requirements

| Platform | Compiler | Install Command |
|----------|----------|-----------------|
| ARM64 Linux | `aarch64-linux-gnu-gcc` | `apt-get install gcc-aarch64-linux-gnu` |
| Windows | `x86_64-w64-mingw32-gcc` | `apt-get install gcc-mingw-w64-x86-64` |
| macOS | Xcode tools | `xcode-select --install` |

## Build Flags

### Hardware (Static Linking)
```
GOOS=linux GOARCH=arm64 CGO_ENABLED=1
CC=aarch64-linux-gnu-gcc
LDFLAGS="-s -w -extldflags '-static'"
```

### Desktop Windows (GUI)
```
GOOS=windows GOARCH=amd64 CGO_ENABLED=1
CC=x86_64-w64-mingw32-gcc
LDFLAGS="-s -w -H windowsgui"
```

### Desktop macOS
```
GOOS=darwin GOARCH={amd64,arm64} CGO_ENABLED=1
LDFLAGS="-s -w"
```

### Desktop Linux
```
GOOS=linux GOARCH=amd64 CGO_ENABLED=1
LDFLAGS="-s -w"
```

## Troubleshooting

### Cross-compiler not found
```bash
./build/cross-compile-setup.sh
```

### CGO linking errors
```bash
# Ubuntu/Debian
sudo apt-get install libpcap-dev

# macOS
xcode-select --install

# Fedora/RHEL
sudo dnf install libpcap-devel
```

### Build fails
```bash
# Clean and rebuild
make clean-all
make build-all
```

## File Locations

| Item | Location |
|------|----------|
| Binaries | `bin/` |
| Build configs | `build/` |
| Makefile | `Makefile` |
| Build guide | `BUILD.md` |
| Config docs | `build/BUILD_CONFIG.md` |

## Binary Sizes

| Binary | Typical Size |
|--------|--------------|
| Hardware (static) | 15-25 MB |
| Desktop Windows | 10-20 MB |
| Desktop macOS | 10-20 MB |
| Desktop Linux | 10-20 MB |

## Test Coverage Targets

| Module | Target |
|--------|--------|
| Core modules | 70%+ |
| Platform implementations | 60%+ |
| Overall | 65%+ |

## For More Information

- **Build Guide**: `BUILD.md`
- **Build Config**: `build/BUILD_CONFIG.md`
- **Build System**: `build/README.md`
- **Makefile**: `Makefile`
