# Heimdal Configuration Reference

This document provides a complete reference for all configuration options available in Heimdal.

## Configuration File Location

The configuration file is located at `/etc/heimdal/config.json` by default. You can specify an alternate location using the `--config` command-line flag:

```bash
/opt/heimdal/bin/heimdal --config /path/to/config.json
```

## Configuration Format

The configuration file uses JSON format with the following top-level sections:

- `database` - Database storage settings
- `network` - Network interface configuration
- `discovery` - Device discovery settings
- `interceptor` - Traffic interception settings
- `profiler` - Behavioral profiling settings
- `api` - Web API and dashboard settings
- `cloud` - Cloud connector settings
- `logging` - Logging configuration

## Complete Configuration Example

```json
{
  "database": {
    "path": "/var/lib/heimdal/db",
    "gc_interval_minutes": 5
  },
  "network": {
    "interface": "",
    "auto_detect": true
  },
  "discovery": {
    "arp_scan_interval_seconds": 60,
    "mdns_enabled": true,
    "inactive_timeout_minutes": 5
  },
  "interceptor": {
    "enabled": true,
    "spoof_interval_seconds": 2,
    "target_macs": []
  },
  "profiler": {
    "persist_interval_seconds": 60,
    "max_destinations": 100
  },
  "api": {
    "port": 8080,
    "host": "0.0.0.0",
    "rate_limit_per_minute": 100
  },
  "cloud": {
    "enabled": false,
    "provider": "aws",
    "aws": {
      "endpoint": "",
      "client_id": "",
      "cert_path": "",
      "key_path": ""
    },
    "gcp": {
      "project_id": "",
      "topic_id": ""
    }
  },
  "logging": {
    "level": "info",
    "file": "/var/log/heimdal/heimdal.log"
  }
}
```

## Configuration Sections

### Database Configuration

Controls the embedded BadgerDB database settings.

```json
{
  "database": {
    "path": "/var/lib/heimdal/db",
    "gc_interval_minutes": 5
  }
}
```

**Options:**

- **`path`** (string, required)
  - Path to the BadgerDB database directory
  - Must be writable by the heimdal user
  - Default: `/var/lib/heimdal/db`
  - Example: `/var/lib/heimdal/db`

- **`gc_interval_minutes`** (integer, optional)
  - Interval in minutes between garbage collection runs
  - Garbage collection reclaims disk space from deleted records
  - Default: `5`
  - Range: 1-60
  - Example: `5`

### Network Configuration

Controls network interface detection and selection.

```json
{
  "network": {
    "interface": "",
    "auto_detect": true
  }
}
```

**Options:**

- **`interface`** (string, optional)
  - Specific network interface to use (e.g., "eth0", "wlan0")
  - Leave empty for automatic detection
  - Default: `""` (auto-detect)
  - Example: `"eth0"`

- **`auto_detect`** (boolean, required)
  - Enable automatic network interface detection
  - When true, sensor will find the primary interface automatically
  - When false, `interface` must be specified
  - Default: `true`
  - Example: `true`

**Auto-Detection Behavior:**
- Searches for interfaces in order: eth0, wlan0, en0, other active interfaces
- Selects first interface with a valid IP address and gateway
- Retries every 5 seconds until a valid network is found
- Blocks application startup until network is detected

### Discovery Configuration

Controls device discovery behavior using ARP and mDNS.

```json
{
  "discovery": {
    "arp_scan_interval_seconds": 60,
    "mdns_enabled": true,
    "inactive_timeout_minutes": 5
  }
}
```

**Options:**

- **`arp_scan_interval_seconds`** (integer, required)
  - Interval in seconds between ARP scans of the entire subnet
  - Lower values provide faster discovery but increase network traffic
  - Default: `60`
  - Range: 10-300
  - Example: `60`

- **`mdns_enabled`** (boolean, required)
  - Enable mDNS/DNS-SD device discovery
  - Discovers device names and services
  - Complements ARP scanning
  - Default: `true`
  - Example: `true`

- **`inactive_timeout_minutes`** (integer, required)
  - Minutes without seeing a device before marking it inactive
  - Inactive devices are not removed, just flagged
  - Default: `5`
  - Range: 1-60
  - Example: `5`

**Discovery Behavior:**
- ARP scanning discovers IP and MAC addresses
- mDNS discovers device names and services
- Both methods run concurrently
- Discovered devices are immediately persisted to database
- Devices are sent to interceptor for traffic monitoring

### Interceptor Configuration

Controls ARP spoofing and traffic interception.

```json
{
  "interceptor": {
    "enabled": true,
    "spoof_interval_seconds": 2,
    "target_macs": []
  }
}
```

**Options:**

- **`enabled`** (boolean, required)
  - Enable or disable ARP spoofing
  - When disabled, sensor operates in passive monitoring mode
  - Default: `true`
  - Example: `true`

