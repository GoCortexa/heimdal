#!/bin/bash
# Script to create macOS PKG installer for Heimdal Desktop
# This creates a standard macOS installer package

set -e

# Configuration
APP_NAME="Heimdal Desktop"
APP_BUNDLE="Heimdal.app"
PKG_NAME="heimdal-desktop-installer"
VERSION="1.0.0"
BUNDLE_ID="io.heimdal.desktop"
PKG_ID="${BUNDLE_ID}.pkg"

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../../.."
BUILD_DIR="${SCRIPT_DIR}/build"
PKG_ROOT="${BUILD_DIR}/pkg-root"
SCRIPTS_DIR="${BUILD_DIR}/scripts"
RESOURCES_DIR="${SCRIPT_DIR}/resources"

echo "Creating macOS PKG installer for ${APP_NAME} ${VERSION}..."

# Clean previous build
rm -rf "${BUILD_DIR}"
mkdir -p "${PKG_ROOT}/Applications"
mkdir -p "${SCRIPTS_DIR}"
mkdir -p "${BUILD_DIR}/output"

# Create application bundle
echo "Creating application bundle..."
"${SCRIPT_DIR}/create-app-bundle.sh"

# Copy application bundle to package root
echo "Copying application bundle..."
cp -R "${BUILD_DIR}/${APP_BUNDLE}" "${PKG_ROOT}/Applications/"

# Create postinstall script
echo "Creating postinstall script..."
cat > "${SCRIPTS_DIR}/postinstall" <<'EOF'
#!/bin/bash
# Post-installation script for Heimdal Desktop

set -e

APP_PATH="/Applications/Heimdal.app"
EXECUTABLE="${APP_PATH}/Contents/MacOS/heimdal-desktop"
USER_HOME=$(eval echo ~${USER})
CONFIG_DIR="${USER_HOME}/Library/Application Support/Heimdal"
LOG_DIR="${USER_HOME}/Library/Logs/Heimdal"

echo "Running post-installation tasks..."

# Create user configuration directory
if [ ! -d "${CONFIG_DIR}" ]; then
    mkdir -p "${CONFIG_DIR}"
    echo "Created configuration directory: ${CONFIG_DIR}"
fi

# Copy default configuration if it doesn't exist
if [ ! -f "${CONFIG_DIR}/config.json" ]; then
    cp "${APP_PATH}/Contents/Resources/config.json" "${CONFIG_DIR}/"
    echo "Installed default configuration"
fi

# Create database directory
mkdir -p "${CONFIG_DIR}/db"

# Create logs directory
mkdir -p "${LOG_DIR}"

# Set proper permissions
chown -R ${USER}:staff "${CONFIG_DIR}"
chown -R ${USER}:staff "${LOG_DIR}"

# Check for libpcap permissions
echo "Checking packet capture permissions..."
if [ -x "${EXECUTABLE}" ]; then
    # Test if we can capture packets
    if ! "${EXECUTABLE}" --check-permissions 2>/dev/null; then
        echo ""
        echo "⚠️  Packet capture permissions required"
        echo ""
        echo "Heimdal Desktop needs permission to capture network packets."
        echo "Please grant Full Disk Access in System Preferences:"
        echo "  1. Open System Preferences → Security & Privacy → Privacy"
        echo "  2. Select 'Full Disk Access' from the left sidebar"
        echo "  3. Click the lock icon and authenticate"
        echo "  4. Click '+' and add: ${APP_PATH}"
        echo ""
    fi
fi

echo "Post-installation completed successfully"
exit 0
EOF

chmod +x "${SCRIPTS_DIR}/postinstall"

# Create preinstall script (optional)
echo "Creating preinstall script..."
cat > "${SCRIPTS_DIR}/preinstall" <<'EOF'
#!/bin/bash
# Pre-installation script for Heimdal Desktop

set -e

APP_PATH="/Applications/Heimdal.app"

# Check if app is running and stop it
if pgrep -f "heimdal-desktop" > /dev/null; then
    echo "Stopping running Heimdal Desktop instance..."
    pkill -f "heimdal-desktop" || true
    sleep 2
fi

# Remove old version if exists
if [ -d "${APP_PATH}" ]; then
    echo "Removing previous installation..."
    rm -rf "${APP_PATH}"
fi

exit 0
EOF

chmod +x "${SCRIPTS_DIR}/preinstall"

