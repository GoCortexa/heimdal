#!/bin/bash
# Script to create Debian (.deb) package for Heimdal Desktop

set -e

# Configuration
PACKAGE_NAME="heimdal-desktop"
VERSION="1.0.0"
ARCHITECTURE="amd64"
MAINTAINER="Heimdal Security <support@heimdal.io>"
DESCRIPTION="Network visibility and security monitoring for Linux desktops"
HOMEPAGE="https://heimdal.io"

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../../.."
BUILD_DIR="${SCRIPT_DIR}/build"
DEB_DIR="${BUILD_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCHITECTURE}"
OUTPUT_DIR="${BUILD_DIR}/output"

echo "Creating Debian package for ${PACKAGE_NAME} ${VERSION}..."

# Clean previous build
rm -rf "${BUILD_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Create package directory structure
mkdir -p "${DEB_DIR}/DEBIAN"
mkdir -p "${DEB_DIR}/opt/heimdal-desktop/bin"
mkdir -p "${DEB_DIR}/opt/heimdal-desktop/web/dashboard"
mkdir -p "${DEB_DIR}/usr/share/applications"
mkdir -p "${DEB_DIR}/usr/share/icons/hicolor/256x256/apps"
mkdir -p "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}"
mkdir -p "${DEB_DIR}/etc/heimdal"

# Copy binary
echo "Copying binary..."
if [ -f "${PROJECT_ROOT}/bin/heimdal-desktop-linux-amd64" ]; then
    cp "${PROJECT_ROOT}/bin/heimdal-desktop-linux-amd64" \
       "${DEB_DIR}/opt/heimdal-desktop/bin/heimdal-desktop"
    chmod +x "${DEB_DIR}/opt/heimdal-desktop/bin/heimdal-desktop"
else
    echo "Error: Binary not found. Please build first:"
    echo "  make build-desktop-linux"
    exit 1
fi

