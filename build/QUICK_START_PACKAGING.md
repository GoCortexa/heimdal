# Quick Start: Building Installers

This is a quick reference for building Heimdal Desktop installers. For detailed documentation, see `build/PACKAGING.md`.

## Prerequisites

Install required tools for your platform:

```bash
# macOS (for building all platforms)
brew install mingw-w64          # Windows cross-compilation
xcode-select --install          # macOS tools

# Linux (Ubuntu/Debian)
sudo apt-get install -y \
    gcc-aarch64-linux-gnu \     # ARM64 cross-compilation
    gcc-mingw-w64-x86-64 \      # Windows cross-compilation
    dpkg-dev \                  # DEB packaging
    rpm                         # RPM packaging

# Windows
choco install nsis              # Windows installer
```

## Build Everything

```bash
# 1. Build all binaries
make build-desktop-all

# 2. Download Npcap (Windows only)
cd build/installers/windows
./download-npcap.sh
cd ../../..

# 3. Build Windows installer
makensis build/installers/windows/heimdal-installer.nsi

# 4. Build macOS installers
cd build/package/macos
./create-dmg.sh
./create-pkg.sh
cd ../../..

# 5. Build Linux packages
cd build/package/linux
./build-all.sh
cd ../../..
```

## Build Individual Platforms

### Windows

```bash
make build-desktop-windows
cd build/installers/windows
./download-npcap.sh
makensis heimdal-installer.nsi
```

**Output**: `heimdal-desktop-installer-1.0.0.exe`

### macOS

```bash
make build-desktop-macos
cd build/package/macos
./create-dmg.sh      # DMG installer
./create-pkg.sh      # PKG installer
```

**Outputs**:
- `build/package/macos/build/output/heimdal-desktop-installer-1.0.0.dmg`
- `build/package/macos/build/output/heimdal-desktop-installer-1.0.0.pkg`

### Linux

```bash
make build-desktop-linux
cd build/package/linux
./create-deb.sh      # Debian package
./create-rpm.sh      # RPM package
```

**Outputs**:
- `build/package/linux/build/output/heimdal-desktop_1.0.0_amd64.deb`
- `build/package/linux/build/output/heimdal-desktop-1.0.0-1.x86_64.rpm`

## Verify Setup

```bash
bash build/verify-packaging.sh
```

## Test Installation

### Windows
```cmd
heimdal-desktop-installer-1.0.0.exe
```

### macOS
```bash
# DMG: Open and drag to Applications
open heimdal-desktop-installer-1.0.0.dmg

# PKG: Double-click to install
open heimdal-desktop-installer-1.0.0.pkg
```

### Linux
```bash
# Debian/Ubuntu
sudo apt install ./heimdal-desktop_1.0.0_amd64.deb

# Fedora/RHEL
sudo dnf install heimdal-desktop-1.0.0-1.x86_64.rpm
```

## Common Issues

**Windows: "Npcap not found"**
```bash
cd build/installers/windows
./download-npcap.sh
```

**macOS: "Command not found"**
```bash
chmod +x build/package/macos/*.sh
```

**Linux: "Command not found"**
```bash
chmod +x build/package/linux/*.sh
```

**All: "Binary not found"**
```bash
make build-desktop-all
```

## Documentation

- **Full Guide**: `build/PACKAGING.md`
- **Windows**: `build/installers/windows/README.md`
- **macOS**: `build/package/macos/README.md`
- **Linux**: `build/package/linux/README.md`

## CI/CD Integration

See `build/PACKAGING.md` for GitHub Actions workflow examples.
