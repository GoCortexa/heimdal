#!/bin/bash
# Automated packaging script for all platforms
# This script builds and packages Heimdal Desktop for macOS, Windows, and Linux

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
BUILD_DIR="$PROJECT_ROOT/bin"
PACKAGE_DIR="$PROJECT_ROOT/dist"

echo "=== Heimdal Desktop Packaging ==="
echo "Version: $VERSION"
echo "Project Root: $PROJECT_ROOT"
echo ""

# Create directories
mkdir -p "$BUILD_DIR"
mkdir -p "$PACKAGE_DIR"

# Function to build for a platform
build_platform() {
    local platform=$1
    local arch=$2
    local output=$3
    
    echo "Building for $platform/$arch..."
    cd "$PROJECT_ROOT"
    
    GOOS=$platform GOARCH=$arch CGO_ENABLED=1 \
        go build -trimpath \
        -ldflags="-s -w -X main.Version=$VERSION -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
        -o "$output" \
        ./cmd/heimdal-desktop
    
    echo "✓ Built $output"
}

# Build macOS binaries
echo "=== Building macOS Binaries ==="
build_platform "darwin" "amd64" "$BUILD_DIR/heimdal-desktop-macos-amd64"
build_platform "darwin" "arm64" "$BUILD_DIR/heimdal-desktop-macos-arm64"

# Create macOS universal binary
if command -v lipo &> /dev/null; then
    echo "Creating universal binary..."
    lipo -create \
        "$BUILD_DIR/heimdal-desktop-macos-amd64" \
        "$BUILD_DIR/heimdal-desktop-macos-arm64" \
        -output "$BUILD_DIR/heimdal-desktop-macos-universal"
    echo "✓ Created universal binary"
fi

# Package macOS (if on macOS)
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "=== Packaging macOS ==="
    if [ -f "$PROJECT_ROOT/build/package/macos/create-app-bundle.sh" ]; then
        bash "$PROJECT_ROOT/build/package/macos/create-app-bundle.sh" "$VERSION"
        echo "✓ Created macOS app bundle"
    fi
fi

# Build Windows binary
echo "=== Building Windows Binary ==="
if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    CC=x86_64-w64-mingw32-gcc \
    build_platform "windows" "amd64" "$BUILD_DIR/heimdal-desktop-windows-amd64.exe"
else
    echo "⚠️  MinGW not found, skipping Windows build"
    echo "   Install with: brew install mingw-w64 (macOS) or apt-get install mingw-w64 (Linux)"
fi

# Build Linux binary
echo "=== Building Linux Binary ==="
build_platform "linux" "amd64" "$BUILD_DIR/heimdal-desktop-linux-amd64"

# Package Linux
echo "=== Packaging Linux ==="
if [ -f "$PROJECT_ROOT/build/package/linux/create-deb.sh" ]; then
    bash "$PROJECT_ROOT/build/package/linux/create-deb.sh" "$VERSION"
    echo "✓ Created Debian package"
fi

if [ -f "$PROJECT_ROOT/build/package/linux/create-rpm.sh" ]; then
    bash "$PROJECT_ROOT/build/package/linux/create-rpm.sh" "$VERSION"
    echo "✓ Created RPM package"
fi

# Create checksums
echo "=== Creating Checksums ==="
cd "$BUILD_DIR"
for file in heimdal-desktop-*; do
    if [ -f "$file" ]; then
        shasum -a 256 "$file" > "$file.sha256"
        echo "✓ Created checksum for $file"
    fi
done

echo ""
echo "=== Packaging Complete ==="
echo "Binaries: $BUILD_DIR"
echo "Packages: $PACKAGE_DIR"
echo ""
echo "Built artifacts:"
ls -lh "$BUILD_DIR" | grep heimdal-desktop

echo ""
echo "Next steps:"
echo "1. Test binaries on target platforms"
echo "2. Sign binaries (macOS: codesign, Windows: signtool)"
echo "3. Create installers (macOS: DMG, Windows: NSIS)"
echo "4. Upload to release page"

