#!/bin/bash
# Script to create macOS application bundle for Heimdal Desktop

set -e

# Configuration
APP_NAME="Heimdal Desktop"
APP_BUNDLE="Heimdal.app"
VERSION="1.0.0"
BUNDLE_ID="io.heimdal.desktop"
EXECUTABLE_NAME="heimdal-desktop"

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../../.."
BUILD_DIR="${SCRIPT_DIR}/build"
BUNDLE_DIR="${BUILD_DIR}/${APP_BUNDLE}"
CONTENTS_DIR="${BUNDLE_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

echo "Creating macOS application bundle for ${APP_NAME}..."

# Clean previous build
rm -rf "${BUNDLE_DIR}"

# Create bundle directory structure
mkdir -p "${MACOS_DIR}"
mkdir -p "${RESOURCES_DIR}"

# Detect architecture and copy appropriate binary
if [ -f "${PROJECT_ROOT}/bin/heimdal-desktop-macos-arm64" ] && [ "$(uname -m)" = "arm64" ]; then
    echo "Using ARM64 binary..."
    cp "${PROJECT_ROOT}/bin/heimdal-desktop-macos-arm64" "${MACOS_DIR}/${EXECUTABLE_NAME}"
elif [ -f "${PROJECT_ROOT}/bin/heimdal-desktop-macos-amd64" ]; then
    echo "Using AMD64 binary..."
    cp "${PROJECT_ROOT}/bin/heimdal-desktop-macos-amd64" "${MACOS_DIR}/${EXECUTABLE_NAME}"
else
    echo "Error: No macOS binary found. Please build first:"
    echo "  make build-desktop-macos"
    exit 1
fi

# Make executable
chmod +x "${MACOS_DIR}/${EXECUTABLE_NAME}"

# Copy web dashboard
echo "Copying web dashboard..."
cp -R "${PROJECT_ROOT}/web/dashboard" "${RESOURCES_DIR}/"

# Copy default configuration
echo "Copying default configuration..."
cp "${PROJECT_ROOT}/config/config.json" "${RESOURCES_DIR}/"

# Create Info.plist
echo "Creating Info.plist..."
cat > "${CONTENTS_DIR}/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>${EXECUTABLE_NAME}</string>
    <key>CFBundleIdentifier</key>
    <string>${BUNDLE_ID}</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright © 2024 Heimdal Security. All rights reserved.</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.utilities</string>
</dict>
</plist>
EOF

# Create PkgInfo
echo "Creating PkgInfo..."
echo -n "APPL????" > "${CONTENTS_DIR}/PkgInfo"

# Copy or create icon (if available)
if [ -f "${SCRIPT_DIR}/resources/AppIcon.icns" ]; then
    echo "Copying application icon..."
    cp "${SCRIPT_DIR}/resources/AppIcon.icns" "${RESOURCES_DIR}/"
else
    echo "Warning: No application icon found at ${SCRIPT_DIR}/resources/AppIcon.icns"
    echo "The app will use the default icon."
fi

echo "✓ Application bundle created: ${BUNDLE_DIR}"

# Verify bundle structure
echo ""
echo "Bundle structure:"
find "${BUNDLE_DIR}" -type f | sed "s|${BUILD_DIR}/||"
echo ""