# Create distribution XML for customization
echo "Creating distribution.xml..."
cat > "${BUILD_DIR}/distribution.xml" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
    <title>${APP_NAME}</title>
    <organization>${BUNDLE_ID}</organization>
    <domains enable_localSystem="true"/>
    <options customize="never" require-scripts="true" hostArchitectures="x86_64,arm64"/>
    
    <welcome file="welcome.html" mime-type="text/html"/>
    <license file="license.txt" mime-type="text/plain"/>
    <conclusion file="conclusion.html" mime-type="text/html"/>
    
    <pkg-ref id="${PKG_ID}"/>
    
    <options customize="never" require-scripts="false"/>
    
    <choices-outline>
        <line choice="default">
            <line choice="${PKG_ID}"/>
        </line>
    </choices-outline>
    
    <choice id="default"/>
    
    <choice id="${PKG_ID}" visible="false">
        <pkg-ref id="${PKG_ID}"/>
    </choice>
    
    <pkg-ref id="${PKG_ID}" version="${VERSION}" onConclusion="none">
        heimdal-desktop-component.pkg
    </pkg-ref>
</installer-gui-script>
EOF

# Create welcome message
cat > "${BUILD_DIR}/welcome.html" <<EOF
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
    </style>
</head>
<body>
    <h1>Welcome to ${APP_NAME}</h1>
    <p>This installer will install ${APP_NAME} version ${VERSION} on your Mac.</p>
    <p>${APP_NAME} provides network visibility and security monitoring for your home or office network.</p>
    <p><strong>Requirements:</strong></p>
    <ul>
        <li>macOS 10.15 (Catalina) or later</li>
        <li>Administrator privileges</li>
        <li>Full Disk Access permission (will be requested after installation)</li>
    </ul>
</body>
</html>
EOF

# Create conclusion message
cat > "${BUILD_DIR}/conclusion.html" <<EOF
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        .important { background-color: #fff3cd; padding: 10px; border-radius: 5px; }
    </style>
</head>
<body>
    <h1>Installation Complete</h1>
    <p>${APP_NAME} has been successfully installed.</p>
    
    <div class="important">
        <p><strong>Important: Grant Permissions</strong></p>
        <p>To enable packet capture, please grant Full Disk Access:</p>
        <ol>
            <li>Open System Preferences → Security & Privacy → Privacy</li>
            <li>Select "Full Disk Access" from the left sidebar</li>
            <li>Click the lock icon and authenticate</li>
            <li>Click "+" and add: /Applications/Heimdal.app</li>
        </ol>
    </div>
    
    <p>You can launch ${APP_NAME} from:</p>
    <ul>
        <li>Applications folder</li>
        <li>Spotlight (⌘ + Space, then type "Heimdal")</li>
        <li>Launchpad</li>
    </ul>
</body>
</html>
EOF

# Copy or create license file
if [ -f "${PROJECT_ROOT}/LICENSE.txt" ]; then
    cp "${PROJECT_ROOT}/LICENSE.txt" "${BUILD_DIR}/license.txt"
else
    echo "MIT License" > "${BUILD_DIR}/license.txt"
fi

# Build component package
echo "Building component package..."
pkgbuild --root "${PKG_ROOT}" \
    --identifier "${PKG_ID}" \
    --version "${VERSION}" \
    --scripts "${SCRIPTS_DIR}" \
    --install-location "/" \
    "${BUILD_DIR}/heimdal-desktop-component.pkg"

# Build product package with distribution
echo "Building product package..."
OUTPUT_PKG="${BUILD_DIR}/output/${PKG_NAME}-${VERSION}.pkg"
productbuild --distribution "${BUILD_DIR}/distribution.xml" \
    --resources "${BUILD_DIR}" \
    --package-path "${BUILD_DIR}" \
    "${OUTPUT_PKG}"

# Sign the package if signing identity is available
SIGNING_IDENTITY="${MACOS_SIGNING_IDENTITY:-}"
if [ -n "${SIGNING_IDENTITY}" ]; then
    echo "Signing package with identity: ${SIGNING_IDENTITY}"
    productsign --sign "${SIGNING_IDENTITY}" \
        "${OUTPUT_PKG}" \
        "${OUTPUT_PKG}.signed"
    mv "${OUTPUT_PKG}.signed" "${OUTPUT_PKG}"
    echo "✓ Package signed successfully"
else
    echo "⚠️  Package not signed (set MACOS_SIGNING_IDENTITY to sign)"
fi

# Calculate package size and checksum
PKG_SIZE=$(du -h "${OUTPUT_PKG}" | cut -f1)
PKG_SHA256=$(shasum -a 256 "${OUTPUT_PKG}" | cut -d' ' -f1)

echo ""
echo "✓ PKG created successfully!"
echo "  Location: ${OUTPUT_PKG}"
echo "  Size: ${PKG_SIZE}"
echo "  SHA256: ${PKG_SHA256}"
echo ""
echo "Installation instructions:"
echo "  1. Double-click the PKG file"
echo "  2. Follow the installation wizard"
echo "  3. Grant Full Disk Access when prompted"
echo ""
