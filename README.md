# Heimdal 2.0 Network Security Sensor

Heimdal is a zero-touch network security sensor designed for deployment on Raspberry Pi hardware. It performs automated network discovery, traffic interception via ARP spoofing, behavioral profiling of network devices, and provides a local web dashboard for monitoring.

## Features

- **Zero-Touch Provisioning**: Simply plug in power and Ethernet - no manual configuration required
- **Automated Device Discovery**: Continuous scanning using ARP and mDNS protocols
- **Traffic Interception**: ARP spoofing to intercept and analyze network traffic
- **Behavioral Profiling**: Build profiles of device communication patterns
- **Local Web Dashboard**: Monitor your network through a browser at `http://<sensor-ip>:8080`
- **Embedded Database**: All data stored locally using BadgerDB
- **Optional Cloud Connectivity**: Stub implementations for AWS IoT and Google Cloud
- **Single Binary**: Statically compiled with no external runtime dependencies
- **Ansible Deployment**: Automated provisioning and updates

## Quick Start

### Prerequisites

- Raspberry Pi 4 (or compatible ARM64 device)
- Raspberry Pi OS (64-bit) or similar Linux distribution
- Network connection (Ethernet recommended)
- Ansible control machine for deployment

### Deployment

1. **Configure Ansible Inventory**

   Edit `ansible/inventory.ini` with your Raspberry Pi details:
   ```ini
   [heimdal_sensors]
   heimdal-sensor-01 ansible_host=10.100.102.131 ansible_user=cortexa ansible_password=Password1
   ```

2. **Build the Binary**

   On your development machine (requires Go 1.21+ and ARM64 cross-compiler):
   ```bash
   ./build.sh
   ```

3. **Deploy with Ansible**

   ```bash
   cd ansible
   ansible-playbook -i inventory.ini playbook.yml
   ```

4. **Access the Dashboard**

   Open your browser to `http://<raspberry-pi-ip>:8080`

### Manual Installation

If you prefer manual installation without Ansible:

1. Copy the binary to `/opt/heimdal/bin/heimdal`
2. Create configuration at `/etc/heimdal/config.json` (see `config/config.json` for template)
3. Create directories: `/var/lib/heimdal`, `/var/log/heimdal`
4. Enable IP forwarding: `sudo sysctl -w net.ipv4.ip_forward=1`
5. Set capabilities: `sudo setcap cap_net_raw,cap_net_admin=eip /opt/heimdal/bin/heimdal`
6. Run: `/opt/heimdal/bin/heimdal --config /etc/heimdal/config.json`

## Architecture

Heimdal uses a concurrent, goroutine-based architecture with the following components:

- **Network Auto-Config**: Automatically detects network configuration
- **Device Discovery**: Scans network using ARP and mDNS
- **Traffic Interceptor**: Performs ARP spoofing to intercept traffic
- **Packet Analyzer**: Captures and analyzes network packets
- **Behavioral Profiler**: Aggregates traffic patterns per device
- **Database**: Embedded BadgerDB for local persistence
- **Web API**: REST API and dashboard for monitoring
- **Cloud Connector**: Optional cloud integration (AWS IoT, Google Cloud)

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md).

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
- ARM64 cross-compiler (`aarch64-linux-gnu-gcc`)
- libpcap development headers

**Build for Raspberry Pi:**
```bash
./build.sh
```

**Build for local testing (x86_64):**
```bash
go build -o heimdal ./cmd/heimdal
```

### Running Tests

**Unit Tests:**
```bash
go test ./internal/...
```

**Integration Tests:**
```bash
go test ./test/integration/...
```

### Project Structure

```
.
├── cmd/heimdal/              # Main entry point
├── internal/                 # Internal packages
│   ├── analyzer/            # Packet analyzer
│   ├── api/                 # Web API server
│   ├── cloud/               # Cloud connectors
│   ├── config/              # Configuration management
│   ├── database/            # BadgerDB wrapper
│   ├── discovery/           # Device discovery
│   ├── errors/              # Error handling
│   ├── interceptor/         # ARP spoofer
│   ├── logger/              # Logging utilities
│   ├── netconfig/           # Network auto-config
│   ├── orchestrator/        # Component orchestration
│   └── profiler/            # Behavioral profiler
├── web/dashboard/           # Web dashboard files
├── ansible/                 # Ansible deployment
├── config/                  # Default configuration
└── test/integration/        # Integration tests
```

## Security Considerations

- **Privilege Management**: Runs as non-root user with Linux capabilities
- **ARP Spoofing**: Inherently invasive - only use on networks you own/control
- **Local Network Trust**: Web API has no authentication (designed for local network use)
- **Data Privacy**: All data stored locally by default; cloud transmission requires explicit configuration

## Troubleshooting

### Sensor Not Starting

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
getcap /opt/heimdal/bin/heimdal
```

### No Devices Discovered

- Ensure the sensor is on the same network as target devices
- Check that the network interface is correctly detected in logs
- Verify ARP scanning is enabled in configuration

### Dashboard Not Accessible

- Check that the API server is running: `curl http://localhost:8080/api/v1/health`
- Verify firewall rules allow access to port 8080
- Check API logs for errors

### High CPU/Memory Usage

- Review packet capture rate limits in configuration
- Check for excessive device count or traffic volume
- Monitor with: `top -p $(pgrep heimdal)`

## Contributing

This is a private project for the Heimdal security platform. For questions or issues, contact the development team.

## License

Proprietary - All rights reserved.

## Related Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) - Detailed architecture and component design
- [CONFIG.md](CONFIG.md) - Complete configuration reference
- [BUILD.md](BUILD.md) - Build process and cross-compilation guide
- [LOGGING_AND_ERROR_HANDLING.md](LOGGING_AND_ERROR_HANDLING.md) - Error handling patterns