- **`spoof_interval_seconds`** (integer, required)
  - Interval in seconds between ARP spoof packets
  - Lower values maintain more reliable interception
  - Higher values reduce network traffic
  - Default: `2`
  - Range: 1-10
  - Example: `2`

- **`target_macs`** (array of strings, optional)
  - List of specific MAC addresses to intercept
  - Empty array means intercept all discovered devices
  - Use to limit interception to specific devices
  - Default: `[]` (all devices)
  - Example: `["aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"]`

**Security Warning:**
ARP spoofing is inherently invasive and can disrupt network connectivity if misconfigured. Only use on networks you own or have explicit permission to monitor. Ensure IP forwarding is enabled on the host system.

**Requirements:**
- IP forwarding must be enabled: `net.ipv4.ip_forward=1`
- Binary must have `CAP_NET_RAW` and `CAP_NET_ADMIN` capabilities
- Network interface must support promiscuous mode

### Profiler Configuration

Controls behavioral profiling and data aggregation.

```json
{
  "profiler": {
    "persist_interval_seconds": 60,
    "max_destinations": 100
  }
}
```

**Options:**

- **`persist_interval_seconds`** (integer, required)
  - Interval in seconds between profile persistence to database
  - Lower values provide more frequent updates but increase I/O
  - Higher values reduce I/O but risk data loss on crash
  - Default: `60`
  - Range: 10-300
  - Example: `60`

- **`max_destinations`** (integer, required)
  - Maximum number of destination IPs to track per device
  - Limits memory usage for devices with many connections
  - Least-recently-seen destinations are pruned when limit reached
  - Default: `100`
  - Range: 10-1000
  - Example: `100`

**Profile Data:**
Profiles include:
- Destination IPs and packet counts
- Destination ports and frequencies
- Protocol distribution (TCP, UDP, ICMP, etc.)
- Total packets and bytes
- Hourly activity pattern (24-hour)

### API Configuration

Controls the web API server and dashboard.

```json
{
  "api": {
    "port": 8080,
    "host": "0.0.0.0",
    "rate_limit_per_minute": 100
  }
}
```

**Options:**

- **`port`** (integer, required)
  - TCP port for the web API server
  - Must not conflict with other services
  - Default: `8080`
  - Range: 1024-65535
  - Example: `8080`

- **`host`** (string, required)
  - IP address to bind the server to
  - `0.0.0.0` binds to all interfaces
  - `127.0.0.1` binds to localhost only
  - Specific IP binds to that interface only
  - Default: `"0.0.0.0"`
  - Example: `"0.0.0.0"`

- **`rate_limit_per_minute`** (integer, required)
  - Maximum API requests per minute per IP address
  - Prevents abuse and resource exhaustion
  - Default: `100`
  - Range: 10-1000
  - Example: `100`

**API Endpoints:**
- `GET /api/v1/devices` - List all devices
- `GET /api/v1/devices/:mac` - Get device details
- `GET /api/v1/profiles/:mac` - Get behavioral profile
- `GET /api/v1/stats` - System statistics
- `GET /api/v1/health` - Health check
- `GET /` - Dashboard HTML

**Security Note:**
The API has no authentication and is designed for local network use. Do not expose to the internet without adding authentication.

### Cloud Configuration

Controls optional cloud connectivity for future integration.

```json
{
  "cloud": {
    "enabled": false,
    "provider": "aws",
    "aws": {
      "endpoint": "",
      "client_id": "",
      "cert_path": "",
      "key_path": ""
    },
    "gcp": {
      "project_id": "",
      "topic_id": ""
    }
  }
}
```

**Options:**

- **`enabled`** (boolean, required)
  - Enable or disable cloud connectivity
  - Default: `false`
  - Example: `false`

- **`provider`** (string, required when enabled)
  - Cloud provider to use: "aws" or "gcp"
  - Default: `"aws"`
  - Example: `"aws"`

#### AWS IoT Configuration

```json
{
  "aws": {
    "endpoint": "xxxxx.iot.us-east-1.amazonaws.com",
    "client_id": "heimdal-sensor-01",
    "cert_path": "/etc/heimdal/certs/device.crt",
    "key_path": "/etc/heimdal/certs/device.key"
  }
}
```

**AWS Options:**

- **`endpoint`** (string, required)
  - AWS IoT Core endpoint URL
  - Format: `xxxxx.iot.region.amazonaws.com`
  - Example: `"a1b2c3d4e5f6g7.iot.us-east-1.amazonaws.com"`

- **`client_id`** (string, required)
  - MQTT client ID for this sensor
  - Must be unique across all sensors
  - Example: `"heimdal-sensor-01"`

- **`cert_path`** (string, required)
  - Path to device certificate file
  - Must be readable by heimdal user
  - Example: `"/etc/heimdal/certs/device.crt"`

- **`key_path`** (string, required)
  - Path to device private key file
  - Must be readable by heimdal user
  - Should have restricted permissions (0600)
  - Example: `"/etc/heimdal/certs/device.key"`

#### Google Cloud Configuration

