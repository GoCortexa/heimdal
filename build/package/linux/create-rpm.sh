#!/bin/bash
# Script to create RPM package for Heimdal Desktop

set -e

# Configuration
PACKAGE_NAME="heimdal-desktop"
VERSION="1.0.0"
RELEASE="1"
ARCHITECTURE="x86_64"
MAINTAINER="Heimdal Security <support@heimdal.io>"
SUMMARY="Network visibility and security monitoring for Linux desktops"
LICENSE="MIT"
URL="https://heimdal.io"

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../../.."
BUILD_DIR="${SCRIPT_DIR}/build"
RPM_BUILD_DIR="${BUILD_DIR}/rpmbuild"
OUTPUT_DIR="${BUILD_DIR}/output"

echo "Creating RPM package for ${PACKAGE_NAME} ${VERSION}..."

# Clean previous build
rm -rf "${BUILD_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Create RPM build directory structure
mkdir -p "${RPM_BUILD_DIR}"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
mkdir -p "${RPM_BUILD_DIR}/BUILDROOT/${PACKAGE_NAME}-${VERSION}-${RELEASE}.${ARCHITECTURE}"

# Set BUILDROOT for convenience
BUILDROOT="${RPM_BUILD_DIR}/BUILDROOT/${PACKAGE_NAME}-${VERSION}-${RELEASE}.${ARCHITECTURE}"

# Create directory structure in BUILDROOT
mkdir -p "${BUILDROOT}/opt/heimdal-desktop/bin"
mkdir -p "${BUILDROOT}/opt/heimdal-desktop/web/dashboard"
mkdir -p "${BUILDROOT}/usr/share/applications"
mkdir -p "${BUILDROOT}/usr/share/icons/hicolor/256x256/apps"
mkdir -p "${BUILDROOT}/usr/lib/systemd/user"
mkdir -p "${BUILDROOT}/etc/heimdal"

# Copy binary
echo "Copying binary..."
if [ -f "${PROJECT_ROOT}/bin/heimdal-desktop-linux-amd64" ]; then
    cp "${PROJECT_ROOT}/bin/heimdal-desktop-linux-amd64" \
       "${BUILDROOT}/opt/heimdal-desktop/bin/heimdal-desktop"
    chmod +x "${BUILDROOT}/opt/heimdal-desktop/bin/heimdal-desktop"
else
    echo "Error: Binary not found. Please build first:"
    echo "  make build-desktop-linux"
    exit 1
fi

# Copy web dashboard
echo "Copying web dashboard..."
cp -R "${PROJECT_ROOT}/web/dashboard"/* \
   "${BUILDROOT}/opt/heimdal-desktop/web/dashboard/"

# Copy default configuration
echo "Copying default configuration..."
cp "${PROJECT_ROOT}/config/config.json" \
   "${BUILDROOT}/etc/heimdal/config.json"

# Create desktop entry
echo "Creating desktop entry..."
cat > "${BUILDROOT}/usr/share/applications/${PACKAGE_NAME}.desktop" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Heimdal Desktop
Comment=${SUMMARY}
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
       "${BUILDROOT}/usr/share/icons/hicolor/256x256/apps/${PACKAGE_NAME}.png"
fi

# Create systemd user service
echo "Creating systemd service..."
cat > "${BUILDROOT}/usr/lib/systemd/user/heimdal-desktop.service" <<'EOF'
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
EOF

# Create spec file
echo "Creating RPM spec file..."
cat > "${RPM_BUILD_DIR}/SPECS/${PACKAGE_NAME}.spec" <<EOF
Name:           ${PACKAGE_NAME}
Version:        ${VERSION}
Release:        ${RELEASE}%{?dist}
Summary:        ${SUMMARY}

License:        ${LICENSE}
URL:            ${URL}
BuildArch:      ${ARCHITECTURE}

Requires:       libpcap >= 1.8.0
Requires:       libcap
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd

%description
Heimdal Desktop provides comprehensive network visibility and security
monitoring for Linux desktop systems. It monitors network traffic,
identifies devices, and detects anomalous behavior.

Features:
- Real-time network monitoring
- Device discovery and profiling
- Anomaly detection
- Web-based dashboard
- System tray integration

%prep
# No prep needed - files already in BUILDROOT

%build
# No build needed - binary is pre-built

%install
# Files already in BUILDROOT

%post
# Set capabilities for packet capture
if command -v setcap >/dev/null 2>&1; then
    setcap cap_net_raw,cap_net_admin=eip /opt/heimdal-desktop/bin/heimdal-desktop || {
        echo "Warning: Failed to set capabilities. You may need to run as root."
    }
fi

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

%preun
# Stop service if running
if command -v systemctl >/dev/null 2>&1; then
    systemctl --user stop heimdal-desktop 2>/dev/null || true
    systemctl --user disable heimdal-desktop 2>/dev/null || true
fi

# Kill any running instances
pkill -f heimdal-desktop || true

%postun
# Reload systemd after uninstall
if command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload 2>/dev/null || true
fi

if [ \$1 -eq 0 ]; then
    # Package removal (not upgrade)
    echo "User configuration preserved in ~/.config/heimdal"
fi

%files
%defattr(-,root,root,-)
/opt/heimdal-desktop/bin/heimdal-desktop
/opt/heimdal-desktop/web/dashboard/*
/usr/share/applications/${PACKAGE_NAME}.desktop
/usr/lib/systemd/user/heimdal-desktop.service
%config(noreplace) /etc/heimdal/config.json

%changelog
* $(date "+%a %b %d %Y") ${MAINTAINER} - ${VERSION}-${RELEASE}
- Initial release
- Network visibility and monitoring
- System tray integration
- Web dashboard interface
EOF

# Build the RPM
echo "Building RPM package..."
rpmbuild --define "_topdir ${RPM_BUILD_DIR}" \
         --define "_builddir %{_topdir}/BUILD" \
         --define "_buildrootdir %{_topdir}/BUILDROOT" \
         --define "_rpmdir %{_topdir}/RPMS" \
         --define "_srcrpmdir %{_topdir}/SRPMS" \
         --define "_specdir %{_topdir}/SPECS" \
         --define "_sourcedir %{_topdir}/SOURCES" \
         -bb "${RPM_BUILD_DIR}/SPECS/${PACKAGE_NAME}.spec"

# Move RPM to output directory
RPM_FILE=$(find "${RPM_BUILD_DIR}/RPMS" -name "*.rpm" -type f)
if [ -n "${RPM_FILE}" ]; then
    mv "${RPM_FILE}" "${OUTPUT_DIR}/"
    RPM_FILE="${OUTPUT_DIR}/$(basename ${RPM_FILE})"
else
    echo "Error: RPM file not found after build"
    exit 1
fi

# Calculate package info
RPM_SIZE=$(du -h "${RPM_FILE}" | cut -f1)
RPM_SHA256=$(sha256sum "${RPM_FILE}" | cut -d' ' -f1)

echo ""
echo "✓ RPM package created successfully!"
echo "  Location: ${RPM_FILE}"
echo "  Size: ${RPM_SIZE}"
echo "  SHA256: ${RPM_SHA256}"
echo ""
echo "Installation:"
echo "  sudo rpm -ivh ${RPM_FILE}"
echo ""
echo "Or with dnf/yum:"
echo "  sudo dnf install ${RPM_FILE}"
echo "  sudo yum install ${RPM_FILE}"
echo ""
