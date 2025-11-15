# Build Instructions

## Prerequisites

### Cross-Compiler Installation

The Heimdal sensor is built for ARM64 (Raspberry Pi) architecture. You need a cross-compiler installed on your build machine.

#### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install gcc-aarch64-linux-gnu
```

#### macOS
```bash
brew tap messense/macos-cross-toolchains
brew install aarch64-unknown-linux-gnu
```

Alternatively, you can use Docker to build in a Linux environment:
```bash
docker run --rm -v "$PWD":/workspace -w /workspace golang:1.21 bash -c "
  apt-get update && apt-get install -y gcc-aarch64-linux-gnu && 
  ./build.sh
"
```

### Go Version

Ensure you have Go 1.21 or later installed:
```bash
go version
```

## Building

### Standard Build

To build the Heimdal sensor for ARM64:

```bash
./build.sh
```

This will:
1. Cross-compile the Go binary for ARM64 (Raspberry Pi)
2. Enable CGO for native library support
3. Statically link all dependencies
4. Output the binary to `ansible/roles/heimdal_sensor/files/heimdal`
5. Verify the binary is correctly built

### Build Output

The build script will display:
- Binary size
- File type verification (ARM64)
- Static linking verification
- Build success/failure status

Example output:
```
Building Heimdal for Raspberry Pi (ARM64)...
Compiling with CGO enabled for ARM64...

Build complete: ansible/roles/heimdal_sensor/files/heimdal

Binary details:
-rwxr-xr-x  1 user  staff  15M Nov 16 01:00 ansible/roles/heimdal_sensor/files/heimdal

File type:
ansible/roles/heimdal_sensor/files/heimdal: ELF 64-bit LSB executable, ARM aarch64, statically linked

✓ Verified: ARM64 binary
✓ Verified: Statically linked

Build successful! Binary ready for deployment.
```

## Troubleshooting

### Cross-Compiler Not Found

If you see:
```
Error: aarch64-linux-gnu-gcc not found
```

Install the cross-compiler using the instructions above.

### CGO Errors

If you encounter CGO-related errors, ensure:
1. The cross-compiler is in your PATH
2. CGO_ENABLED=1 is set (the build script does this)
3. You have the necessary development libraries

### Static Linking Issues

If the binary is not statically linked:
1. Ensure you're using the correct linker flags
2. Check that libpcap-dev is available for the target architecture
3. Consider using a Docker build environment

## Manual Build

If you need to customize the build process:

```bash
CGO_ENABLED=1 \
CC=aarch64-linux-gnu-gcc \
GOOS=linux \
GOARCH=arm64 \
go build -a \
  -ldflags="-s -w -extldflags '-static'" \
  -tags netgo \
  -o ansible/roles/heimdal_sensor/files/heimdal \
  ./cmd/heimdal
```

## Next Steps

After building:
1. The binary is ready at `ansible/roles/heimdal_sensor/files/heimdal`
2. Deploy using Ansible: `cd ansible && ansible-playbook -i inventory.ini playbook.yml`
3. The binary will be copied to the Raspberry Pi and configured as a systemd service