```json
{
  "gcp": {
    "project_id": "heimdal-project",
    "topic_id": "sensor-data"
  }
}
```

**GCP Options:**

- **`project_id`** (string, required)
  - Google Cloud project ID
  - Example: `"heimdal-project"`

- **`topic_id`** (string, required)
  - Pub/Sub topic ID for sensor data
  - Topic must exist in the project
  - Example: `"sensor-data"`

**Cloud Behavior:**
- Transmits behavioral profiles every 5 minutes
- Retries failed transmissions with exponential backoff
- Continues local operations if cloud unavailable
- Queues up to 100 failed transmissions
- Drops oldest data if queue full

**Note:** Current implementations are stubs showing the integration pattern. Full implementation requires valid cloud credentials and additional configuration.

### Logging Configuration

Controls application logging behavior.

```json
{
  "logging": {
    "level": "info",
    "file": "/var/log/heimdal/heimdal.log"
  }
}
```

**Options:**

- **`level`** (string, required)
  - Logging level: "debug", "info", "warn", "error"
  - Default: `"info"`
  - Example: `"info"`

- **`file`** (string, required)
  - Path to log file
  - Must be writable by heimdal user
  - Logs also go to stdout for systemd journal
  - Default: `"/var/log/heimdal/heimdal.log"`
  - Example: `"/var/log/heimdal/heimdal.log"`

**Log Levels:**

- **debug**: Detailed packet information, component state changes
- **info**: Startup, shutdown, device discovery, profile updates
- **warn**: Retry attempts, degraded functionality, buffer usage
- **error**: Component failures, database errors, critical issues

**Log Rotation:**
Configure log rotation using logrotate:

```
/var/log/heimdal/heimdal.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 heimdal heimdal
    postrotate
        systemctl reload heimdal
    endscript
}
```

## Configuration Validation

The sensor validates configuration on startup:

- Required fields must be present
- Numeric values must be within valid ranges
- File paths must be accessible
- Network interface must exist (if specified)
- Cloud credentials must be valid (if cloud enabled)

Validation errors are logged and prevent startup.

## Environment Variables

The sensor does not use environment variables for configuration. All settings must be in the configuration file.

## Configuration Updates

To update configuration:

1. Edit `/etc/heimdal/config.json`
2. Validate JSON syntax: `jq . /etc/heimdal/config.json`
3. Restart the service: `sudo systemctl restart heimdal`

The sensor does not support hot-reloading of configuration. A restart is required for changes to take effect.

## Ansible Configuration Management

When deploying with Ansible, configuration is managed via the template `ansible/roles/heimdal_sensor/templates/config.json.j2`.

Edit variables in `ansible/group_vars/all.yml`:

```yaml
heimdal_config:
  database:
    path: /var/lib/heimdal/db
  discovery:
    arp_scan_interval_seconds: 60
  api:
    port: 8080
```

Then deploy:

```bash
ansible-playbook -i inventory.ini playbook.yml
```

## Troubleshooting Configuration Issues

### Sensor Won't Start

Check configuration syntax:
```bash
jq . /etc/heimdal/config.json
```

Check systemd logs:
```bash
sudo journalctl -u heimdal -n 50
```

### Permission Errors

Ensure heimdal user can access all configured paths:
```bash
sudo -u heimdal ls -la /etc/heimdal/config.json
sudo -u heimdal ls -la /var/lib/heimdal/db
sudo -u heimdal ls -la /var/log/heimdal/
```

### Network Interface Not Found

List available interfaces:
```bash
ip link show
```

Update configuration with correct interface name.

### Database Errors

Check database directory permissions:
```bash
ls -la /var/lib/heimdal/
```

Ensure sufficient disk space:
```bash
df -h /var/lib/heimdal/
```

### API Port Already in Use

Check what's using the port:
```bash
sudo lsof -i :8080
```

Change port in configuration or stop conflicting service.

## Security Best Practices

1. **Restrict Configuration File Permissions**
   ```bash
   sudo chmod 0644 /etc/heimdal/config.json
   sudo chown heimdal:heimdal /etc/heimdal/config.json
   ```

2. **Protect Cloud Credentials**
   ```bash
   sudo chmod 0600 /etc/heimdal/certs/*.key
   sudo chown heimdal:heimdal /etc/heimdal/certs/*
   ```

3. **Limit API Access**
   - Use firewall rules to restrict API access to local network
   - Consider binding to specific interface instead of 0.0.0.0

4. **Monitor Logs**
   - Regularly review logs for errors and security events
   - Set up log monitoring and alerting

5. **Backup Configuration**
   - Keep backups of configuration files
   - Version control configuration templates

## Related Documentation

- [README.md](README.md) - Project overview and quick start
- [ARCHITECTURE.md](ARCHITECTURE.md) - Architecture and component design
- [BUILD.md](BUILD.md) - Build process and cross-compilation
- [LOGGING_AND_ERROR_HANDLING.md](LOGGING_AND_ERROR_HANDLING.md) - Error handling patterns
