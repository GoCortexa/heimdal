# Heimdal Network Security Platform

Heimdal is a network security platform offering two product lines:

1. **Heimdal Hardware**: Zero-touch network security sensor for Raspberry Pi
2. **Heimdal Desktop**: Cross-platform software agent for Windows, macOS, and Linux

Both products share core functionality for network discovery, traffic analysis, behavioral profiling, and anomaly detection, while maintaining platform-specific optimizations through a unified monorepo architecture.

## Features

### Shared Features (Both Products)
- **Automated Device Discovery**: Continuous scanning using ARP and mDNS protocols
- **Traffic Interception**: ARP spoofing to intercept and analyze network traffic
- **Behavioral Profiling**: Build profiles of device communication patterns
- **Anomaly Detection**: Identify unusual communication patterns and potential threats
- **Local Web Dashboard**: Monitor your network through a browser
- **Embedded Database**: All data stored locally using BadgerDB
- **Cloud Connectivity**: Optional integration with AWS IoT and Google Cloud
- **Single Binary**: Statically compiled with no external runtime dependencies

### Hardware-Specific Features
- **Zero-Touch Provisioning**: Simply plug in power and Ethernet - no manual configuration required
- **Router-Level Protection**: Protects entire subnet from a dedicated device
- **Ansible Deployment**: Automated provisioning and updates

### Desktop-Specific Features
- **System Tray Integration**: Native OS integration with menu bar/system tray
- **Desktop Notifications**: Real-time alerts for new devices and anomalies
- **Feature Tiers**: Free tier for network visibility, Pro tier for advanced features
- **Auto-Start**: Optional automatic startup on system boot
- **Cross-Platform**: Runs on Windows, macOS, and Linux

## Quick Start

### Heimdal Hardware (Raspberry Pi)

#### Prerequisites
- Raspberry Pi 4 (or compatible ARM64 device)
- Raspberry Pi OS (64-bit) or similar Linux distribution
- Network connection (Ethernet recommended)
- Ansible control machine for deployment

#### Deployment

1. **Configure Ansible Inventory**

   Edit `ansible/inventory.ini` with your Raspberry Pi details:
   ```ini
   [heimdal_sensors]
   heimdal-sensor-01 ansible_host=10.100.102.131 ansible_user=cortexa ansible_password=Password1
   ```

2. **Build the Binary**

   On your development machine (requires Go 1.21+ and ARM64 cross-compiler):
   ```bash
   make build-hardware
   ```

3. **Deploy with Ansible**

   ```bash
   cd ansible
   ansible-playbook -i inventory.ini playbook.yml
   ```

4. **Access the Dashboard**

   Open your browser to `http://<raspberry-pi-ip>:8080`

#### Manual Installation

If you prefer manual installation without Ansible:

1. Copy the binary to `/opt/heimdal/bin/heimdal-hardware`
2. Create configuration at `/etc/heimdal/config.json` (see `config/config.json` for template)
3. Create directories: `/var/lib/heimdal`, `/var/log/heimdal`
4. Enable IP forwarding: `sudo sysctl -w net.ipv4.ip_forward=1`
5. Set capabilities: `sudo setcap cap_net_raw,cap_net_admin=eip /opt/heimdal/bin/heimdal-hardware`
6. Run: `/opt/heimdal/bin/heimdal-hardware --config /etc/heimdal/config.json`

### Heimdal Desktop

#### Windows

1. **Download the Installer**
   - Download `heimdal-desktop-windows-installer.exe` from releases
   - The installer includes Npcap (required for packet capture)

2. **Run the Installer**
   - Right-click the installer and select "Run as Administrator"
   - Follow the installation wizard
   - Grant administrator permissions when prompted

3. **First Launch**
   - Heimdal Desktop will appear in your system tray
   - Complete the onboarding wizard to configure network monitoring
   - Access the dashboard by clicking "Open Dashboard" from the system tray menu

4. **Access the Dashboard**
   - Open your browser to `http://localhost:8080`
   - Or click the system tray icon and select "Open Dashboard"

