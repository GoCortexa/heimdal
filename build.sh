#!/bin/bash
set -e

# Configuration
GOOS=linux
GOARCH=arm64
OUTPUT=ansible/roles/heimdal_sensor/files/heimdal
MODULE_PATH=./cmd/heimdal

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building Heimdal for Raspberry Pi (ARM64)...${NC}"

# Check if cross-compiler is available
if ! command -v aarch64-linux-gnu-gcc &> /dev/null; then
    echo -e "${RED}Error: aarch64-linux-gnu-gcc not found${NC}"
    echo "Please install the cross-compiler:"
    echo "  Ubuntu/Debian: sudo apt-get install gcc-aarch64-linux-gnu"
    echo "  macOS: brew install aarch64-elf-gcc"
    exit 1
fi

# Ensure output directory exists
mkdir -p "$(dirname "$OUTPUT")"

# Build with cross-compilation
echo "Compiling with CGO enabled for ARM64..."
CGO_ENABLED=1 \
CC=aarch64-linux-gnu-gcc \
GOOS=$GOOS \
GOARCH=$GOARCH \
go build -a \
  -ldflags="-s -w -extldflags '-static'" \
  -tags netgo \
  -o "$OUTPUT" \
  "$MODULE_PATH"

# Build verification
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Build complete: $OUTPUT${NC}"
    echo ""
    echo "Binary details:"
    ls -lh "$OUTPUT"
    echo ""
    echo "File type:"
    file "$OUTPUT"
    echo ""
    
    # Verify it's an ARM64 binary
    if file "$OUTPUT" | grep -q "ARM aarch64"; then
        echo -e "${GREEN}✓ Verified: ARM64 binary${NC}"
    else
        echo -e "${RED}✗ Warning: Binary may not be ARM64${NC}"
        exit 1
    fi
    
    # Check if it's statically linked
    if file "$OUTPUT" | grep -q "statically linked"; then
        echo -e "${GREEN}✓ Verified: Statically linked${NC}"
    else
        echo -e "${YELLOW}⚠ Warning: Binary may not be statically linked${NC}"
    fi
    
    echo ""
    echo -e "${GREEN}Build successful! Binary ready for deployment.${NC}"
else
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi
