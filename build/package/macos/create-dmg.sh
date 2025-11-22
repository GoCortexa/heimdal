#!/bin/bash
# Script to create macOS DMG installer for Heimdal Desktop
# This creates a drag-and-drop DMG with the application bundle

set -e

# Configuration
APP_NAME="Heimdal Desktop"
APP_BUNDLE="Heimdal.app"
DMG_NAME="heimdal-desktop-installer"
VERSION="1.0.0"
VOLUME_NAME="Heimdal Desktop ${VERSION}"
DMG_BACKGROUND="dmg-background.png"

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../../.."
BUILD_DIR="${SCRIPT_DIR}/build"
DMG_DIR="${BUILD_DIR}/dmg"
RESOURCES_DIR="${SCRIPT_DIR}/resources"

echo "Creating macOS DMG installer for ${APP_NAME} ${VERSION}..."

# Clean previous build
rm -rf "${BUILD_DIR}"
mkdir -p "${DMG_DIR}"
mkdir -p "${BUILD_DIR}/output"

# Create application bundle
echo "Creating application bundle..."
"${SCRIPT_DIR}/create-app-bundle.sh"

# Copy application bundle to DMG directory
echo "Copying application bundle..."
cp -R "${BUILD_DIR}/${APP_BUNDLE}" "${DMG_DIR}/"

# Create Applications symlink for drag-and-drop installation
echo "Creating Applications symlink..."
ln -s /Applications "${DMG_DIR}/Applications"

# Copy background image if it exists
if [ -f "${RESOURCES_DIR}/${DMG_BACKGROUND}" ]; then
    mkdir -p "${DMG_DIR}/.background"
    cp "${RESOURCES_DIR}/${DMG_BACKGROUND}" "${DMG_DIR}/.background/"
fi

# Create temporary DMG
echo "Creating temporary DMG..."
TEMP_DMG="${BUILD_DIR}/temp-${DMG_NAME}.dmg"
hdiutil create -srcfolder "${DMG_DIR}" \
    -volname "${VOLUME_NAME}" \
    -fs HFS+ \
    -fsargs "-c c=64,a=16,e=16" \
    -format UDRW \
    -size 200m \
    "${TEMP_DMG}"

# Mount the temporary DMG
echo "Mounting temporary DMG..."
MOUNT_DIR="/Volumes/${VOLUME_NAME}"
hdiutil attach "${TEMP_DMG}" -readwrite -noverify -noautoopen

# Wait for mount
sleep 2

# Set DMG window properties using AppleScript
echo "Configuring DMG window..."
if [ -f "${RESOURCES_DIR}/${DMG_BACKGROUND}" ]; then
    # With background image
    osascript <<EOF
tell application "Finder"
    tell disk "${VOLUME_NAME}"
        open
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set the bounds of container window to {100, 100, 700, 500}
        set viewOptions to the icon view options of container window
        set arrangement of viewOptions to not arranged
        set icon size of viewOptions to 128
        set background picture of viewOptions to file ".background:${DMG_BACKGROUND}"
        set position of item "${APP_BUNDLE}" of container window to {150, 200}
        set position of item "Applications" of container window to {450, 200}
        close
        open
        update without registering applications
        delay 2
    end tell
end tell
EOF
else
    # Without background image
    osascript <<EOF
tell application "Finder"
    tell disk "${VOLUME_NAME}"
        open
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set the bounds of container window to {100, 100, 600, 400}
        set viewOptions to the icon view options of container window
        set arrangement of viewOptions to not arranged
        set icon size of viewOptions to 128
        set position of item "${APP_BUNDLE}" of container window to {150, 150}
        set position of item "Applications" of container window to {450, 150}
        close
        open
        update without registering applications
        delay 2
    end tell
end tell
EOF
fi

# Unmount the temporary DMG
echo "Unmounting temporary DMG..."
hdiutil detach "${MOUNT_DIR}"

# Convert to compressed read-only DMG
echo "Creating final compressed DMG..."
OUTPUT_DMG="${BUILD_DIR}/output/${DMG_NAME}-${VERSION}.dmg"
hdiutil convert "${TEMP_DMG}" \
    -format UDZO \
    -imagekey zlib-level=9 \
    -o "${OUTPUT_DMG}"

# Clean up
rm -f "${TEMP_DMG}"

# Calculate DMG size and checksum
DMG_SIZE=$(du -h "${OUTPUT_DMG}" | cut -f1)
DMG_SHA256=$(shasum -a 256 "${OUTPUT_DMG}" | cut -d' ' -f1)

echo ""
echo "✓ DMG created successfully!"
echo "  Location: ${OUTPUT_DMG}"
echo "  Size: ${DMG_SIZE}"
echo "  SHA256: ${DMG_SHA256}"
echo ""
echo "Installation instructions:"
echo "  1. Open the DMG file"
echo "  2. Drag Heimdal.app to the Applications folder"
echo "  3. Launch from Applications or Spotlight"
echo ""
