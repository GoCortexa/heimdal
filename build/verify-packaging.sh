#!/bin/bash
# Script to verify that all packaging files are in place

set -e

echo "Verifying Heimdal Desktop packaging setup..."
echo ""

ERRORS=0
WARNINGS=0

# Function to check file exists
check_file() {
    if [ -f "$1" ]; then
        echo "✓ $1"
    else
        echo "✗ $1 (missing)"
        ERRORS=$((ERRORS + 1))
    fi
}

# Function to check directory exists
check_dir() {
    if [ -d "$1" ]; then
        echo "✓ $1/"
    else
        echo "✗ $1/ (missing)"
        ERRORS=$((ERRORS + 1))
    fi
}

# Function to check executable
check_executable() {
    if [ -x "$1" ]; then
        echo "✓ $1 (executable)"
    else
        echo "⚠ $1 (not executable)"
        WARNINGS=$((WARNINGS + 1))
    fi
}

echo "=== Windows Installer ==="
check_dir "build/installers/windows"
check_file "build/installers/windows/heimdal-installer.nsi"
check_file "build/installers/windows/README.md"
check_file "build/installers/windows/download-npcap.sh"
check_executable "build/installers/windows/download-npcap.sh"
echo ""

echo "=== macOS Installers ==="
check_dir "build/package/macos"
check_file "build/package/macos/create-dmg.sh"
check_file "build/package/macos/create-app-bundle.sh"
check_file "build/package/macos/create-pkg.sh"
check_file "build/package/macos/README.md"
check_executable "build/package/macos/create-dmg.sh"
check_executable "build/package/macos/create-app-bundle.sh"
check_executable "build/package/macos/create-pkg.sh"
check_dir "build/package/macos/resources"
echo ""

echo "=== Linux Packages ==="
check_dir "build/package/linux"
check_file "build/package/linux/create-deb.sh"
check_file "build/package/linux/create-rpm.sh"
check_file "build/package/linux/build-all.sh"
check_file "build/package/linux/README.md"
check_executable "build/package/linux/create-deb.sh"
check_executable "build/package/linux/create-rpm.sh"
check_executable "build/package/linux/build-all.sh"
check_dir "build/package/linux/resources"
echo ""

echo "=== Documentation ==="
check_file "build/PACKAGING.md"
check_file "Makefile"
echo ""

echo "=== Binaries (optional - built separately) ==="
if [ -f "bin/heimdal-desktop-windows-amd64.exe" ]; then
    echo "✓ bin/heimdal-desktop-windows-amd64.exe"
else
    echo "⚠ bin/heimdal-desktop-windows-amd64.exe (not built yet - run: make build-desktop-windows)"
    WARNINGS=$((WARNINGS + 1))
fi

if [ -f "bin/heimdal-desktop-macos-amd64" ]; then
    echo "✓ bin/heimdal-desktop-macos-amd64"
else
    echo "⚠ bin/heimdal-desktop-macos-amd64 (not built yet - run: make build-desktop-macos)"
    WARNINGS=$((WARNINGS + 1))
fi

if [ -f "bin/heimdal-desktop-macos-arm64" ]; then
    echo "✓ bin/heimdal-desktop-macos-arm64"
else
    echo "⚠ bin/heimdal-desktop-macos-arm64 (not built yet - run: make build-desktop-macos)"
    WARNINGS=$((WARNINGS + 1))
fi

if [ -f "bin/heimdal-desktop-linux-amd64" ]; then
    echo "✓ bin/heimdal-desktop-linux-amd64"
else
    echo "⚠ bin/heimdal-desktop-linux-amd64 (not built yet - run: make build-desktop-linux)"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

echo "=== Summary ==="
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo "✓ All packaging files are in place!"
    echo ""
    echo "Next steps:"
    echo "  1. Build binaries: make build-desktop-all"
    echo "  2. Download Npcap: cd build/installers/windows && ./download-npcap.sh"
    echo "  3. Build installers using platform-specific scripts"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo "⚠ Packaging setup complete with $WARNINGS warnings"
    echo ""
    echo "Warnings are typically about missing binaries or non-executable scripts."
    echo "Build binaries with: make build-desktop-all"
    exit 0
else
    echo "✗ Packaging setup incomplete: $ERRORS errors, $WARNINGS warnings"
    echo ""
    echo "Please ensure all required files are present."
    exit 1
fi
