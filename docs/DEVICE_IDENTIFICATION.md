# Device Identification System

Heimdal uses multiple methods to identify and classify network devices, providing rich context for behavioral analysis and anomaly detection.

## Identification Methods

### 1. OUI Vendor Lookup

**What it is:** The first 24 bits (3 bytes) of every MAC address identify the manufacturer (OUI - Organizationally Unique Identifier).

**How it works:**
- Embedded IEEE OUI database with 38,400+ vendor entries (6MB)
- Instant O(1) lookup from MAC address to vendor name
- Provides both short name (e.g., "Apple") and full manufacturer name (e.g., "Apple, Inc.")

**Example:**
```
MAC: e0:4f:43:a0:f7:73
→ Vendor: "Apple"
→ Manufacturer: "Apple, Inc."
```

### 2. Device Classification

**What it is:** Automatic categorization of devices into types (Phone, Computer, IoT, etc.)

**Classification signals:**
- **Vendor patterns**: Apple → Phone/Computer, Cisco → Router
- **Hostname patterns**: "iPhone" → Phone, "raspberrypi" → IoT
- **mDNS services**: `_airplay._tcp` → Streaming, `_printer._tcp` → Printer
- **Weighted confidence**: Combines multiple signals with confidence scoring

**Device Types:**
- Endpoint: Phone, Tablet, Computer, Laptop, Wearable
- Infrastructure: Router, Switch, Server, NAS
- IoT: Generic IoT, Smart Home, Camera, Speaker
- Peripheral: Printer, Scanner
- Entertainment: TV, Streaming Device, Game Console

**Example:**
```
Vendor: "Apple" + Hostname: "Johns-iPhone" + Service: "_airplay._tcp"
→ Type: "phone"
→ Category: "endpoint"
→ Confidence: 0.85
```

### 3. Hostname Resolution

**What it is:** Discovering the human-readable name of a device.

**Methods (in priority order):**
1. **mDNS name**: From mDNS/Bonjour announcements (highest priority)
2. **Reverse DNS**: PTR record lookup via system resolver
3. **NetBIOS**: Windows network name query (planned)

**Features:**
- Async resolution with 2s timeout per method
- Background enrichment for existing devices
- Fallback chain ensures best available name

**Example:**
```
IP: 10.100.102.115
→ mDNS: "Moshe's MacBook Pro"
→ Reverse DNS: "macbook-pro.local"
→ Final: "Moshe's MacBook Pro" (mDNS preferred)
```

### 4. Service Discovery (mDNS)

**What it is:** Discovering what services a device offers.

**Detected services:**
- Media: `_airplay._tcp`, `_googlecast._tcp`, `_spotify-connect._tcp`
- Printing: `_printer._tcp`, `_ipp._tcp`
- Smart Home: `_hap._tcp` (HomeKit), `_matter._tcp`
- File Sharing: `_smb._tcp`, `_afpovertcp._tcp`
- Workstations: `_workstation._tcp`, `_ssh._tcp`

**Usage:**
- Improves device classification accuracy
- Helps identify device purpose
- Useful for network inventory

## Data Flow

```
Device Discovered (ARP)
  ↓
MAC Address
  ↓
OUI Lookup → Vendor + Manufacturer
  ↓
mDNS Scan → Services + Name
  ↓
Hostname Resolution → Hostname
  ↓
Classification → Device Type + Confidence
  ↓
Save to Database
```

## Privacy Considerations

### Local-First
- All identification happens locally
- No external API calls
- Embedded 6MB OUI database
- Works completely offline

### Cloud Telemetry (Optional)
When cloud is enabled, you control what's shared:
- `send_device_info`: Share device discovery data (helps build device database)
- `send_profiles`: Share behavioral profiles (helps ML models)
- `anonymize_data`: Hash sensitive fields before sending
- `send_diagnostics`: Share diagnostic telemetry

### Anonymization
When enabled, the following fields are hashed:
- Device names
- Hostnames  
- IP addresses (last octet only)
- MAC addresses (vendor prefix preserved)

## Configuration

### Enable/Disable Methods

```json
{
  "discovery": {
    "mdns_enabled": true,
    "scan_interval_seconds": 60
  }
}
```

### Privacy Controls

```json
{
  "cloud": {
    "enabled": true,
    "send_device_info": true,
    "send_profiles": true,
    "send_anomalies": false,
    "anonymize_data": true,
    "send_diagnostics": false
  }
}
```

## Performance

- **OUI Lookup**: <1µs per MAC address
- **Classification**: <10µs per device
- **Hostname Resolution**: <2s per device (async, non-blocking)
- **mDNS Scan**: ~10s every 5 minutes
- **Memory**: ~6MB for OUI database + ~1KB per device

## Future Enhancements

### Planned Features
1. **DHCP Fingerprinting**: OS detection from DHCP options
2. **SSDP/UPnP Discovery**: Enhanced IoT device identification
3. **Port Scanning**: Service identification via open ports
4. **Banner Grabbing**: Service version detection
5. **Machine Learning**: Device classification model training

### Community Contributions
- Custom classification rules
- Additional service types
- Regional OUI databases
- Device icon packs

## Troubleshooting

### Devices show as "Unknown"
- **Cause**: MAC address not in OUI database (rare/custom hardware)
- **Solution**: Device will be classified by hostname/services if available

### No hostnames resolved
- **Cause**: DNS not configured or devices don't respond to reverse DNS
- **Solution**: mDNS names will be used when available

### Classification confidence low
- **Cause**: Limited signals (no hostname, no services, generic vendor)
- **Solution**: Classification improves as more data is gathered

### mDNS not discovering services
- **Cause**: Firewall blocking multicast or devices not advertising
- **Solution**: Check firewall rules for UDP port 5353

## Related Documentation

- [ARCHITECTURE.md](../ARCHITECTURE.md) - System architecture
- [CONFIG.md](../CONFIG.md) - Configuration reference
- [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) - Contributing to device identification

