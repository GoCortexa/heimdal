#!/bin/bash
# Script to build all Linux packages (DEB and RPM)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Building all Linux packages..."
echo ""

# Build DEB package
echo "=== Building DEB package ==="
"${SCRIPT_DIR}/create-deb.sh"
echo ""

# Build RPM package
echo "=== Building RPM package ==="
"${SCRIPT_DIR}/create-rpm.sh"
echo ""

echo "=== Build Summary ==="
echo ""
echo "All packages built successfully!"
echo ""
echo "DEB package:"
ls -lh "${SCRIPT_DIR}/build/output"/*.deb 2>/dev/null || echo "  (not found)"
echo ""
echo "RPM package:"
ls -lh "${SCRIPT_DIR}/build/output"/*.rpm 2>/dev/null || echo "  (not found)"
echo ""
