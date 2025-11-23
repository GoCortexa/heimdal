# Getting Started with Heimdal Desktop

Welcome to Heimdal Desktop - your intelligent network security companion!

## What is Heimdal Desktop?

Heimdal Desktop is a free network monitoring tool that:
- **Discovers** all devices on your network automatically
- **Identifies** devices by manufacturer, type, and hostname
- **Monitors** network traffic and builds behavioral profiles
- **Detects** anomalies that might indicate security threats
- **Protects** your privacy with local-first processing

## Quick Start (5 minutes)

### 1. Download & Install

**macOS:**
```bash
# Download the DMG
open heimdal-desktop-macos.dmg

# Drag to Applications folder
# Launch from Applications
```

**Windows:**
```bash
# Download the installer
heimdal-desktop-windows-installer.exe

# Run as Administrator
# Follow installation wizard
```

**Linux:**
```bash
# Debian/Ubuntu
sudo dpkg -i heimdal-desktop-linux-amd64.deb

# Fedora/RHEL
sudo rpm -i heimdal-desktop-linux-amd64.rpm

# Grant capabilities
sudo setcap cap_net_raw,cap_net_admin=eip /usr/bin/heimdal-desktop
```

### 2. First Launch

When you first launch Heimdal Desktop:

1. **Accept Permissions**: Heimdal needs packet capture permissions to monitor your network
2. **Select Network Interface**: Choose your primary network interface (usually auto-detected)
3. **Configure Auto-Start** (optional): Start Heimdal automatically when you log in

### 3. Access Dashboard

Open your browser to: **http://localhost:8080**

You'll see:
- List of all devices on your network
- Device types with icons (📱💻🖨️📺🔌)
- Vendor information
- Real-time activity status

### 4. View Device Profiles

Click "View Profile" on any device to see:
- Traffic patterns and volume
- Communication destinations
- Port usage
- 24-hour activity timeline
- Behavioral baselines

## Understanding Your Dashboard

### Device List

Each device shows:
- **Status**: ● (active) or ○ (inactive)
- **Icon**: Device type indicator
- **Name**: Hostname or mDNS name
- **Type Badge**: Device classification
- **Vendor**: Manufacturer name
- **Last Seen**: When device was last active

### Device Types

Heimdal automatically classifies devices:
- 📱 **Phones & Tablets**: iPhones, Android phones, iPads
- 💻 **Computers**: Laptops, desktops, servers
- 📡 **Network Equipment**: Routers, switches, access points
- 🖨️ **Printers & Scanners**: Network printers
- 📺 **Entertainment**: Smart TVs, streaming devices, game consoles
- 🔌 **IoT Devices**: Smart home, cameras, speakers
- 🏠 **Smart Home**: HomeKit, Matter, smart lights
- 💾 **Storage**: NAS devices

### Filtering & Search

Use the filter button (🔍) to:
- Filter by device type
- Search by name, IP, MAC, or vendor
- Find specific devices quickly

## Privacy & Cloud Features

### Free Tier (What You Get)

✅ **Included:**
- Full network visibility
- Device discovery and identification
- Behavioral profiling with baselines
- Anomaly detection
- Local dashboard
- Desktop notifications

✅ **Optional Cloud Features:**
- Help improve device identification (send device data)
- Contribute to anomaly detection models (send profiles)
- All data is anonymized by default

### Privacy Controls

Configure what data is shared in `config.json`:

```json
{
  "cloud": {
    "enabled": true,
    "send_device_info": true,    // Help build device database
    "send_profiles": true,       // Help train ML models
    "send_anomalies": false,     // Keep anomalies private
    "anonymize_data": true,      // Hash sensitive fields
    "send_diagnostics": false    // Opt-in for diagnostics
  }
}
```

**What gets anonymized:**
- Device names → Hashed
- Hostnames → Hashed
- IP addresses → Last octet hashed
- MAC addresses → Vendor prefix preserved, rest hashed

**What stays local:**
- All raw packet data
- Full device profiles
- Anomaly details
- Network topology

## Anomaly Detection

Heimdal detects:
- **Unexpected Destinations**: Communication with unusual IPs
- **Unusual Ports**: Traffic on non-standard ports
- **Traffic Spikes**: Sudden increases in activity
- **Protocol Shifts**: Changes in TCP/UDP/ICMP distribution
- **Destination Anomalies**: Unusual number of communication partners

Anomalies are shown in:
- Desktop notifications (real-time)
- Dashboard anomaly section
- Device profile view

## Troubleshooting

### No devices appearing

**Check:**
1. Dashboard is at http://localhost:8080
2. Heimdal Desktop is running (check system tray)
3. Network interface is correct (check logs)
4. Packet capture permissions granted

**macOS:** May need to run with sudo first time
**Windows:** Requires Npcap installed
**Linux:** Requires CAP_NET_RAW capability

### Devices show as "Unknown"

This is normal for:
- New/rare hardware not in OUI database
- Devices that don't respond to mDNS
- Devices without distinctive hostnames

Classification improves as Heimdal gathers more data.

### High CPU/Memory usage

**Normal usage:**
- CPU: 5-15% during scans, <5% idle
- Memory: 50-150MB depending on device count

**If higher:**
- Check number of devices (100+ devices = more resources)
- Reduce scan frequency in config
- Disable mDNS if not needed

### Dashboard not loading

1. Check Heimdal Desktop is running
2. Try http://127.0.0.1:8080 instead
3. Check firewall isn't blocking port 8080
4. Check logs: `~/Library/Logs/Heimdal/heimdal.log` (macOS)

## Configuration

Config file location:
- **macOS**: `~/Library/Application Support/Heimdal/config.json`
- **Windows**: `%APPDATA%\Heimdal\config.json`
- **Linux**: `~/.config/heimdal/config.json`

Key settings:
```json
{
  "discovery": {
    "scan_interval_seconds": 60,  // How often to scan
    "mdns_enabled": true           // Enable service discovery
  },
  "detection": {
    "sensitivity": 0.7             // 0.0 (low) to 1.0 (high)
  },
  "visualizer": {
    "port": 8080                   // Dashboard port
  }
}
```

## Upgrading to Pro

Want more features?

**Pro Tier includes:**
- Active traffic blocking
- Advanced filtering rules
- Cloud synchronization
- Priority support

Contact: sales@heimdal.io

## Getting Help

- **Documentation**: https://heimdal.io/docs
- **Community**: https://community.heimdal.io
- **Issues**: https://github.com/your-org/heimdal/issues
- **Email**: support@heimdal.io

## What's Next?

1. **Explore your network**: See what devices are connected
2. **Review profiles**: Understand device behavior patterns
3. **Monitor anomalies**: Watch for unusual activity
4. **Customize settings**: Adjust sensitivity and scan intervals
5. **Share feedback**: Help us improve Heimdal!

---

**Thank you for using Heimdal Desktop!** 🛡️

Your network security starts here.

