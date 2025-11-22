# macOS Installer

This directory contains scripts to create macOS installers for Heimdal Desktop.

## Installer Types

We provide two types of macOS installers:

### 1. DMG (Disk Image) - Recommended for most users
- Drag-and-drop installation
- Simple and familiar to Mac users
- No administrator password required for installation
- Created with `create-dmg.sh`

### 2. PKG (Package) - For enterprise deployment
- Standard macOS installer
- Can be deployed via MDM systems
- Runs pre/post-installation scripts
- Can be signed for Gatekeeper
- Created with `create-pkg.sh`

## Prerequisites

### Required Tools
- macOS 10.15 (Catalina) or later
- Xcode Command Line Tools: `xcode-select --install`
- Heimdal Desktop binary (built for macOS)

### Optional Tools
- **For signing**: Apple Developer certificate
  - Set `MACOS_SIGNING_IDENTITY` environment variable
  - Example: `export MACOS_SIGNING_IDENTITY="Developer ID Application: Your Name (TEAM_ID)"`

## Building Installers

### Build DMG (Recommended)

```bash
# From project root
make package-macos

# Or manually
cd build/package/macos
./create-dmg.sh
```

Output: `build/package/macos/build/output/heimdal-desktop-installer-1.0.0.dmg`

### Build PKG

```bash
# From project root
cd build/package/macos
./create-pkg.sh
```

Output: `build/package/macos/build/output/heimdal-desktop-installer-1.0.0.pkg`

### Build Both

```bash
# Build DMG
./create-dmg.sh

# Build PKG
./create-pkg.sh
```

## Application Bundle Structure

Both installers create the same application bundle:

```
Heimdal.app/
├── Contents/
│   ├── Info.plist
│   ├── PkgInfo
│   ├── MacOS/
│   │   └── heimdal-desktop
│   └── Resources/
│       ├── AppIcon.icns (optional)
│       ├── config.json
│       └── dashboard/
│           ├── index.html
│           ├── app.js
│           └── styles.css
```

## Installation Paths

After installation, files are located at:

```
/Applications/Heimdal.app/

~/Library/Application Support/Heimdal/
├── config.json
└── db/

~/Library/Logs/Heimdal/
└── heimdal.log
```

## Customization

### Change Version

Edit the version in the scripts:

```bash
VERSION="1.0.0"
```

### Add Custom Icon

1. Create an `.icns` file (macOS icon format)
2. Place it at: `build/package/macos/resources/AppIcon.icns`
3. The scripts will automatically include it

To create an `.icns` file from a PNG:

```bash
# Create iconset directory
mkdir AppIcon.iconset

# Create required sizes (example with ImageMagick)
sips -z 16 16     icon.png --out AppIcon.iconset/icon_16x16.png
sips -z 32 32     icon.png --out AppIcon.iconset/icon_16x16@2x.png
sips -z 32 32     icon.png --out AppIcon.iconset/icon_32x32.png
sips -z 64 64     icon.png --out AppIcon.iconset/icon_32x32@2x.png
sips -z 128 128   icon.png --out AppIcon.iconset/icon_128x128.png
sips -z 256 256   icon.png --out AppIcon.iconset/icon_128x128@2x.png
sips -z 256 256   icon.png --out AppIcon.iconset/icon_256x256.png
sips -z 512 512   icon.png --out AppIcon.iconset/icon_256x256@2x.png
sips -z 512 512   icon.png --out AppIcon.iconset/icon_512x512.png
sips -z 1024 1024 icon.png --out AppIcon.iconset/icon_512x512@2x.png

# Convert to icns
iconutil -c icns AppIcon.iconset
```

### Add DMG Background

1. Create a background image (PNG, 600x400 recommended)
2. Save as: `build/package/macos/resources/dmg-background.png`
3. The script will automatically use it

### Modify Bundle Identifier

Edit in the scripts:

```bash
BUNDLE_ID="io.heimdal.desktop"
```

## Code Signing

### Sign the Application

```bash
# Set your signing identity
export MACOS_SIGNING_IDENTITY="Developer ID Application: Your Name (TEAM_ID)"

# Sign the app bundle
codesign --deep --force --verify --verbose \
    --sign "${MACOS_SIGNING_IDENTITY}" \
    --options runtime \
    --entitlements entitlements.plist \
    Heimdal.app
```