**Note**: Windows requires Npcap for packet capture. The installer will prompt you to install it if not already present.

#### macOS

1. **Download the Installer**
   - Download `heimdal-desktop-macos.dmg` from releases

2. **Install the Application**
   - Open the DMG file
   - Drag Heimdal Desktop to your Applications folder
   - Launch Heimdal Desktop from Applications

3. **Grant Permissions**
   - macOS will prompt for packet capture permissions
   - Go to System Preferences → Security & Privacy → Privacy
   - Grant Full Disk Access to Heimdal Desktop
   - You may need to run with sudo on first launch: `sudo /Applications/Heimdal\ Desktop.app/Contents/MacOS/heimdal-desktop`

4. **First Launch**
   - Heimdal Desktop will appear in your menu bar
   - Complete the onboarding wizard
   - Access the dashboard from the menu bar icon

5. **Access the Dashboard**
   - Open your browser to `http://localhost:8080`
   - Or click the menu bar icon and select "Open Dashboard"

**Note**: macOS requires libpcap permissions. The application will guide you through granting these permissions.

#### Linux

1. **Install from Package**

   **Debian/Ubuntu:**
   ```bash
   sudo dpkg -i heimdal-desktop-linux-amd64.deb
   sudo apt-get install -f  # Install dependencies
   ```

   **Fedora/RHEL:**
   ```bash
   sudo rpm -i heimdal-desktop-linux-amd64.rpm
   ```

2. **Grant Capabilities**
   ```bash
   sudo setcap cap_net_raw,cap_net_admin=eip /usr/bin/heimdal-desktop
   ```

3. **Launch the Application**
   ```bash
   heimdal-desktop
   ```
   
   Or launch from your application menu (if desktop environment supports system tray)

4. **First Launch**
   - Heimdal Desktop will appear in your system tray (if available)
   - Complete the onboarding wizard
   - Access the dashboard from the system tray or browser

5. **Access the Dashboard**
   - Open your browser to `http://localhost:8080`

**Note**: Linux requires libpcap-dev and appropriate capabilities (CAP_NET_RAW, CAP_NET_ADMIN) for packet capture.

## Monorepo Architecture

Heimdal uses a unified monorepo structure that maximizes code reuse between hardware and desktop products while maintaining clean separation of platform-specific concerns.

### Directory Structure

```
heimdal/
├── cmd/                      # Entry points
│   ├── heimdal-hardware/    # Hardware product entry point
│   └── heimdal-desktop/     # Desktop product entry point
├── internal/
│   ├── core/                # Shared business logic
│   │   ├── packet/          # Packet analysis (shared)
│   │   ├── cloud/           # Cloud communication (shared)
│   │   ├── detection/       # Anomaly detection (shared)
│   │   └── profiler/        # Behavioral profiling (shared)
│   ├── platform/            # Platform abstraction layer
│   │   ├── interfaces.go    # Core interfaces
│   │   ├── linux_embedded/  # Raspberry Pi implementations
│   │   ├── desktop_windows/ # Windows implementations
│   │   ├── desktop_macos/   # macOS implementations
│   │   └── desktop_linux/   # Linux desktop implementations
│   ├── hardware/            # Hardware-specific logic
│   │   └── orchestrator/    # Hardware orchestrator
│   └── desktop/             # Desktop-specific logic
│       ├── orchestrator/    # Desktop orchestrator
│       ├── featuregate/     # Tier management
│       ├── visualizer/      # Local dashboard server
│       ├── systray/         # System tray integration
│       └── installer/       # Installation logic
├── web/dashboard/           # Shared web UI
├── test/                    # Tests
│   ├── integration/         # Integration tests
│   ├── property/            # Property-based tests
│   └── mocks/               # Mock implementations
├── build/                   # Build system
├── ansible/                 # Ansible deployment (hardware)
└── Makefile                 # Build targets
```

### Core Components (Shared)

Both products share these core components:

- **Packet Analysis**: Protocol parsing, metadata extraction
- **Cloud Communication**: AWS IoT and Google Cloud connectors
- **Anomaly Detection**: Behavioral analysis and threat detection
- **Behavioral Profiler**: Traffic pattern aggregation
- **Database**: BadgerDB for local persistence

