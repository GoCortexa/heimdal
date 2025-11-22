# Linux Packages

This directory contains scripts to create Linux packages for Heimdal Desktop.

## Package Types

We provide two types of Linux packages:

### 1. DEB (Debian/Ubuntu)
- For Debian, Ubuntu, Linux Mint, Pop!_OS, and derivatives
- Uses `dpkg` and `apt` package managers
- Created with `create-deb.sh`

### 2. RPM (Red Hat/Fedora)
- For Fedora, RHEL, CentOS, openSUSE, and derivatives
- Uses `rpm`, `dnf`, and `yum` package managers
- Created with `create-rpm.sh`

## Prerequisites

### For DEB Packages
- `dpkg-deb` (usually pre-installed on Debian-based systems)
- `fakeroot` (optional, for building as non-root)

```bash
sudo apt-get install dpkg-dev fakeroot
```

### For RPM Packages
- `rpmbuild` (part of rpm-build package)

```bash
# Fedora/RHEL/CentOS
sudo dnf install rpm-build

# openSUSE
sudo zypper install rpm-build
```

### Common Requirements
- Heimdal Desktop binary (built for Linux)
- Build tools: `make`, `bash`

## Building Packages

### Build DEB Package

```bash
# From project root
make package-linux

# Or manually
cd build/package/linux
./create-deb.sh
```

Output: `build/package/linux/build/output/heimdal-desktop_1.0.0_amd64.deb`

### Build RPM Package

```bash
# From project root
cd build/package/linux
./create-rpm.sh
```

Output: `build/package/linux/build/output/heimdal-desktop-1.0.0-1.x86_64.rpm`

### Build Both

```bash
# Build DEB
./create-deb.sh

# Build RPM
./create-rpm.sh
```

## Installation

### DEB Package

```bash
# Install with dpkg
sudo dpkg -i heimdal-desktop_1.0.0_amd64.deb

# Install dependencies if needed
sudo apt-get install -f

# Or install with apt (handles dependencies automatically)
sudo apt install ./heimdal-desktop_1.0.0_amd64.deb
```

### RPM Package

```bash
# Install with rpm
sudo rpm -ivh heimdal-desktop-1.0.0-1.x86_64.rpm

# Or install with dnf (handles dependencies automatically)
sudo dnf install heimdal-desktop-1.0.0-1.x86_64.rpm

# Or with yum
sudo yum install heimdal-desktop-1.0.0-1.x86_64.rpm
```

## Installation Paths

After installation, files are located at:

```
/opt/heimdal-desktop/
├── bin/
│   └── heimdal-desktop
└── web/
    └── dashboard/
        ├── index.html
        ├── app.js
        └── styles.css

/etc/heimdal/
└── config.json

/usr/share/applications/
└── heimdal-desktop.desktop

/usr/lib/systemd/user/
└── heimdal-desktop.service

~/.config/heimdal/
└── config.json (created on first run)

~/.local/share/heimdal/
└── db/

~/.local/share/heimdal/logs/
└── heimdal.log
```

## Dependencies

### Runtime Dependencies

**DEB Package:**
- `libpcap0.8` (>= 1.8.0) - Packet capture library
- `libcap2-bin` - Capability management tools

**RPM Package:**
- `libpcap` (>= 1.8.0) - Packet capture library
- `libcap` - Capability management tools

These dependencies are automatically installed by the package manager.

## Usage

### Launch Application

```bash
# From application menu
# Look for "Heimdal Desktop" in Network or System categories

# From command line
/opt/heimdal-desktop/bin/heimdal-desktop

# Or if added to PATH
heimdal-desktop
```

### Enable Auto-Start

```bash
# Enable systemd user service
systemctl --user enable heimdal-desktop

# Start service
systemctl --user start heimdal-desktop

# Check status
systemctl --user status heimdal-desktop
```

### Access Dashboard

Open browser to: http://localhost:8080

### Configuration

Edit configuration file:
```bash
# User-specific configuration
nano ~/.config/heimdal/config.json

# System-wide default (requires sudo)
sudo nano /etc/heimdal/config.json
```

## Permissions

### Packet Capture Capabilities

The package automatically sets Linux capabilities for packet capture:

```bash
# Verify capabilities
getcap /opt/heimdal-desktop/bin/heimdal-desktop

# Should show:
# /opt/heimdal-desktop/bin/heimdal-desktop = cap_net_raw,cap_net_admin+eip
```

If capabilities are not set, you can set them manually:

```bash
sudo setcap cap_net_raw,cap_net_admin=eip /opt/heimdal-desktop/bin/heimdal-desktop
```

### Running Without Capabilities

If capabilities are not available, run with sudo:

```bash
sudo /opt/heimdal-desktop/bin/heimdal-desktop
```

## Uninstallation

### DEB Package

```bash
# Remove package but keep configuration
sudo apt-get remove heimdal-desktop

# Remove package and configuration
sudo apt-get purge heimdal-desktop
```

### RPM Package

```bash
# Remove package
sudo dnf remove heimdal-desktop

# Or with rpm
sudo rpm -e heimdal-desktop
```

### Remove User Data

User data is preserved during uninstallation. To remove:

```bash
rm -rf ~/.config/heimdal
rm -rf ~/.local/share/heimdal
```

## Customization

### Change Version

Edit the version in the scripts:

```bash
VERSION="1.0.0"
```

For RPM, also update:
```bash
RELEASE="1"
```

### Add Custom Icon

1. Create a 256x256 PNG icon
2. Save as: `build/package/linux/resources/icon.png`
3. The scripts will automatically include it

### Modify Package Metadata

Edit the configuration variables at the top of each script:
- `PACKAGE_NAME`
- `MAINTAINER`
- `DESCRIPTION` / `SUMMARY`
- `HOMEPAGE` / `URL`

### Add Additional Files

Edit the scripts to copy additional files to the BUILDROOT or DEB_DIR.

## Testing

### Test DEB Installation

```bash
# Install on clean Debian/Ubuntu VM
sudo apt install ./heimdal-desktop_1.0.0_amd64.deb

# Verify installation
dpkg -L heimdal-desktop

# Test application
heimdal-desktop --version

# Check service
systemctl --user status heimdal-desktop

# Uninstall
sudo apt-get purge heimdal-desktop
```

### Test RPM Installation

```bash
# Install on clean Fedora/RHEL VM
sudo dnf install heimdal-desktop-1.0.0-1.x86_64.rpm

# Verify installation
rpm -ql heimdal-desktop

# Test application
heimdal-desktop --version

# Check service
systemctl --user status heimdal-desktop

# Uninstall
sudo dnf remove heimdal-desktop
```

### Test on Multiple Distributions

Test on:
- **Debian-based**: Ubuntu 22.04, Debian 12, Linux Mint 21
- **RPM-based**: Fedora 39, RHEL 9, openSUSE Leap 15

### Verify Package Quality

```bash
# Check DEB package
lintian heimdal-desktop_1.0.0_amd64.deb

# Check RPM package
rpmlint heimdal-desktop-1.0.0-1.x86_64.rpm
```

## Repository Distribution

### Create APT Repository

```bash
# Create repository structure
mkdir -p repo/deb/pool/main
cp heimdal-desktop_1.0.0_amd64.deb repo/deb/pool/main/

# Generate Packages file
cd repo/deb
dpkg-scanpackages pool /dev/null | gzip -9c > pool/Packages.gz

# Create Release file
cat > Release <<EOF
Origin: Heimdal
Label: Heimdal Desktop
Suite: stable
Codename: stable
Architectures: amd64
Components: main
Description: Heimdal Desktop Repository
EOF
```

Users can then add the repository:
```bash
echo "deb [trusted=yes] https://repo.heimdal.io/deb stable main" | \
    sudo tee /etc/apt/sources.list.d/heimdal.list
sudo apt-get update
sudo apt-get install heimdal-desktop
```

### Create YUM/DNF Repository

```bash
# Create repository structure
mkdir -p repo/rpm
cp heimdal-desktop-1.0.0-1.x86_64.rpm repo/rpm/

# Generate repository metadata
createrepo repo/rpm
```

Users can then add the repository:
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

## Troubleshooting

### "libpcap not found"

Install libpcap:
```bash
# Debian/Ubuntu
sudo apt-get install libpcap0.8

# Fedora/RHEL
sudo dnf install libpcap
```

### "Permission denied" when capturing packets

Set capabilities:
```bash
sudo setcap cap_net_raw,cap_net_admin=eip /opt/heimdal-desktop/bin/heimdal-desktop
```

Or run with sudo:
```bash
sudo heimdal-desktop
```

### Service won't start

Check logs:
```bash
journalctl --user -u heimdal-desktop -f
```

Check status:
```bash
systemctl --user status heimdal-desktop
```

### Desktop entry doesn't appear

Update desktop database:
```bash
update-desktop-database ~/.local/share/applications
```

### Package conflicts

Remove old version first:
```bash
# DEB
sudo apt-get remove heimdal-desktop

# RPM
sudo dnf remove heimdal-desktop
```

## Requirements Validation

These packages satisfy:
- **Requirement 8.5**: Desktop installers bundle required dependencies
- **Requirement 11.1**: Installation packages for Linux

## Distribution Support

### Tested Distributions

**DEB Package:**
- Ubuntu 22.04 LTS (Jammy)
- Ubuntu 20.04 LTS (Focal)
- Debian 12 (Bookworm)
- Debian 11 (Bullseye)
- Linux Mint 21
- Pop!_OS 22.04

**RPM Package:**
- Fedora 39
- Fedora 38
- RHEL 9
- CentOS Stream 9
- openSUSE Leap 15.5

### Minimum Requirements

- Linux kernel 4.15+
- systemd (for service management)
- libpcap 1.8.0+
- 64-bit x86 architecture (amd64/x86_64)

## Notes

- Packages install to `/opt/heimdal-desktop` to avoid conflicts
- User configuration is stored in `~/.config/heimdal`
- System-wide default configuration is in `/etc/heimdal`
- Capabilities are set automatically for non-root packet capture
- systemd user service is provided for auto-start
- Desktop entry is created for GUI launchers
- User data is preserved during uninstallation