### Sign the PKG

The `create-pkg.sh` script automatically signs the PKG if `MACOS_SIGNING_IDENTITY` is set:

```bash
export MACOS_SIGNING_IDENTITY="Developer ID Installer: Your Name (TEAM_ID)"
./create-pkg.sh
```

### Notarize for Gatekeeper

After signing, notarize the installer for macOS Gatekeeper:

```bash
# Submit for notarization
xcrun notarytool submit heimdal-desktop-installer-1.0.0.pkg \
    --apple-id "your@email.com" \
    --team-id "TEAM_ID" \
    --password "app-specific-password" \
    --wait

# Staple the notarization ticket
xcrun stapler staple heimdal-desktop-installer-1.0.0.pkg
```

## Testing

### Test DMG Installation

1. Build the DMG
2. Open the DMG file
3. Drag Heimdal.app to Applications
4. Launch from Applications
5. Verify permissions prompt appears
6. Grant Full Disk Access
7. Verify application functions correctly

### Test PKG Installation

1. Build the PKG
2. Double-click the PKG file
3. Follow installation wizard
4. Verify post-installation scripts run
5. Launch from Applications
6. Verify permissions and functionality

### Test on Clean System

Use a macOS VM or clean user account:

```bash
# Create test user
sudo dscl . -create /Users/testuser
sudo dscl . -create /Users/testuser UserShell /bin/bash
sudo dscl . -create /Users/testuser RealName "Test User"
sudo dscl . -create /Users/testuser UniqueID 1001
sudo dscl . -create /Users/testuser PrimaryGroupID 20
sudo dscl . -create /Users/testuser NFSHomeDirectory /Users/testuser
sudo dscl . -passwd /Users/testuser password
sudo createhomedir -c -u testuser

# Log in as test user and install
```

### Test Uninstallation

```bash
# Remove application
sudo rm -rf /Applications/Heimdal.app

# Remove user data (optional)
rm -rf ~/Library/Application\ Support/Heimdal
rm -rf ~/Library/Logs/Heimdal
```

## Troubleshooting

### "App is damaged and can't be opened"

This happens when the app is not signed or notarized. Solutions:

1. **Sign and notarize** (recommended for distribution)
2. **Remove quarantine attribute** (for testing):
   ```bash
   xattr -cr /Applications/Heimdal.app
   ```

### "Permission denied" when capturing packets

Grant Full Disk Access:
1. System Preferences → Security & Privacy → Privacy
2. Select "Full Disk Access"
3. Add Heimdal.app

### DMG window doesn't look right

The AppleScript for window positioning may fail. This is cosmetic only and doesn't affect functionality.

### PKG installation fails

Check the installer log:
```bash
cat /var/log/install.log | grep -i heimdal
```

## Universal Binary (Apple Silicon + Intel)

To create a universal binary that works on both Apple Silicon and Intel Macs:

```bash
# Build both architectures
make build-desktop-macos

# Create universal binary
lipo -create \
    bin/heimdal-desktop-macos-amd64 \
    bin/heimdal-desktop-macos-arm64 \
    -output bin/heimdal-desktop-macos-universal

# Use universal binary in installer
cp bin/heimdal-desktop-macos-universal bin/heimdal-desktop-macos-arm64
```

Then build the installer as normal.

## Requirements Validation

These installers satisfy:
- **Requirement 8.5**: Desktop installers bundle required dependencies
- **Requirement 11.1**: Graphical installer for macOS (DMG/PKG)

## Distribution

### For Public Release
1. Sign the application with Developer ID
2. Notarize with Apple
3. Distribute DMG for end users
4. Distribute PKG for enterprise/MDM

### For Testing
1. Build unsigned DMG/PKG
2. Distribute to testers
3. Instruct testers to remove quarantine: `xattr -cr Heimdal.app`

## Notes

- DMG is recommended for most users (simpler installation)
- PKG is better for enterprise deployment and MDM systems
- Both installers create the same application bundle
- libpcap is included with macOS, no additional dependencies needed
- Full Disk Access permission is required for packet capture
- The app runs as a menu bar application (LSUIElement=true)