# Copy web dashboard
echo "Copying web dashboard..."
cp -R "${PROJECT_ROOT}/web/dashboard"/* \
   "${DEB_DIR}/opt/heimdal-desktop/web/dashboard/"

# Copy default configuration
echo "Copying default configuration..."
cp "${PROJECT_ROOT}/config/config.json" \
   "${DEB_DIR}/etc/heimdal/config.json"

# Create desktop entry
echo "Creating desktop entry..."
cat > "${DEB_DIR}/usr/share/applications/${PACKAGE_NAME}.desktop" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Heimdal Desktop
Comment=${DESCRIPTION}
Exec=/opt/heimdal-desktop/bin/heimdal-desktop
Icon=heimdal-desktop
Terminal=false
Categories=Network;Security;System;
Keywords=network;security;monitoring;firewall;
StartupNotify=true
EOF

# Create icon placeholder (if icon exists)
if [ -f "${SCRIPT_DIR}/resources/icon.png" ]; then
    cp "${SCRIPT_DIR}/resources/icon.png" \
       "${DEB_DIR}/usr/share/icons/hicolor/256x256/apps/${PACKAGE_NAME}.png"
fi

# Create copyright file
echo "Creating copyright file..."
cat > "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}/copyright" <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: ${PACKAGE_NAME}
Upstream-Contact: ${MAINTAINER}
Source: ${HOMEPAGE}

Files: *
Copyright: 2024 Heimdal Security
License: MIT
 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:
 .
 The above copyright notice and this permission notice shall be included in all
 copies or substantial portions of the Software.
 .
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 SOFTWARE.
EOF

# Create changelog
echo "Creating changelog..."
cat > "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}/changelog.Debian" <<EOF
${PACKAGE_NAME} (${VERSION}) unstable; urgency=medium

  * Initial release
  * Network visibility and monitoring
  * System tray integration
  * Web dashboard interface

 -- ${MAINTAINER}  $(date -R)
EOF
gzip -9 "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}/changelog.Debian"

# Create control file
echo "Creating control file..."
cat > "${DEB_DIR}/DEBIAN/control" <<EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCHITECTURE}
Depends: libpcap0.8 (>= 1.8.0), libcap2-bin
Recommends: systemd
Suggests: network-manager
Maintainer: ${MAINTAINER}
Homepage: ${HOMEPAGE}
Description: ${DESCRIPTION}
 Heimdal Desktop provides comprehensive network visibility and security
 monitoring for Linux desktop systems. It monitors network traffic,
 identifies devices, and detects anomalous behavior.
 .
 Features:
  - Real-time network monitoring
  - Device discovery and profiling
  - Anomaly detection
  - Web-based dashboard
  - System tray integration
EOF

# Create postinst script
echo "Creating postinst script..."
cat > "${DEB_DIR}/DEBIAN/postinst" <<'EOF'
#!/bin/bash
set -e

case "$1" in
    configure)
        # Create user configuration directory template
        CONFIG_TEMPLATE="/etc/heimdal/config.json"
        
        # Set capabilities for packet capture
        if command -v setcap >/dev/null 2>&1; then
            echo "Setting packet capture capabilities..."
            setcap cap_net_raw,cap_net_admin=eip /opt/heimdal-desktop/bin/heimdal-desktop || {
                echo "Warning: Failed to set capabilities. You may need to run as root."
            }
        else
            echo "Warning: setcap not found. Install libcap2-bin to enable non-root packet capture."
        fi
        
        # Create systemd user service directory
        mkdir -p /usr/lib/systemd/user
        
        # Create systemd user service
        cat > /usr/lib/systemd/user/heimdal-desktop.service <<'SYSTEMD_EOF'
[Unit]
Description=Heimdal Desktop Network Monitor
After=network.target

[Service]
Type=simple
ExecStart=/opt/heimdal-desktop/bin/heimdal-desktop
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
SYSTEMD_EOF
        
        # Reload systemd
        if command -v systemctl >/dev/null 2>&1; then
            systemctl --user daemon-reload 2>/dev/null || true
        fi
        
        echo ""
        echo "Heimdal Desktop installed successfully!"
        echo ""
        echo "To start Heimdal Desktop:"
        echo "  1. Launch from application menu"
        echo "  2. Or run: /opt/heimdal-desktop/bin/heimdal-desktop"
        echo ""
        echo "To enable auto-start:"
        echo "  systemctl --user enable heimdal-desktop"
        echo "  systemctl --user start heimdal-desktop"
        echo ""
        echo "Configuration: ~/.config/heimdal/config.json"
        echo "Dashboard: http://localhost:8080"
        echo ""
        ;;
esac

exit 0
EOF
chmod +x "${DEB_DIR}/DEBIAN/postinst"

# Create prerm script
echo "Creating prerm script..."
cat > "${DEB_DIR}/DEBIAN/prerm" <<'EOF'
#!/bin/bash
set -e

case "$1" in
    remove|upgrade|deconfigure)
        # Stop service if running
        if command -v systemctl >/dev/null 2>&1; then
            systemctl --user stop heimdal-desktop 2>/dev/null || true
            systemctl --user disable heimdal-desktop 2>/dev/null || true
        fi
        
        # Kill any running instances
        pkill -f heimdal-desktop || true
        ;;
esac

exit 0
EOF
chmod +x "${DEB_DIR}/DEBIAN/prerm"

# Create postrm script
echo "Creating postrm script..."
cat > "${DEB_DIR}/DEBIAN/postrm" <<'EOF'
#!/bin/bash
set -e

case "$1" in
    purge)
        # Remove systemd service
        rm -f /usr/lib/systemd/user/heimdal-desktop.service
        
        # Reload systemd
        if command -v systemctl >/dev/null 2>&1; then
            systemctl --user daemon-reload 2>/dev/null || true
        fi
        
        # Note: We don't remove user data (~/.config/heimdal, ~/.local/share/heimdal)
        # as it may contain important network profiles
        echo "User configuration preserved in ~/.config/heimdal"
        ;;
esac

exit 0
EOF
chmod +x "${DEB_DIR}/DEBIAN/postrm"

# Calculate installed size
INSTALLED_SIZE=$(du -sk "${DEB_DIR}" | cut -f1)
echo "Installed-Size: ${INSTALLED_SIZE}" >> "${DEB_DIR}/DEBIAN/control"

# Build the package
echo "Building Debian package..."
dpkg-deb --build --root-owner-group "${DEB_DIR}" "${OUTPUT_DIR}"

# Calculate package info
DEB_FILE="${OUTPUT_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCHITECTURE}.deb"
DEB_SIZE=$(du -h "${DEB_FILE}" | cut -f1)
DEB_SHA256=$(sha256sum "${DEB_FILE}" | cut -d' ' -f1)

echo ""
echo "✓ Debian package created successfully!"
echo "  Location: ${DEB_FILE}"
echo "  Size: ${DEB_SIZE}"
echo "  SHA256: ${DEB_SHA256}"
echo ""
echo "Installation:"
echo "  sudo dpkg -i ${DEB_FILE}"
echo "  sudo apt-get install -f  # Install dependencies if needed"
echo ""
echo "Or add to repository and install with apt:"
echo "  sudo apt install ${PACKAGE_NAME}"
echo ""
