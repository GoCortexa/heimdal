#!/bin/bash
# Heimdal Cross-Compilation Setup Script
# Installs necessary cross-compilation toolchains for building Heimdal binaries

set -e

echo "=========================================="
echo "Heimdal Cross-Compilation Setup"
echo "=========================================="
echo ""

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     PLATFORM=Linux;;
    Darwin*)    PLATFORM=Mac;;
    *)          PLATFORM="UNKNOWN:${OS}"
esac

echo "Detected platform: ${PLATFORM}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install on Ubuntu/Debian
install_debian() {
    echo "Installing cross-compilers for Debian/Ubuntu..."
    
    # Update package list
    sudo apt-get update
    
    # Install ARM64 Linux cross-compiler
    echo "Installing ARM64 Linux cross-compiler..."
    sudo apt-get install -y gcc-aarch64-linux-gnu g++-aarch64-linux-gnu
    
    # Install Windows cross-compiler
    echo "Installing Windows cross-compiler..."
    sudo apt-get install -y gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64
    
    # Install libpcap development files
    echo "Installing libpcap development files..."
    sudo apt-get install -y libpcap-dev
    
    echo "Debian/Ubuntu setup complete!"
}

# Function to install on Fedora/RHEL
install_fedora() {
    echo "Installing cross-compilers for Fedora/RHEL..."
    
    # Install ARM64 Linux cross-compiler
    echo "Installing ARM64 Linux cross-compiler..."
    sudo dnf install -y gcc-aarch64-linux-gnu
    
    # Install Windows cross-compiler
    echo "Installing Windows cross-compiler..."
    sudo dnf install -y mingw64-gcc mingw64-gcc-c++
    
    # Install libpcap development files
    echo "Installing libpcap development files..."
    sudo dnf install -y libpcap-devel
    
    echo "Fedora/RHEL setup complete!"
}

# Function to install on macOS
install_macos() {
    echo "Installing cross-compilers for macOS..."
    
    # Check if Homebrew is installed
    if ! command_exists brew; then
        echo "Error: Homebrew is not installed."
        echo "Please install Homebrew from https://brew.sh/"
        exit 1
    fi
    
    # Install Xcode command line tools
    if ! command_exists xcode-select; then
        echo "Installing Xcode command line tools..."
        xcode-select --install
        echo "Please complete the Xcode installation and run this script again."
        exit 0
    fi
    
    # Install Windows cross-compiler
    echo "Installing Windows cross-compiler..."
    brew install mingw-w64
    
    # Install ARM64 Linux cross-compiler (optional, may not work on all macOS versions)
    echo "Installing ARM64 Linux cross-compiler..."
    if ! brew tap | grep -q "messense/macos-cross-toolchains"; then
        brew tap messense/macos-cross-toolchains
    fi
    
    # Try to install ARM64 cross-compiler
    if brew list aarch64-unknown-linux-gnu &>/dev/null; then
        echo "ARM64 cross-compiler already installed"
    else
        echo "Attempting to install ARM64 cross-compiler..."
        brew install aarch64-unknown-linux-gnu || {
            echo "Warning: ARM64 cross-compiler installation failed."
            echo "You may need to build the hardware binary on a Linux system."
        }
    fi
    
    echo "macOS setup complete!"
}

# Verify installations
verify_installation() {
    echo ""
    echo "=========================================="
    echo "Verifying Installation"
    echo "=========================================="
    echo ""
    
    # Check Go
    if command_exists go; then
        echo "✓ Go: $(go version)"
    else
        echo "✗ Go: Not found"
        echo "  Please install Go from https://golang.org/dl/"
    fi
    
    # Check ARM64 cross-compiler
    if command_exists aarch64-linux-gnu-gcc; then
        echo "✓ ARM64 cross-compiler: $(aarch64-linux-gnu-gcc --version | head -n1)"
    else
        echo "✗ ARM64 cross-compiler: Not found"
    fi
    
    # Check Windows cross-compiler
    if command_exists x86_64-w64-mingw32-gcc; then
        echo "✓ Windows cross-compiler: $(x86_64-w64-mingw32-gcc --version | head -n1)"
    else
        echo "✗ Windows cross-compiler: Not found"
    fi
    
    # Check libpcap
    if [ "${PLATFORM}" = "Linux" ]; then
        if ldconfig -p | grep -q libpcap; then
            echo "✓ libpcap: Installed"
        else
            echo "✗ libpcap: Not found"
        fi
    elif [ "${PLATFORM}" = "Mac" ]; then
        if [ -f /usr/lib/libpcap.dylib ] || [ -f /usr/local/lib/libpcap.dylib ]; then
            echo "✓ libpcap: Installed"
        else
            echo "✗ libpcap: Not found"
        fi
    fi
    
    echo ""
    echo "=========================================="
    echo "Setup Summary"
    echo "=========================================="
    echo ""
    echo "You can now build Heimdal binaries using:"
    echo "  make build-all          # Build all binaries"
    echo "  make build-hardware     # Build hardware binary"
    echo "  make build-desktop-all  # Build all desktop binaries"
    echo ""
    echo "For more information, see build/README.md"
}

# Main installation logic
case "${PLATFORM}" in
    Linux)
        # Detect Linux distribution
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            case "${ID}" in
                ubuntu|debian)
                    install_debian
                    ;;
                fedora|rhel|centos)
                    install_fedora
                    ;;
                *)
                    echo "Unsupported Linux distribution: ${ID}"
                    echo "Please install cross-compilers manually."
                    echo "See build/README.md for instructions."
                    exit 1
                    ;;
            esac
        else
            echo "Cannot detect Linux distribution."
            echo "Please install cross-compilers manually."
            echo "See build/README.md for instructions."
            exit 1
        fi
        ;;
    Mac)
        install_macos
        ;;
    *)
        echo "Unsupported platform: ${PLATFORM}"
        echo "Please install cross-compilers manually."
        echo "See build/README.md for instructions."
        exit 1
        ;;
esac

# Verify installation
verify_installation

echo ""
echo "Setup complete!"
