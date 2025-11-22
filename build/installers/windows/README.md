# Windows Installer

This directory contains the NSIS installer script for Heimdal Desktop on Windows.

## Prerequisites

To build the Windows installer, you need:

1. **NSIS (Nullsoft Scriptable Install System)** version 3.0 or later
   - Download from: https://nsis.sourceforge.io/Download
   - Or install via Chocolatey: `choco install nsis`

2. **Npcap Installer**
   - Download the latest Npcap installer from: https://npcap.com/#download
   - Rename it to `npcap-installer.exe`
   - Place it in this directory (`build/installers/windows/`)

3. **Heimdal Desktop Binary**
   - Build the Windows binary first: `make build-desktop-windows`
   - The binary should be at: `bin/heimdal-desktop-windows.exe`

4. **License File**
   - Ensure `LICENSE.txt` exists in the project root
   - Or create a placeholder: `echo "MIT License" > ../../LICENSE.txt`

## Building the Installer

### Using Make (Recommended)

```bash
make package-windows
```

This will:
1. Build the Windows binary if needed
2. Compile the NSIS script
3. Create the installer: `build/installers/windows/heimdal-desktop-installer-1.0.0.exe`

### Manual Build

```bash
# From project root
cd build/installers/windows

# Compile the installer
makensis heimdal-installer.nsi
```

## Installer Features

The installer includes:

### Core Components (Required)
- Heimdal Desktop executable
- Web dashboard files
- Default configuration
- Start Menu shortcuts
- Desktop shortcut

### Npcap (Required)
- Automatically installs Npcap if not present
- Configures Npcap with loopback support and WinPcap compatibility mode
- Skips installation if Npcap is already installed

### Windows Service (Optional)
- Installs Heimdal Desktop as a Windows Service
- Allows background operation without user login
- Can be started/stopped from Services management console

### Auto-start (Optional)
- Configures the service to start automatically on boot
- Ensures continuous network monitoring

## Installation Paths

The installer creates the following directory structure:

```
C:\Program Files\Heimdal\
├── heimdal-desktop.exe
├── uninstall.exe
└── web\dashboard\
    ├── index.html
    ├── app.js
    └── styles.css

%APPDATA%\Heimdal\
├── config.json
└── db\

%LOCALAPPDATA%\Heimdal\
└── logs\
```

## Uninstallation

The installer creates an uninstaller that:
- Stops and removes the Windows Service
- Removes all program files
- Removes Start Menu and Desktop shortcuts
- Optionally removes configuration and data files (user choice)
- Does NOT remove Npcap (as other applications may use it)

Users can uninstall via:
- Start Menu → Heimdal → Uninstall
- Control Panel → Programs and Features
- Settings → Apps & features

## Customization

### Changing Version

Edit the version in `heimdal-installer.nsi`:

```nsis
!define PRODUCT_VERSION "1.0.0"
```

### Changing Installation Directory

Edit the default installation directory:

```nsis
InstallDir "$PROGRAMFILES64\Heimdal"
```

### Adding Custom Icons

Replace the default NSIS icons:

```nsis
!define MUI_ICON "path\to\your\icon.ico"
!define MUI_UNICON "path\to\your\uninstall-icon.ico"
```

### Modifying Components

Edit the sections in the installer script to add/remove components.

## Testing

### Test Installation

1. Build the installer
2. Run the installer on a clean Windows 10/11 VM
3. Verify all components install correctly
4. Test the application launches
5. Test the uninstaller

### Test Upgrade

1. Install an older version
2. Run the new installer
3. Verify it detects the old version
4. Verify upgrade completes successfully

### Test Silent Installation

```cmd
heimdal-desktop-installer-1.0.0.exe /S
```

### Test Silent Uninstallation

```cmd
"C:\Program Files\Heimdal\uninstall.exe" /S
```

## Troubleshooting

### "Npcap installation failed"

- Download Npcap manually from https://npcap.com/
- Install with administrator privileges
- Ensure "WinPcap API-compatible Mode" is enabled

### "Failed to install Windows Service"

- Ensure the installer is run with administrator privileges
- Check Windows Event Viewer for service installation errors
- Try installing the service manually: `heimdal-desktop.exe --install-service`

### "Application won't start"

- Verify Npcap is installed: Check "Programs and Features"
- Run as administrator for first launch
- Check logs at: `%LOCALAPPDATA%\Heimdal\logs\`

## Requirements Validation

This installer satisfies:
- **Requirement 8.5**: Desktop installers bundle required dependencies (Npcap)
- **Requirement 11.1**: Graphical installer for Windows (NSIS)

## Notes

- The installer requires administrator privileges
- Npcap is required for packet capture functionality
- The installer is compatible with Windows 10 and later (64-bit only)
- Silent installation is supported for enterprise deployment
