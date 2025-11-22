# Heimdal Desktop Packaging Guide

This document provides a comprehensive guide to building and distributing installers for Heimdal Desktop across all supported platforms.

## Overview

Heimdal Desktop provides native installers for three major platforms:

- **Windows**: NSIS installer with Npcap bundling
- **macOS**: DMG (drag-and-drop) and PKG (standard installer)
- **Linux**: DEB (Debian/Ubuntu) and RPM (Fedora/RHEL) packages

## Quick Start

### Build All Packages

```bash
# Build binaries for all platforms
make build-desktop-all

# Create Windows installer (requires NSIS)
cd build/installers/windows
./download-npcap.sh  # Download Npcap first
makensis heimdal-installer.nsi

# Create macOS installers
cd build/package/macos
./create-dmg.sh      # DMG installer
./create-pkg.sh      # PKG installer

# Create Linux packages
cd build/package/linux
./create-deb.sh      # Debian package
./create-rpm.sh      # RPM package
```

## Platform-Specific Guides

### Windows Installer

**Location**: `build/installers/windows/`

**Prerequisites**:
- NSIS 3.0+ (Nullsoft Scriptable Install System)
- Npcap installer executable
- Windows binary built

**Features**:
- Installs Heimdal Desktop to Program Files
- Bundles and installs Npcap if not present
- Creates Start Menu shortcuts
- Optional Windows Service installation
- Optional auto-start configuration
- Uninstaller with data preservation option

**Build**:
```bash
# Download Npcap
cd build/installers/windows
./download-npcap.sh

# Build installer
makensis heimdal-installer.nsi
```

**Output**: `heimdal-desktop-installer-1.0.0.exe`

**Documentation**: See `build/installers/windows/README.md`

### macOS Installers

**Location**: `build/package/macos/`

**Prerequisites**:
- macOS 10.15+ (for building)
- Xcode Command Line Tools
- macOS binaries built (amd64 and arm64)

**Installer Types**:

1. **DMG (Recommended for end users)**
   - Drag-and-drop installation
   - Familiar to Mac users
   - No admin password for installation
   - Beautiful presentation with custom background

2. **PKG (For enterprise/MDM)**
   - Standard macOS installer
   - Can be deployed via MDM
   - Runs pre/post-installation scripts
   - Can be signed and notarized

**Build**:
```bash
cd build/package/macos

# Build DMG
./create-dmg.sh

# Build PKG
./create-pkg.sh
```

**Outputs**:
- `build/package/macos/build/output/heimdal-desktop-installer-1.0.0.dmg`
- `build/package/macos/build/output/heimdal-desktop-installer-1.0.0.pkg`

**Documentation**: See `build/package/macos/README.md`

### Linux Packages

**Location**: `build/package/linux/`

**Prerequisites**:
- `dpkg-deb` for DEB packages
- `rpmbuild` for RPM packages
- Linux binary built

**Package Types**:

1. **DEB (Debian/Ubuntu/Mint)**
   - For Debian-based distributions
   - Managed by apt/dpkg
   - Automatic dependency resolution

2. **RPM (Fedora/RHEL/CentOS)**
   - For Red Hat-based distributions
   - Managed by dnf/yum/rpm
   - Automatic dependency resolution

**Build**:
```bash
cd build/package/linux

# Build DEB
./create-deb.sh

# Build RPM
./create-rpm.sh

# Build both
./build-all.sh
```

**Outputs**:
- `build/package/linux/build/output/heimdal-desktop_1.0.0_amd64.deb`
- `build/package/linux/build/output/heimdal-desktop-1.0.0-1.x86_64.rpm`

**Documentation**: See `build/package/linux/README.md`

## Installation Paths

### Windows
```
C:\Program Files\Heimdal\
├── heimdal-desktop.exe
└── web\dashboard\

%APPDATA%\Heimdal\
├── config.json
└── db\

%LOCALAPPDATA%\Heimdal\
└── logs\
```

### macOS
```
/Applications/Heimdal.app/

~/Library/Application Support/Heimdal/
├── config.json
└── db/

~/Library/Logs/Heimdal/
```

