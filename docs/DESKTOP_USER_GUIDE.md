# Heimdal Desktop User Guide

Welcome to Heimdal Desktop! This guide will help you install, configure, and use Heimdal Desktop to monitor and secure your home or office network.

## Table of Contents

1. [Introduction](#introduction)
2. [System Requirements](#system-requirements)
3. [Installation](#installation)
4. [First Launch and Onboarding](#first-launch-and-onboarding)
5. [Using the Dashboard](#using-the-dashboard)
6. [System Tray Features](#system-tray-features)
7. [Feature Tiers](#feature-tiers)
8. [Configuration](#configuration)
9. [Troubleshooting](#troubleshooting)
10. [FAQ](#faq)

## Introduction

Heimdal Desktop is a network security monitoring application that runs on your computer to provide visibility into your home or office network. It discovers devices, monitors traffic patterns, detects anomalies, and provides a real-time dashboard for network visualization.

### What Heimdal Desktop Does

- **Device Discovery**: Automatically finds all devices on your network
- **Traffic Monitoring**: Analyzes network traffic to understand communication patterns
- **Behavioral Profiling**: Builds profiles of normal device behavior
- **Anomaly Detection**: Alerts you to unusual or suspicious activity
- **Real-Time Visualization**: Shows your network map and traffic flows in a web dashboard
- **Desktop Integration**: Runs quietly in the background with system tray access

### How It Works

Heimdal Desktop uses ARP spoofing from your computer to gain visibility into network traffic. This allows it to see what devices are communicating without requiring dedicated hardware. The application runs locally on your machine and stores all data locally by default.

**Important**: ARP spoofing is an invasive technique. Only use Heimdal Desktop on networks you own or have explicit permission to monitor.

## System Requirements

### Windows

- **Operating System**: Windows 10 or later (64-bit)
- **RAM**: 2 GB minimum, 4 GB recommended
- **Disk Space**: 500 MB for application and data
- **Network**: Ethernet or Wi-Fi connection
- **Dependencies**: Npcap (included in installer)
- **Permissions**: Administrator access required

### macOS

- **Operating System**: macOS 10.15 (Catalina) or later
- **RAM**: 2 GB minimum, 4 GB recommended
- **Disk Space**: 500 MB for application and data
- **Network**: Ethernet or Wi-Fi connection
- **Dependencies**: libpcap (included in macOS)
- **Permissions**: Full Disk Access and packet capture permissions

### Linux

- **Operating System**: Ubuntu 20.04+, Fedora 34+, or equivalent
- **RAM**: 2 GB minimum, 4 GB recommended
- **Disk Space**: 500 MB for application and data
- **Network**: Ethernet or Wi-Fi connection
- **Dependencies**: libpcap-dev
- **Permissions**: CAP_NET_RAW and CAP_NET_ADMIN capabilities

## Installation

### Windows Installation

1. **Download the Installer**
   - Download `heimdal-desktop-windows-installer.exe` from the releases page
   - Save it to your Downloads folder

2. **Run the Installer**
   - Right-click the installer file
   - Select "Run as Administrator"
   - If Windows SmartScreen appears, click "More info" then "Run anyway"

3. **Follow the Installation Wizard**
   - Accept the license agreement
   - Choose installation location (default: `C:\Program Files\Heimdal Desktop`)
   - Select whether to install Npcap (required - recommended to install)
   - Choose whether to start on system boot
   - Click "Install"

4. **Complete Npcap Installation**
   - If Npcap is not already installed, the Npcap installer will launch
   - **Important**: Enable "WinPcap API-compatible Mode" during Npcap installation
   - Complete the Npcap installation wizard

5. **Launch Heimdal Desktop**
   - The installer will offer to launch Heimdal Desktop
   - Or launch it from the Start Menu: Start → Heimdal Desktop

### macOS Installation

1. **Download the Installer**
   - Download `heimdal-desktop-macos.dmg` from the releases page
   - Save it to your Downloads folder

2. **Open the DMG**
   - Double-click the DMG file to mount it
   - A Finder window will open showing the Heimdal Desktop application

3. **Install the Application**
   - Drag the Heimdal Desktop icon to the Applications folder
   - Wait for the copy to complete
   - Eject the DMG

4. **First Launch**
   - Open Applications folder
   - Right-click Heimdal Desktop and select "Open" (first time only)
   - If macOS shows "unidentified developer" warning, click "Open"
   - Alternatively, go to System Preferences → Security & Privacy and click "Open Anyway"

5. **Grant Permissions**
   - macOS will prompt for various permissions
   - Grant Full Disk Access: System Preferences → Security & Privacy → Privacy → Full Disk Access
   - Add Heimdal Desktop to the list and enable it
   - You may need to run with sudo on first launch:
     ```bash
     sudo /Applications/Heimdal\ Desktop.app/Contents/MacOS/heimdal-desktop
     ```

### Linux Installation

#### Debian/Ubuntu

1. **Download the Package**
   ```bash
   wget https://releases.heimdal.io/heimdal-desktop-linux-amd64.deb
   ```

2. **Install the Package**
   ```bash
   sudo dpkg -i heimdal-desktop-linux-amd64.deb
   sudo apt-get install -f  # Install any missing dependencies
   ```

3. **Grant Capabilities**
   ```bash
   sudo setcap cap_net_raw,cap_net_admin=eip /usr/bin/heimdal-desktop
   ```

4. **Launch the Application**
   ```bash
   heimdal-desktop
   ```
   Or launch from your application menu

#### Fedora/RHEL

1. **Download the Package**
   ```bash
   wget https://releases.heimdal.io/heimdal-desktop-linux-amd64.rpm
   ```

2. **Install the Package**
   ```bash
   sudo rpm -i heimdal-desktop-linux-amd64.rpm
   ```

3. **Install Dependencies**
   ```bash
   sudo dnf install libpcap-devel
   ```

4. **Grant Capabilities**
   ```bash
   sudo setcap cap_net_raw,cap_net_admin=eip /usr/bin/heimdal-desktop
   ```

5. **Launch the Application**
   ```bash
   heimdal-desktop
   ```

## First Launch and Onboarding

When you launch Heimdal Desktop for the first time, you'll be guided through an onboarding wizard.

### Step 1: Welcome Screen

The welcome screen explains what Heimdal Desktop does and what permissions it needs.

- Read the overview
- Click "Get Started" to continue

### Step 2: Permission Check

Heimdal Desktop will check for required permissions:

**Windows:**
- Administrator rights
- Npcap installation

**macOS:**
- Full Disk Access
- Packet capture permissions

**Linux:**
- CAP_NET_RAW capability
- CAP_NET_ADMIN capability
- libpcap installation

If any permissions are missing, the wizard will provide instructions for granting them.

### Step 3: Network Interface Selection

Heimdal Desktop will automatically detect your network interfaces.

- The wizard will show all available network interfaces
- The primary interface (with internet connectivity) will be pre-selected
- You can change the selection if needed
- Click "Continue" to proceed

**Tip**: Choose the interface that connects to your router (usually Ethernet or Wi-Fi).

### Step 4: Configuration

Configure basic settings:

- **Auto-Start**: Choose whether Heimdal Desktop should start automatically when you log in
- **Cloud Sync**: Optionally enable cloud synchronization (requires account)
- **Notifications**: Choose whether to receive desktop notifications for events

### Step 5: Complete Setup

- Review your settings
- Click "Start Monitoring" to begin
- The onboarding wizard will close
- Heimdal Desktop will appear in your system tray/menu bar
- The dashboard will open in your default browser

## Using the Dashboard

The Heimdal Desktop dashboard is a web-based interface that runs locally on your computer.

### Accessing the Dashboard

- **From System Tray**: Click the Heimdal icon and select "Open Dashboard"
- **From Browser**: Navigate to `http://localhost:8080`
- **Keyboard Shortcut**: The system tray menu may show a keyboard shortcut

### Dashboard Overview

The dashboard consists of several sections:

#### Network Map

The network map shows all discovered devices and their connections.

- **Nodes**: Each circle represents a device on your network
- **Lines**: Lines between nodes show active communication
- **Colors**: Colors indicate device type (computer, mobile, IoT, etc.)
- **Size**: Node size may indicate traffic volume

**Interactions:**
- Click a device to see details
- Hover over a device to see quick info
- Drag devices to rearrange the map
- Zoom in/out with mouse wheel or pinch gesture

#### Device List

The device list shows all discovered devices in a table format.

**Columns:**
- **Name**: Device hostname or vendor name
- **IP Address**: Current IP address
- **MAC Address**: Hardware address
- **Type**: Device category (Computer, Mobile, IoT, etc.)
- **Status**: Active or Inactive
- **First Seen**: When the device was first discovered
- **Last Seen**: Most recent activity

**Actions:**
- Click a row to see device details
- Sort by clicking column headers
- Filter using the search box
- Export the list (Pro tier)

#### Device Details

Click any device to see detailed information:

- **Basic Info**: Name, IP, MAC, vendor
- **Activity**: Traffic volume, packet counts
- **Behavioral Profile**: Communication patterns
  - Top destinations
  - Common ports
  - Protocol distribution
  - Hourly activity chart
- **Anomalies**: Any detected unusual behavior
- **History**: Timeline of device activity

#### Traffic View

The traffic view shows real-time network activity.

- **Live Feed**: Scrolling list of recent packets
- **Traffic Graph**: Real-time chart of network throughput
- **Protocol Breakdown**: Pie chart of protocol distribution
- **Top Talkers**: Devices with most traffic

#### Anomalies

The anomalies section shows detected unusual behavior.

**Anomaly Types:**
- **New Device**: A device appeared on the network
- **Unexpected Destination**: Device contacted an unusual IP
- **Unusual Port**: Device used an uncommon port
- **Traffic Spike**: Sudden increase in traffic volume

**Anomaly Details:**
- Severity (Low, Medium, High, Critical)
- Description of the anomaly
- Affected device
- Timestamp
- Evidence and context

**Actions:**
- Mark as false positive
- Investigate further
- Block device (Pro tier)

#### Settings

Access dashboard settings from the gear icon.

**Available Settings:**
- **General**: Dashboard refresh rate, theme
- **Notifications**: Configure alert preferences
- **Feature Tier**: View current tier and upgrade options
- **Cloud**: Configure cloud synchronization
- **Advanced**: Expert settings for power users

### Real-Time Updates

The dashboard updates in real-time using WebSocket connections.

- New devices appear automatically
- Traffic flows update live
- Anomalies appear as they're detected
- No need to refresh the page

**Connection Status**: Look for the connection indicator in the top-right corner:
- Green: Connected and receiving updates
- Yellow: Reconnecting
- Red: Disconnected (check that Heimdal Desktop is running)

## System Tray Features

Heimdal Desktop runs in the background and provides quick access through the system tray (Windows/Linux) or menu bar (macOS).

### System Tray Icon

The icon changes to indicate status:

- **Green**: Monitoring active, no issues
- **Yellow**: Monitoring paused or warning
- **Red**: Error or monitoring stopped
- **Animated**: Processing or updating

### System Tray Menu

Right-click (Windows/Linux) or click (macOS) the icon to access the menu:

- **Open Dashboard**: Opens the web dashboard in your browser
- **Pause Monitoring**: Temporarily stops network monitoring
- **Resume Monitoring**: Resumes monitoring after pause
- **Settings**: Opens the settings dialog
- **About**: Shows version and license information
- **Quit**: Exits Heimdal Desktop

### Desktop Notifications

Heimdal Desktop can show desktop notifications for important events:

- **New Device Detected**: When a new device joins the network
- **Anomaly Detected**: When unusual behavior is identified
- **Connection Issues**: When cloud sync or monitoring encounters problems

**Configuring Notifications:**
- Access notification settings from the system tray menu or dashboard
- Choose which events trigger notifications
- Set notification priority levels
- Enable/disable sound alerts

## Feature Tiers

Heimdal Desktop offers three subscription tiers with different feature sets.

### Free Tier

The Free tier provides basic network visibility at no cost.

**Included Features:**
- Network device discovery
- Real-time traffic monitoring
- Basic behavioral profiling
- Local dashboard access
- Desktop notifications
- Up to 50 devices

**Limitations:**
- Read-only (no traffic blocking)
- No cloud synchronization
- No advanced filtering
- Community support only

### Pro Tier

The Pro tier adds advanced security features for power users.

**All Free Features, Plus:**
- Active traffic blocking
- Advanced filtering rules
- Cloud synchronization
- Unlimited devices
- Historical data retention (90 days)
- Priority email support
- API access for automation

**Pricing**: $9.99/month or $99/year

### Enterprise Tier

The Enterprise tier is designed for businesses and advanced users.

**All Pro Features, Plus:**
- Multi-device management
- Centralized dashboard for multiple sensors
- Custom integrations
- Extended data retention (1 year)
- Advanced reporting
- Dedicated support
- SLA guarantees

**Pricing**: Contact sales for custom pricing

### Upgrading Your Tier

To upgrade from Free to Pro or Enterprise:

1. Open the dashboard
2. Go to Settings → Feature Tier
3. Click "Upgrade"
4. Choose your desired tier
5. Enter payment information
6. Enter the license key you receive
7. Restart Heimdal Desktop

Your new features will be available immediately after restart.

## Configuration

Heimdal Desktop stores its configuration in a platform-specific location.

### Configuration File Location

**Windows**: `%APPDATA%\Heimdal\config.json`
- Typically: `C:\Users\<username>\AppData\Roaming\Heimdal\config.json`

**macOS**: `~/Library/Application Support/Heimdal/config.json`

**Linux**: `~/.config/heimdal/config.json`

### Configuration Format

The configuration file is in JSON format. Here's an example:

```json
{
  "database": {
    "path": "~/.local/share/heimdal/db",
    "sync_writes": true
  },
  "discovery": {
    "arp_interval": "30s",
    "mdns_enabled": true
  },
  "interceptor": {
    "enabled": true,
    "interface": "en0"
  },
  "profiler": {
    "persist_interval": "5m"
  },
  "cloud": {
    "enabled": false,
    "provider": "aws",
    "region": "us-west-2"
  },
  "api": {
    "port": 8080,
    "enable_cors": false
  },
  "desktop": {
    "feature_gate": {
      "tier": "free",
      "license_key": ""
    },
    "system_tray": {
      "show_notifications": true,
      "notification_level": "medium"
    },
    "auto_start": false,
    "update_check": true
  }
}
```

### Key Configuration Options

#### Database Settings

- `path`: Where to store the local database
- `sync_writes`: Whether to sync writes to disk immediately (slower but safer)

#### Discovery Settings

- `arp_interval`: How often to scan for devices (e.g., "30s", "1m")
- `mdns_enabled`: Whether to use mDNS for device discovery

#### Interceptor Settings

- `enabled`: Whether to enable ARP spoofing
- `interface`: Network interface to monitor (auto-detected if not specified)

#### API Settings

- `port`: Port for the web dashboard (default: 8080)
- `enable_cors`: Whether to enable CORS (for development)

#### Desktop Settings

- `feature_gate.tier`: Your subscription tier ("free", "pro", "enterprise")
- `feature_gate.license_key`: Your license key (for Pro/Enterprise)
- `system_tray.show_notifications`: Enable/disable notifications
- `auto_start`: Start automatically on system boot
- `update_check`: Check for updates automatically

### Editing Configuration

**Option 1: Dashboard Settings**
- Most settings can be changed through the dashboard
- Go to Settings in the dashboard
- Changes are saved automatically

**Option 2: Manual Editing**
- Close Heimdal Desktop
- Edit the configuration file with a text editor
- Ensure valid JSON syntax
- Save the file
- Restart Heimdal Desktop

**Warning**: Invalid JSON will prevent Heimdal Desktop from starting. Always validate your JSON before saving.

### Configuration Hot-Reload

Some configuration changes can be applied without restarting:

- Discovery intervals
- Notification settings
- Dashboard refresh rate

Other changes require a restart:

- Network interface
- Database path
- API port
- Feature tier

## Troubleshooting

### Common Issues

#### Heimdal Desktop Won't Start

**Symptoms**: Application doesn't launch or crashes immediately

**Solutions**:
1. Check that you have required permissions (see Installation section)
2. Verify configuration file is valid JSON
3. Check logs for error messages:
   - Windows: `%APPDATA%\Heimdal\logs\`
   - macOS: `~/Library/Logs/Heimdal/`
   - Linux: `~/.local/share/heimdal/logs/`
4. Try running from command line to see error messages
5. Reinstall the application

#### No Devices Discovered

**Symptoms**: Dashboard shows no devices or only your computer

**Solutions**:
1. Verify you're on the same network as other devices
2. Check that the correct network interface is selected
3. Ensure ARP spoofing is enabled in configuration
4. Check firewall isn't blocking ARP packets
5. Wait a few minutes - discovery takes time
6. Try manually triggering a scan from the dashboard

#### Dashboard Won't Load

**Symptoms**: Browser shows "Connection refused" or "Cannot connect"

**Solutions**:
1. Verify Heimdal Desktop is running (check system tray)
2. Check that port 8080 isn't used by another application
3. Try accessing via `http://localhost:8080` instead of `http://127.0.0.1:8080`
4. Check firewall settings
5. Look for errors in the application logs
6. Try restarting Heimdal Desktop

#### High CPU Usage

**Symptoms**: Computer is slow, fan running constantly

**Solutions**:
1. Check how many devices are on your network
2. Reduce discovery scan frequency in configuration
3. Disable mDNS if not needed
4. Check for packet capture rate limits
5. Close and reopen the dashboard (browser may be consuming resources)
6. Consider upgrading your computer's RAM

#### Permission Errors

**Windows**: "Access Denied" or "Administrator rights required"
- Right-click Heimdal Desktop and select "Run as Administrator"
- Check that Npcap is installed correctly

**macOS**: "Operation not permitted"
- Grant Full Disk Access in System Preferences
- Run with sudo: `sudo /Applications/Heimdal\ Desktop.app/Contents/MacOS/heimdal-desktop`

**Linux**: "Capability error" or "Permission denied"
- Grant capabilities: `sudo setcap cap_net_raw,cap_net_admin=eip /usr/bin/heimdal-desktop`
- Check that libpcap is installed

#### System Tray Icon Missing

**Windows**:
- Check notification area settings
- Ensure Heimdal Desktop is allowed to show in system tray
- Try restarting Windows Explorer

**macOS**:
- Check menu bar settings
- Ensure the application has accessibility permissions
- Try restarting Heimdal Desktop

**Linux**:
- Ensure your desktop environment supports system tray
- Try a different desktop environment
- Access dashboard directly via browser if tray is unavailable

### Getting Help

If you can't resolve your issue:

1. **Check the FAQ** (see below)
2. **Review the logs** for error messages
3. **Search the documentation** for your specific error
4. **Contact Support**:
   - Free tier: Community forums
   - Pro tier: Email support (support@heimdal.io)
   - Enterprise tier: Dedicated support channel

When contacting support, include:
- Operating system and version
- Heimdal Desktop version
- Description of the problem
- Steps to reproduce
- Relevant log excerpts
- Screenshots if applicable

## FAQ

### General Questions

**Q: Is Heimdal Desktop safe to use?**
A: Yes, Heimdal Desktop is safe when used on networks you own or have permission to monitor. It uses ARP spoofing, which is a standard network monitoring technique. However, it should not be used on networks without authorization.

**Q: Does Heimdal Desktop slow down my network?**
A: Heimdal Desktop has minimal impact on network performance. It passively monitors traffic and doesn't interfere with normal network operations. You may notice a slight increase in ARP traffic, but this is negligible.

**Q: Can I run Heimdal Desktop on multiple computers?**
A: Yes, but each instance will only monitor traffic it can see. For best results, run it on a computer that's always on and connected to your network. Enterprise tier supports centralized management of multiple instances.

**Q: Does Heimdal Desktop work with VPNs?**
A: Heimdal Desktop monitors your local network, not VPN traffic. If you're connected to a VPN, it will monitor devices on your local network but won't see traffic going through the VPN tunnel.

**Q: What data does Heimdal Desktop collect?**
A: Heimdal Desktop collects network metadata (IP addresses, MAC addresses, ports, protocols) and stores it locally on your computer. With cloud sync enabled, this data is transmitted to Heimdal's cloud platform. No packet payloads or personal data are collected.

### Technical Questions

**Q: What's the difference between Heimdal Hardware and Heimdal Desktop?**
A: Heimdal Hardware is a dedicated Raspberry Pi device that monitors your entire network from the router level. Heimdal Desktop is software that runs on your computer and monitors from the host level. Hardware provides better coverage, while Desktop is easier to set up.

**Q: Can I use Heimdal Desktop and Heimdal Hardware together?**
A: Yes! They can work together and provide complementary coverage. Use the Enterprise tier to manage both from a single dashboard.

**Q: How much disk space does Heimdal Desktop use?**
A: Heimdal Desktop typically uses 100-500 MB depending on how many devices you have and how long you retain data. The database grows over time but is automatically pruned based on your tier's retention policy.

**Q: Can I export my data?**
A: Yes, Pro and Enterprise tiers can export device lists and reports. Free tier has limited export capabilities.

**Q: Does Heimdal Desktop support IPv6?**
A: Currently, Heimdal Desktop focuses on IPv4 networks. IPv6 support is planned for a future release.

**Q: Can I run Heimdal Desktop on a server without a GUI?**
A: Yes, Heimdal Desktop can run headless. The system tray won't be available, but you can access the dashboard via browser. This is common on Linux servers.

### Billing and Licensing

**Q: How do I upgrade from Free to Pro?**
A: Go to Settings → Feature Tier in the dashboard and click "Upgrade". You'll be directed to the payment page. After payment, you'll receive a license key to enter in the application.

**Q: Can I try Pro features before buying?**
A: Yes, we offer a 14-day free trial of Pro tier. No credit card required. Go to Settings → Feature Tier and click "Start Trial".

**Q: What happens if my Pro subscription expires?**
A: Your account will revert to Free tier. You'll lose access to Pro features (blocking, cloud sync, etc.) but your data will be retained. You can re-subscribe at any time to regain access.

**Q: Can I get a refund?**
A: Yes, we offer a 30-day money-back guarantee. Contact support@heimdal.io to request a refund.

**Q: Do you offer educational or non-profit discounts?**
A: Yes! Contact sales@heimdal.io with proof of educational or non-profit status for special pricing.

### Privacy and Security

**Q: Is my data encrypted?**
A: Yes, all data stored locally is encrypted at rest. Cloud transmissions use TLS encryption. Your license key and credentials are stored securely in your system's keychain.

**Q: Do you sell my data?**
A: No, we never sell user data. Your network data is yours. We only use aggregated, anonymized data for product improvement.

**Q: Can I use Heimdal Desktop offline?**
A: Yes, Heimdal Desktop works completely offline. Cloud sync is optional and can be disabled.

**Q: What happens to my data if I uninstall?**
A: Your local database is not automatically deleted. You can manually delete it from the configuration directory if desired. Cloud data (if enabled) remains in your account and can be deleted from the web portal.

---

## Need More Help?

- **Documentation**: https://docs.heimdal.io
- **Community Forums**: https://community.heimdal.io
- **Email Support**: support@heimdal.io (Pro/Enterprise)
- **Sales**: sales@heimdal.io

Thank you for using Heimdal Desktop!
