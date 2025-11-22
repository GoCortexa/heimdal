# Heimdal Ansible Deployment

This directory contains Ansible playbooks for deploying the Heimdal network security sensor to Raspberry Pi hardware.

## Overview

The Ansible playbooks automate the deployment of Heimdal sensors including:

- System dependencies installation
- Binary deployment
- Configuration management
- Systemd service setup
- Security hardening

## Backward Compatibility

These playbooks are **fully compatible** with both:

- **Legacy binary**: `heimdal` (pre-monorepo)
- **New binary**: `heimdal-hardware` (monorepo architecture)

The sensor includes automatic configuration migration, so existing deployments will continue to work without modification.

## Quick Start

### Prerequisites

- Ansible 2.9 or later
- SSH access to target Raspberry Pi devices
- Heimdal binary built for ARM64 Linux

### Basic Deployment

1. **Build the hardware binary**:
   ```bash
   make build-hardware
   ```

2. **Copy binary to Ansible files directory**:
   ```bash
   cp bin/heimdal-hardware-arm64 ansible/roles/heimdal_sensor/files/heimdal
   ```

3. **Update inventory**:
   Edit `ansible/inventory.ini` with your sensor IP addresses:
   ```ini
   [heimdal_sensors]
   sensor1 ansible_host=192.168.1.100
   sensor2 ansible_host=192.168.1.101
   ```

4. **Configure variables** (optional):
   Edit `ansible/group_vars/all.yml` to customize settings

5. **Deploy**:
   ```bash
   ansible-playbook -i ansible/inventory.ini ansible/playbook.yml
   ```

## Directory Structure

```
ansible/
├── README.md                           # This file
├── MIGRATION_GUIDE.md                  # Configuration migration guide
├── playbook.yml                        # Main playbook
├── inventory.ini                       # Inventory file
├── group_vars/
│   └── all.yml                         # Global variables
└── roles/
    └── heimdal_sensor/
        ├── tasks/
        │   └── main.yml                # Deployment tasks
        ├── templates/
        │   ├── config.json.j2          # Configuration template
        │   └── heimdal.service.j2      # Systemd service template
        ├── handlers/
        │   └── main.yml                # Service handlers
        └── files/
            └── .gitkeep                # Place binary here
```

## Configuration

### Variables

All configuration variables are defined in `group_vars/all.yml`. Key variables include:

- **Paths**:
  - `heimdal_bin_path`: Binary installation path (default: `/opt/heimdal/bin`)
  - `heimdal_config_path`: Configuration path (default: `/etc/heimdal`)
  - `heimdal_data_path`: Data storage path (default: `/var/lib/heimdal`)
  - `heimdal_log_path`: Log file path (default: `/var/log/heimdal`)

- **Network**:
  - `heimdal_network_auto_detect`: Auto-detect network interface (default: `true`)
  - `heimdal_network_interface`: Specific interface to use (default: `""`)

- **Discovery**:
  - `heimdal_arp_scan_interval`: ARP scan interval in seconds (default: `60`)
  - `heimdal_mdns_enabled`: Enable mDNS discovery (default: `true`)
  - `heimdal_inactive_timeout`: Device inactive timeout in minutes (default: `5`)

- **Interceptor**:
  - `heimdal_interceptor_enabled`: Enable traffic interception (default: `true`)
  - `heimdal_spoof_interval`: ARP spoof interval in seconds (default: `2`)

- **API**:
  - `heimdal_api_port`: API server port (default: `8080`)
  - `heimdal_api_host`: API server host (default: `0.0.0.0`)

- **Cloud**:
  - `heimdal_cloud_enabled`: Enable cloud connectivity (default: `false`)
  - `heimdal_cloud_provider`: Cloud provider (`aws` or `gcp`)

- **Logging**:
  - `heimdal_log_level`: Log level (default: `info`)

### Customization

To customize for specific environments:

1. **Per-host variables**: Create `host_vars/<hostname>.yml`
2. **Per-group variables**: Create additional group variable files
3. **Override at runtime**: Use `-e` flag with ansible-playbook

Example:
```bash
ansible-playbook -i inventory.ini playbook.yml -e "heimdal_api_port=9090"
```

## Binary Compatibility

### Legacy Binary (Pre-Monorepo)

If deploying the legacy `heimdal` binary:

```bash
cp bin/heimdal ansible/roles/heimdal_sensor/files/heimdal
```

### New Binary (Monorepo)

If deploying the new `heimdal-hardware` binary:

```bash
cp bin/heimdal-hardware-arm64 ansible/roles/heimdal_sensor/files/heimdal
```

Note: The playbook expects the binary to be named `heimdal` in the files directory, regardless of the source binary name.

## Configuration Migration

The Heimdal sensor includes automatic configuration migration:

- **Legacy configurations are detected** and migrated automatically
- **Backups are created** before migration (`config.json.backup.YYYYMMDD-HHMMSS`)
- **Migration is logged** to systemd journal and log files
- **No playbook changes required** for configuration format updates

See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for detailed information.

## Verification

After deployment, verify the installation:

```bash
# Check service status
ansible heimdal_sensors -i inventory.ini -m shell -a "systemctl status heimdal"

# Check logs
ansible heimdal_sensors -i inventory.ini -m shell -a "journalctl -u heimdal -n 50"

# Check configuration
ansible heimdal_sensors -i inventory.ini -m shell -a "cat /etc/heimdal/config.json"

# Check for migration
ansible heimdal_sensors -i inventory.ini -m shell -a "ls -la /etc/heimdal/config.json.backup.*"
```

## Updating Sensors

To update sensors with a new binary:

1. Build new binary
2. Copy to files directory
3. Run playbook (will trigger restart via handler)

```bash
make build-hardware
cp bin/heimdal-hardware-arm64 ansible/roles/heimdal_sensor/files/heimdal
ansible-playbook -i inventory.ini playbook.yml
```

## Troubleshooting

### Service Won't Start

Check logs:
```bash
ansible heimdal_sensors -i inventory.ini -m shell -a "journalctl -u heimdal -n 100"
```

Common issues:
- Configuration validation errors
- Missing capabilities on binary
- Network interface not found
- Permission issues

### Configuration Migration Issues

Check for migration errors:
```bash
ansible heimdal_sensors -i inventory.ini -m shell -a "journalctl -u heimdal | grep -i migration"
```

Restore from backup if needed:
```bash
ansible heimdal_sensors -i inventory.ini -m shell -a "cp /etc/heimdal/config.json.backup.* /etc/heimdal/config.json"
```

### Binary Compatibility

Verify binary architecture:
```bash
ansible heimdal_sensors -i inventory.ini -m shell -a "file /opt/heimdal/bin/heimdal"
```

Should show: `ELF 64-bit LSB executable, ARM aarch64`

## Security Considerations

The playbook implements security best practices:

- **Dedicated system user**: Runs as non-root `heimdal` user
- **Minimal capabilities**: Only `CAP_NET_RAW` and `CAP_NET_ADMIN`
- **Systemd hardening**: `NoNewPrivileges`, `PrivateTmp`, `ProtectSystem`
- **Read-only system**: Only data and log directories are writable
- **IP forwarding**: Enabled for traffic interception

## Support

For issues or questions:

1. Check logs: `/var/log/heimdal/heimdal.log`
2. Check systemd journal: `journalctl -u heimdal`
3. Review configuration: `/etc/heimdal/config.json`
4. See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for migration issues

## License

See main project LICENSE file.