### Linux
```
/opt/heimdal-desktop/
├── bin/heimdal-desktop
└── web/dashboard/

~/.config/heimdal/
└── config.json

~/.local/share/heimdal/
└── db/
```

## Dependencies

### Windows
- **Npcap** (bundled in installer)
  - Required for packet capture
  - Automatically installed if missing
  - Compatible with WinPcap API

### macOS
- **libpcap** (system-provided)
  - Included with macOS
  - Requires Full Disk Access permission

### Linux
- **libpcap** (package dependency)
  - Automatically installed by package manager
  - Version 1.8.0 or later required
- **libcap** (for capabilities)
  - Required for non-root packet capture

## Code Signing and Notarization

### Windows

Sign the installer with a code signing certificate:

```bash
# Using signtool (Windows SDK)
signtool sign /f certificate.pfx /p password /t http://timestamp.digicert.com heimdal-desktop-installer-1.0.0.exe
```

### macOS

Sign and notarize for Gatekeeper:

```bash
# Set signing identity
export MACOS_SIGNING_IDENTITY="Developer ID Application: Your Name (TEAM_ID)"

# Sign the app
codesign --deep --force --verify --verbose \
    --sign "${MACOS_SIGNING_IDENTITY}" \
    --options runtime \
    Heimdal.app

# Build and sign PKG
export MACOS_SIGNING_IDENTITY="Developer ID Installer: Your Name (TEAM_ID)"
./create-pkg.sh

# Notarize
xcrun notarytool submit heimdal-desktop-installer-1.0.0.pkg \
    --apple-id "your@email.com" \
    --team-id "TEAM_ID" \
    --password "app-specific-password" \
    --wait

# Staple notarization
xcrun stapler staple heimdal-desktop-installer-1.0.0.pkg
```

### Linux

Sign packages with GPG:

```bash
# Sign DEB
dpkg-sig --sign builder heimdal-desktop_1.0.0_amd64.deb

# Sign RPM
rpm --addsign heimdal-desktop-1.0.0-1.x86_64.rpm
```

## Distribution

### Direct Download

Host installers on a web server:

```
https://downloads.heimdal.io/
├── windows/
│   └── heimdal-desktop-installer-1.0.0.exe
├── macos/
│   ├── heimdal-desktop-installer-1.0.0.dmg
│   └── heimdal-desktop-installer-1.0.0.pkg
└── linux/
    ├── heimdal-desktop_1.0.0_amd64.deb
    └── heimdal-desktop-1.0.0-1.x86_64.rpm
```

### Package Repositories

#### APT Repository (Debian/Ubuntu)

```bash
# Create repository
mkdir -p repo/deb/pool/main
cp heimdal-desktop_1.0.0_amd64.deb repo/deb/pool/main/
cd repo/deb
dpkg-scanpackages pool /dev/null | gzip -9c > pool/Packages.gz
```

Users add repository:
```bash
echo "deb [trusted=yes] https://repo.heimdal.io/deb stable main" | \
    sudo tee /etc/apt/sources.list.d/heimdal.list
sudo apt update
sudo apt install heimdal-desktop
```

#### YUM/DNF Repository (Fedora/RHEL)

```bash
# Create repository
mkdir -p repo/rpm
cp heimdal-desktop-1.0.0-1.x86_64.rpm repo/rpm/
createrepo repo/rpm
```

Users add repository:
```bash
sudo tee /etc/yum.repos.d/heimdal.repo <<EOF
[heimdal]
name=Heimdal Desktop Repository
baseurl=https://repo.heimdal.io/rpm
enabled=1
gpgcheck=0
EOF
sudo dnf install heimdal-desktop
```

### App Stores

#### Microsoft Store (Windows)

Package as MSIX:
```bash
# Convert NSIS installer to MSIX
# Requires Desktop App Converter or manual MSIX creation
```

#### Mac App Store (macOS)

- Requires Apple Developer Program membership
- Must use App Store provisioning profile
- Submit via App Store Connect

#### Snap Store (Linux)

Create snapcraft.yaml and build snap:
```bash
snapcraft
snapcraft upload heimdal-desktop_1.0.0_amd64.snap
```

## Testing

### Test Matrix

Test each installer on:

**Windows**:
- Windows 11 (latest)
- Windows 10 (21H2, 22H2)
- Clean install and upgrade scenarios

