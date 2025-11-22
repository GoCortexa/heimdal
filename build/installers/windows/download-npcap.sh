#!/bin/bash
# Script to download the latest Npcap installer
# This script should be run before building the Windows installer

set -e

NPCAP_VERSION="1.79"
NPCAP_URL="https://npcap.com/dist/npcap-${NPCAP_VERSION}.exe"
OUTPUT_FILE="npcap-installer.exe"

echo "Downloading Npcap ${NPCAP_VERSION}..."

# Check if file already exists
if [ -f "$OUTPUT_FILE" ]; then
    echo "Npcap installer already exists: $OUTPUT_FILE"
    read -p "Do you want to re-download? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Using existing Npcap installer"
        exit 0
    fi
    rm "$OUTPUT_FILE"
fi

# Download using curl or wget
if command -v curl &> /dev/null; then
    curl -L -o "$OUTPUT_FILE" "$NPCAP_URL"
elif command -v wget &> /dev/null; then
    wget -O "$OUTPUT_FILE" "$NPCAP_URL"
else
    echo "Error: Neither curl nor wget is available"
    echo "Please download Npcap manually from: https://npcap.com/#download"
    echo "Save it as: $OUTPUT_FILE"
    exit 1
fi

# Verify download
if [ -f "$OUTPUT_FILE" ]; then
    SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null)
    if [ "$SIZE" -gt 1000000 ]; then
        echo "Npcap installer downloaded successfully: $OUTPUT_FILE (${SIZE} bytes)"
    else
        echo "Warning: Downloaded file seems too small (${SIZE} bytes)"
        echo "Please verify the download or download manually from: https://npcap.com/#download"
        exit 1
    fi
else
    echo "Error: Download failed"
    exit 1
fi

echo ""
echo "Note: Npcap is licensed separately. Please review the license at:"
echo "https://github.com/nmap/npcap/blob/master/LICENSE"