### Platform Abstraction Layer

The platform abstraction layer defines three key interfaces:

1. **PacketCaptureProvider**: Abstracts packet capture mechanisms
   - Hardware: Raw sockets / AF_PACKET
   - Desktop: gopacket with PCAP/Npcap

2. **SystemIntegrator**: Abstracts OS-level service integration
   - Hardware: systemd service
   - Desktop Windows: Windows Service API
   - Desktop macOS: LaunchAgent
   - Desktop Linux: systemd user service

3. **StorageProvider**: Abstracts data persistence
   - Platform-specific paths for configuration and data

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md) and the [Developer Guide](docs/DEVELOPER_GUIDE.md).

## Configuration

The sensor is configured via `/etc/heimdal/config.json`. Key settings include:

- **Database Path**: Location for BadgerDB storage
- **Discovery Intervals**: ARP scan frequency and mDNS settings
- **Interceptor**: Enable/disable ARP spoofing
- **API Port**: Web dashboard port (default: 8080)
- **Cloud Connector**: Optional cloud platform integration

For complete configuration documentation, see [CONFIG.md](CONFIG.md).

## API Endpoints

The sensor provides a REST API for programmatic access:

- `GET /api/v1/devices` - List all discovered devices
- `GET /api/v1/devices/:mac` - Get device details
- `GET /api/v1/profiles/:mac` - Get behavioral profile
- `GET /api/v1/stats` - System statistics
- `GET /api/v1/health` - Health check

## Development

### Building from Source

**Requirements:**
- Go 1.21 or later
- Cross-compilation toolchains for target platforms
- Make

**Build Targets:**

```bash
# Build all products for all platforms
make build-all

# Build hardware product (ARM64 Linux)
make build-hardware

# Build desktop products
make build-desktop-windows    # Windows (amd64)
make build-desktop-macos      # macOS (amd64 and arm64)
make build-desktop-linux      # Linux (amd64)
make build-desktop-all        # All desktop platforms

# Clean build artifacts
make clean
```

**Build Output:**
- Hardware binary: `bin/heimdal-hardware`
- Desktop binaries: `bin/heimdal-desktop-{platform}-{arch}`

For detailed build instructions including cross-compilation setup, see [BUILD.md](BUILD.md).

### Running Tests

```bash
# Run all tests
make test

# Run specific test suites
make test-unit         # Unit tests only
make test-property     # Property-based tests (100 iterations each)
make test-integration  # Integration tests
make test-coverage     # Generate coverage report

# Run platform-specific tests (must run on target platform)
make test-windows      # Windows-specific tests
make test-macos        # macOS-specific tests
make test-linux        # Linux-specific tests
```

**Test Coverage Goals:**
- Core modules: 70% minimum
- Platform implementations: Best effort
- Integration tests: Critical paths

### Creating Installers

```bash
# Create Windows installer (requires NSIS or WiX)
make package-windows

# Create macOS installer (requires macOS)
make package-macos

# Create Linux packages
make package-linux-deb    # Debian/Ubuntu
make package-linux-rpm    # Fedora/RHEL
```

Installers are output to `build/installers/{platform}/`

## Security Considerations

- **Privilege Management**: Runs as non-root user with Linux capabilities
- **ARP Spoofing**: Inherently invasive - only use on networks you own/control
- **Local Network Trust**: Web API has no authentication (designed for local network use)
- **Data Privacy**: All data stored locally by default; cloud transmission requires explicit configuration

## Troubleshooting

### Hardware Product

#### Sensor Not Starting

Check systemd logs:
```bash
sudo journalctl -u heimdal -f
```

Verify IP forwarding is enabled:
```bash
sysctl net.ipv4.ip_forward
```

Check capabilities:
```bash
getcap /opt/heimdal/bin/heimdal-hardware
```

#### No Devices Discovered

- Ensure the sensor is on the same network as target devices
- Check that the network interface is correctly detected in logs
- Verify ARP scanning is enabled in configuration