**macOS**:
- macOS 14 (Sonoma) - Intel and Apple Silicon
- macOS 13 (Ventura) - Intel and Apple Silicon
- macOS 12 (Monterey) - Intel and Apple Silicon

**Linux**:
- Ubuntu 22.04 LTS, 20.04 LTS
- Debian 12, 11
- Fedora 39, 38
- RHEL 9, CentOS Stream 9

### Test Checklist

For each platform:

- [ ] Clean installation succeeds
- [ ] Upgrade from previous version succeeds
- [ ] Application launches successfully
- [ ] Dependencies are installed correctly
- [ ] Permissions are granted/requested properly
- [ ] Configuration is created correctly
- [ ] Dashboard is accessible
- [ ] System tray/menu bar appears
- [ ] Auto-start works (if enabled)
- [ ] Uninstallation succeeds
- [ ] Data preservation option works

## Automation

### CI/CD Pipeline

Example GitHub Actions workflow:

```yaml
name: Build Installers

on:
  push:
    tags:
      - 'v*'

jobs:
  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build Windows binary
        run: make build-desktop-windows
      - name: Download Npcap
        run: ./build/installers/windows/download-npcap.sh
      - name: Build installer
        run: makensis build/installers/windows/heimdal-installer.nsi
      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: windows-installer
          path: build/installers/windows/*.exe

  build-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build macOS binaries
        run: make build-desktop-macos
      - name: Build DMG
        run: ./build/package/macos/create-dmg.sh
      - name: Build PKG
        run: ./build/package/macos/create-pkg.sh
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: macos-installers
          path: build/package/macos/build/output/*

  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install dependencies
        run: sudo apt-get install -y dpkg-dev rpm
      - name: Build Linux binary
        run: make build-desktop-linux
      - name: Build DEB
        run: ./build/package/linux/create-deb.sh
      - name: Build RPM
        run: ./build/package/linux/create-rpm.sh
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: linux-packages
          path: build/package/linux/build/output/*
```

## Troubleshooting

### Common Issues

**Windows: "Npcap installation failed"**
- Download Npcap manually from https://npcap.com/
- Run installer with administrator privileges

**macOS: "App is damaged and can't be opened"**
- Sign and notarize the application
- Or remove quarantine: `xattr -cr /Applications/Heimdal.app`

**Linux: "libpcap not found"**
- Install libpcap: `sudo apt install libpcap0.8` or `sudo dnf install libpcap`

**All platforms: "Permission denied" for packet capture**
- Windows: Run as administrator
- macOS: Grant Full Disk Access
- Linux: Set capabilities or run with sudo

## Version Management

Update version in:
- `build/installers/windows/heimdal-installer.nsi` (PRODUCT_VERSION)
- `build/package/macos/create-dmg.sh` (VERSION)
- `build/package/macos/create-pkg.sh` (VERSION)
- `build/package/linux/create-deb.sh` (VERSION)
- `build/package/linux/create-rpm.sh` (VERSION)

Or use a version file:
```bash
echo "1.0.0" > VERSION
VERSION=$(cat VERSION)
```

## Requirements Validation

This packaging implementation satisfies:

- **Requirement 8.5**: Desktop installers bundle required dependencies
  - Windows: Npcap bundled
  - macOS: libpcap system-provided
  - Linux: Dependencies declared in package metadata

- **Requirement 11.1**: Graphical installers for desktop platforms
  - Windows: NSIS installer with GUI
  - macOS: DMG and PKG with standard macOS UI
  - Linux: Standard package managers with GUI frontends

## Resources

### Documentation
- Windows: `build/installers/windows/README.md`
- macOS: `build/package/macos/README.md`
- Linux: `build/package/linux/README.md`

### External Links
- NSIS: https://nsis.sourceforge.io/
- Npcap: https://npcap.com/
- Apple Developer: https://developer.apple.com/
- Debian Packaging: https://www.debian.org/doc/manuals/maint-guide/
- RPM Packaging: https://rpm-packaging-guide.github.io/

## Support

For packaging issues:
- Check platform-specific README files
- Review troubleshooting sections
- Check build logs for errors
- Test on clean VMs before distribution