#### Dashboard Not Accessible

- Check that the API server is running: `curl http://localhost:8080/api/v1/health`
- Verify firewall rules allow access to port 8080
- Check API logs for errors

### Desktop Product

#### Windows Issues

**Npcap Not Found:**
- Download and install Npcap from https://npcap.com/
- Ensure "WinPcap API-compatible Mode" is enabled during installation
- Restart Heimdal Desktop after installing Npcap

**Permission Denied:**
- Right-click Heimdal Desktop and select "Run as Administrator"
- Check Windows Firewall settings

**System Tray Icon Not Appearing:**
- Check Windows notification area settings
- Ensure Heimdal Desktop is allowed to show notifications

#### macOS Issues

**Permission Denied:**
- Grant Full Disk Access in System Preferences → Security & Privacy
- Run with sudo on first launch: `sudo /Applications/Heimdal\ Desktop.app/Contents/MacOS/heimdal-desktop`
- Check that libpcap permissions are granted

**Menu Bar Icon Not Appearing:**
- Check macOS menu bar settings
- Ensure the application has accessibility permissions

**"Unidentified Developer" Warning:**
- Right-click the application and select "Open"
- Or go to System Preferences → Security & Privacy and click "Open Anyway"

#### Linux Issues

**Capability Errors:**
```bash
# Grant required capabilities
sudo setcap cap_net_raw,cap_net_admin=eip /usr/bin/heimdal-desktop

# Verify capabilities
getcap /usr/bin/heimdal-desktop
```

**libpcap Not Found:**
```bash
# Debian/Ubuntu
sudo apt-get install libpcap-dev

# Fedora/RHEL
sudo dnf install libpcap-devel
```

**System Tray Not Available:**
- Heimdal Desktop requires a desktop environment with system tray support
- On headless systems, access the dashboard directly at http://localhost:8080

### Common Issues (Both Products)

#### High CPU/Memory Usage

- Review packet capture rate limits in configuration
- Check for excessive device count or traffic volume
- Monitor resource usage:
  - Hardware: `top -p $(pgrep heimdal-hardware)`
  - Desktop: `top -p $(pgrep heimdal-desktop)`

#### Dashboard Not Loading

- Verify the application is running
- Check that port 8080 is not in use by another application
- Try accessing via http://localhost:8080 instead of http://127.0.0.1:8080
- Check browser console for JavaScript errors

#### Configuration Errors

- Validate JSON syntax in configuration file
- Check file permissions (must be readable by the application)
- Review logs for specific configuration errors
- See [CONFIG.md](CONFIG.md) for configuration reference

For more detailed troubleshooting, see the [Desktop User Guide](docs/DESKTOP_USER_GUIDE.md).

## Contributing

This is a private project for the Heimdal security platform. For questions or issues, contact the development team.

## License

Proprietary - All rights reserved.

## Feature Tiers (Desktop Product)

Heimdal Desktop offers multiple subscription tiers:

### Free Tier
- Network device discovery and visualization
- Real-time traffic monitoring
- Basic behavioral profiling
- Local dashboard access
- Desktop notifications

### Pro Tier
- All Free tier features
- Active traffic blocking
- Advanced filtering rules
- Cloud synchronization
- Priority support

### Enterprise Tier
- All Pro tier features
- Multi-device management
- API access for automation
- Custom integrations
- Dedicated support

To upgrade your tier, visit the dashboard settings or contact sales.

## Related Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) - Detailed architecture and component design
- [CONFIG.md](CONFIG.md) - Complete configuration reference
- [BUILD.md](BUILD.md) - Build process and cross-compilation guide
- [LOGGING_AND_ERROR_HANDLING.md](LOGGING_AND_ERROR_HANDLING.md) - Error handling patterns
- [Desktop User Guide](docs/DESKTOP_USER_GUIDE.md) - Desktop installation and usage guide
- [Developer Guide](docs/DEVELOPER_GUIDE.md) - Contributing and extending Heimdal
- [Ansible Migration Guide](ansible/MIGRATION_GUIDE.md) - Migrating hardware deployments
