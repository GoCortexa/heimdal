# Design Document

## Overview

Heimdal 2.0 is a Go-based network security sensor designed for zero-touch deployment on Raspberry Pi hardware. The system uses a concurrent, goroutine-based architecture to perform network discovery, traffic interception via ARP spoofing, behavioral profiling, and local web-based monitoring. The application compiles to a single statically-linked binary and is deployed exclusively via Ansible.

### Design Principles

1. **Zero-Touch Operation**: Automatic network detection and service startup with no manual configuration
2. **Concurrency-First**: All major components run as independent goroutines communicating via channels
3. **Resilience**: Automatic recovery from component failures without full application restart
4. **Resource Efficiency**: Optimized for Raspberry Pi constraints (< 200MB RAM, < 25% CPU)
5. **Single Binary**: Statically compiled with no external runtime dependencies
6. **Local-First**: Full functionality without cloud connectivity; cloud integration is optional

## Architecture

### High-Level Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Main Orchestrator                        │
│  - Configuration Management                                      │
│  - Component Lifecycle                                           │
│  - Graceful Shutdown (SIGTERM/SIGINT)                           │
└────────────┬────────────────────────────────────────────────────┘
             │
             ├──────────────┬──────────────┬──────────────┬────────┤
             ▼              ▼              ▼              ▼        ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
    │ Network  │  │  Device  │  │ Traffic  │  │  Packet  │  │Behavioral│
    │AutoConfig│  │Discovery │  │Intercept │  │ Analyzer │  │ Profiler │
    └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
         │             │             │             │             │
         └─────────────┴─────────────┴─────────────┴─────────────┘
                                     │
                         ┌───────────┴───────────┐
                         ▼                       ▼
                  ┌─────────────┐        ┌─────────────┐
                  │  Database   │        │   Web API   │
                  │  (BadgerDB) │        │  Dashboard  │
                  └─────────────┘        └─────────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │   Cloud     │
                  │ Connector   │
                  │ (Optional)  │
                  └─────────────┘
```

### Component Communication

Components communicate via typed Go channels:
- **deviceChan**: Discovered devices (Device struct)
- **packetChan**: Analyzed packet metadata (PacketInfo struct)
- **profileChan**: Updated behavioral profiles (BehavioralProfile struct)
- **shutdownChan**: Graceful shutdown signal (empty struct)

## Components and Interfaces

### 1. Main Orchestrator

**Location**: `cmd/heimdal/main.go`, `internal/orchestrator/orchestrator.go`

**Responsibilities**:
- Load configuration from `/etc/heimdal/config.json`
- Initialize all components in correct order
- Start each component as a goroutine
- Handle OS signals (SIGTERM, SIGINT) for graceful shutdown
- Coordinate shutdown sequence with 5-second timeout

**Key Structures**:
```go
type Orchestrator struct {
    config      *Config
    db          *database.DatabaseManager
    components  []Component
    shutdownCh  chan struct{}
    wg          sync.WaitGroup
}

type Component interface {
    Start(ctx context.Context) error
    Stop() error
    Name() string
}
```

**Startup Sequence**:
1. Load configuration
2. Initialize DatabaseManager
3. Initialize Network Auto-Config (blocking until network detected)
4. Start Device Discovery
5. Start Traffic Interceptor
6. Start Packet Analyzer
7. Start Behavioral Profiler
8. Start Web API
9. Start Cloud Connector (if enabled)

### 2. Network Auto-Config Component

**Location**: `internal/netconfig/autoconfig.go`

**Responsibilities**:
- Detect primary network interface (eth0, wlan0)
- Determine gateway IP via routing table
- Calculate subnet mask and CIDR notation
- Provide network configuration to other components

**Technology**:
- Standard library `net` package
- Parse `/proc/net/route` for gateway detection
- Use `net.InterfaceAddrs()` for interface enumeration

**Key Structures**:
```go
type NetworkConfig struct {
    Interface   string
    LocalIP     net.IP
    Gateway     net.IP
    Subnet      *net.IPNet
    CIDR        string
}

type AutoConfig struct {
    config *NetworkConfig
    mu     sync.RWMutex
}
```

**Operation**:
- Runs once at startup (blocking)
- Retries every 5 seconds until network is detected
- Provides read-only access to network configuration

### 3. Device Discovery Component

**Location**: `internal/discovery/scanner.go`

**Responsibilities**:
- Continuously scan local network for devices
- Perform ARP scanning for IP/MAC discovery
- Perform mDNS/DNS-SD scanning for device names
- Send discovered devices to deviceChan
- Update database with new/changed devices

**Technology**:
- **ARP Scanning**: `github.com/google/gopacket` with ARP layer
- **mDNS Discovery**: `github.com/hashicorp/mdns` for DNS-SD service discovery
- **Alternative**: `github.com/grandcat/zeroconf` for pure Go mDNS

**Key Structures**:
```go
type Device struct {
    MAC         string    `json:"mac"`
    IP          string    `json:"ip"`
    Name        string    `json:"name"`
    Vendor      string    `json:"vendor"`
    FirstSeen   time.Time `json:"first_seen"`
    LastSeen    time.Time `json:"last_seen"`
    IsActive    bool      `json:"is_active"`
}

type Scanner struct {
    netConfig   *netconfig.NetworkConfig
    db          *database.DatabaseManager
    deviceChan  chan<- Device
    scanInterval time.Duration
}
```

**Operation**:
- ARP scan every 60 seconds across entire subnet
- mDNS passive listening for service announcements
- Active mDNS query every 5 minutes
- Mark devices inactive if not seen for 5 minutes
- Persist to database immediately on discovery/update

### 4. Traffic Interceptor Component (ARP Spoofer)

**Location**: `internal/interceptor/arp_spoofer.go`

**Responsibilities**:
- Perform ARP spoofing to intercept traffic
- Maintain ARP spoofing for all discovered devices
- Forward packets to maintain connectivity
- Automatically recover from failures
- Monitor spoofing health

**Technology**:
- **Packet Crafting**: `github.com/google/gopacket` with layers.ARP
- **Raw Socket**: `github.com/google/gopacket/pcap` or `github.com/google/gopacket/afpacket`
- **Requires**: CAP_NET_RAW and CAP_NET_ADMIN capabilities

**Key Structures**:
```go
type ARPSpoofer struct {
    netConfig   *netconfig.NetworkConfig
    handle      *pcap.Handle
    targets     map[string]*SpoofTarget
    mu          sync.RWMutex
    deviceChan  <-chan Device
}

type SpoofTarget struct {
    MAC         net.HardwareAddr
    IP          net.IP
    LastSpoof   time.Time
    IsActive    bool
}
```

**Operation**:
- Listen on deviceChan for new devices to spoof
- Send ARP replies every 2 seconds to each target
- Send spoofed packets to both target device and gateway
- Monitor for failures and restart spoofing if needed
- Implement exponential backoff on repeated failures

**Safety Mechanisms**:
- Verify IP forwarding is enabled before starting
- Graceful cleanup: restore original ARP tables on shutdown
- Health check: verify packets are being forwarded

### 5. Packet Analyzer Component

**Location**: `internal/analyzer/sniffer.go`

**Responsibilities**:
- Capture packets from network interface
- Extract relevant metadata (src MAC, dst IP, dst port, protocol)
- Filter out noise (broadcast, multicast, local traffic)
- Send packet metadata to packetChan
- Implement rate limiting to prevent overload

**Technology**:
- **Packet Capture**: `github.com/google/gopacket/pcap`
- **Packet Parsing**: `github.com/google/gopacket/layers`

**Key Structures**:
```go
type PacketInfo struct {
    Timestamp   time.Time
    SrcMAC      string
    DstIP       string
    DstPort     uint16
    Protocol    string
    Size        uint32
}

type Sniffer struct {
    netConfig   *netconfig.NetworkConfig
    handle      *pcap.Handle
    packetChan  chan<- PacketInfo
    rateLimiter *rate.Limiter
}
```

**Operation**:
- Open interface in promiscuous mode
- Apply BPF filter to reduce noise: `not broadcast and not multicast`
- Parse Ethernet, IP, TCP/UDP layers
- Extract metadata and send to packetChan
- Rate limit to 10,000 packets/second
- Use buffered channel (size 1000) to prevent blocking

**Optimization**:
- Zero-copy packet processing where possible
- Batch processing for high-traffic scenarios
- Drop packets if channel is full (non-blocking send)

### 6. Behavioral Profiler Component

**Location**: `internal/profiler/profiler.go`

**Responsibilities**:
- Receive packet metadata from packetChan
- Aggregate data per MAC address
- Build behavioral profiles (destinations, ports, timing)
- Persist profiles to database every 60 seconds
- Detect anomalies (future enhancement)

**Key Structures**:
```go
type BehavioralProfile struct {
    MAC             string                 `json:"mac"`
    Destinations    map[string]*DestInfo   `json:"destinations"`
    Ports           map[uint16]int         `json:"ports"`
    Protocols       map[string]int         `json:"protocols"`
    TotalPackets    int64                  `json:"total_packets"`
    TotalBytes      int64                  `json:"total_bytes"`
    FirstSeen       time.Time              `json:"first_seen"`
    LastSeen        time.Time              `json:"last_seen"`
    HourlyActivity  [24]int                `json:"hourly_activity"`
}

type DestInfo struct {
    IP          string
    Count       int64
    LastSeen    time.Time
}

type Profiler struct {
    profiles    map[string]*BehavioralProfile
    mu          sync.RWMutex
    packetChan  <-chan PacketInfo
    db          *database.DatabaseManager
    persistTicker *time.Ticker
}
```

**Operation**:
- Load existing profiles from database on startup
- Receive packets from packetChan
- Update in-memory profile for source MAC
- Track: destination IPs, ports, protocols, timing, volume
- Persist all profiles to database every 60 seconds
- Implement sliding window for hourly activity (24-hour)

**Aggregation Logic**:
- Increment destination counter for each unique dst IP
- Track port usage frequency
- Update hourly activity based on packet timestamp
- Calculate total packets and bytes

### 7. Database Component (BadgerDB)

**Location**: `internal/database/badger_db.go`

**Responsibilities**:
- Provide persistent storage for Device Map and Behavioral Profiles
- Implement key-value storage with prefixed keys
- Handle serialization/deserialization (JSON)
- Provide transaction support
- Implement in-memory buffering for unavailable database

**Technology**:
- **BadgerDB**: `github.com/dgraph-io/badger/v4`
- Pure Go, embedded, LSM-tree based
- Optimized for high write throughput
- Excellent concurrency support
- No C dependencies

**Key Schema**:
```
device:<MAC_ADDRESS>   → JSON-serialized Device struct
profile:<MAC_ADDRESS>  → JSON-serialized BehavioralProfile struct
meta:config            → System metadata
```

**Key Structures**:
```go
type DatabaseManager struct {
    db          *badger.DB
    path        string
    buffer      *MemoryBuffer
    mu          sync.RWMutex
}

type MemoryBuffer struct {
    devices     map[string]*Device
    profiles    map[string]*BehavioralProfile
    maxSize     int
    mu          sync.RWMutex
}
```

**Interface**:
```go
// Device operations
func (dm *DatabaseManager) SaveDevice(device *Device) error
func (dm *DatabaseManager) GetDevice(mac string) (*Device, error)
func (dm *DatabaseManager) GetAllDevices() ([]*Device, error)
func (dm *DatabaseManager) DeleteDevice(mac string) error

// Profile operations
func (dm *DatabaseManager) SaveProfile(profile *BehavioralProfile) error
func (dm *DatabaseManager) GetProfile(mac string) (*BehavioralProfile, error)
func (dm *DatabaseManager) GetAllProfiles() ([]*BehavioralProfile, error)

// Batch operations
func (dm *DatabaseManager) SaveDeviceBatch(devices []*Device) error
func (dm *DatabaseManager) SaveProfileBatch(profiles []*BehavioralProfile) error

// Lifecycle
func NewDatabaseManager(path string) (*DatabaseManager, error)
func (dm *DatabaseManager) Close() error
```

**Operation**:
- Initialize BadgerDB at `/var/lib/heimdal/db`
- Use JSON encoding for human-readable storage
- Implement write-through cache for frequently accessed data
- Buffer up to 1000 records in memory if database unavailable
- Flush buffer when database becomes available
- Run garbage collection every 5 minutes

**Error Handling**:
- Retry failed writes up to 3 times with exponential backoff
- Fall back to memory buffer if database is unavailable
- Log all database errors for debugging
- Gracefully handle corrupted data (skip and log)

### 8. Web API and Dashboard Component

**Location**: `internal/api/server.go`, `web/dashboard/`

**Responsibilities**:
- Serve REST API for device and profile data
- Serve static HTML/CSS/JS dashboard
- Provide real-time updates (optional: WebSocket)
- Handle CORS for local network access

**Technology**:
- **HTTP Server**: Standard library `net/http`
- **Router**: `github.com/gorilla/mux` (lightweight)
- **Frontend**: Vanilla JavaScript with minimal dependencies

**API Endpoints**:
```
GET  /api/v1/devices              → List all devices
GET  /api/v1/devices/:mac         → Get device details
GET  /api/v1/profiles/:mac        → Get behavioral profile
GET  /api/v1/stats                → System statistics
GET  /api/v1/health               → Health check
GET  /                            → Dashboard HTML
```

**Key Structures**:
```go
type APIServer struct {
    db          *database.DatabaseManager
    router      *mux.Router
    server      *http.Server
    port        int
}

type DeviceResponse struct {
    Devices []*Device `json:"devices"`
    Count   int       `json:"count"`
}

type StatsResponse struct {
    TotalDevices    int       `json:"total_devices"`
    ActiveDevices   int       `json:"active_devices"`
    TotalPackets    int64     `json:"total_packets"`
    Uptime          string    `json:"uptime"`
    LastUpdate      time.Time `json:"last_update"`
}
```

**Dashboard Features**:
- Device list table with MAC, IP, Name, Status
- Click device to view behavioral profile
- Visual representation of top destinations
- Port usage chart
- Activity timeline (24-hour)
- Auto-refresh every 10 seconds

**Security**:
- Listen only on local network interface (not 0.0.0.0)
- No authentication required (local network trust model)
- Rate limiting: 100 requests/minute per IP
- Input validation on all endpoints

### 9. Cloud Connector Component (Optional)

**Location**: `internal/cloud/connector.go`, `internal/cloud/aws/`, `internal/cloud/gcp/`

**Responsibilities**:
- Define interface for cloud connectivity
- Provide stub implementations for AWS IoT and Google Cloud
- Transmit behavioral profiles to cloud platform
- Handle connection failures gracefully
- Respect enable/disable configuration

**Technology**:
- **AWS IoT**: `github.com/aws/aws-sdk-go-v2` with MQTT
- **Google Cloud**: `cloud.google.com/go/pubsub`

**Interface**:
```go
type CloudConnector interface {
    Connect() error
    Disconnect() error
    SendProfile(profile *BehavioralProfile) error
    SendDevice(device *Device) error
    IsConnected() bool
}

type AWSIoTConnector struct {
    endpoint    string
    clientID    string
    certPath    string
    keyPath     string
    client      mqtt.Client
}

type GoogleCloudConnector struct {
    projectID   string
    topicID     string
    client      *pubsub.Client
}
```

**Operation**:
- Disabled by default in configuration
- When enabled, connect on startup
- Send profiles every 5 minutes (configurable)
- Retry failed transmissions with exponential backoff
- Continue local operations if cloud unavailable
- Log all cloud errors without blocking

**Configuration**:
```json
{
  "cloud": {
    "enabled": false,
    "provider": "aws",
    "aws": {
      "endpoint": "xxxxx.iot.us-east-1.amazonaws.com",
      "client_id": "heimdal-sensor-01",
      "cert_path": "/etc/heimdal/certs/device.crt",
      "key_path": "/etc/heimdal/certs/device.key"
    },
    "gcp": {
      "project_id": "heimdal-project",
      "topic_id": "sensor-data"
    }
  }
}
```

## Data Models

### Device Model
```go
type Device struct {
    MAC         string    `json:"mac" badger:"key"`
    IP          string    `json:"ip"`
    Name        string    `json:"name"`
    Vendor      string    `json:"vendor"`
    FirstSeen   time.Time `json:"first_seen"`
    LastSeen    time.Time `json:"last_seen"`
    IsActive    bool      `json:"is_active"`
}
```

### Behavioral Profile Model
```go
type BehavioralProfile struct {
    MAC             string                 `json:"mac" badger:"key"`
    Destinations    map[string]*DestInfo   `json:"destinations"`
    Ports           map[uint16]int         `json:"ports"`
    Protocols       map[string]int         `json:"protocols"`
    TotalPackets    int64                  `json:"total_packets"`
    TotalBytes      int64                  `json:"total_bytes"`
    FirstSeen       time.Time              `json:"first_seen"`
    LastSeen        time.Time              `json:"last_seen"`
    HourlyActivity  [24]int                `json:"hourly_activity"`
}

type DestInfo struct {
    IP          string    `json:"ip"`
    Count       int64     `json:"count"`
    LastSeen    time.Time `json:"last_seen"`
}
```

### Configuration Model
```go
type Config struct {
    Database    DatabaseConfig    `json:"database"`
    Network     NetworkConfig     `json:"network"`
    Discovery   DiscoveryConfig   `json:"discovery"`
    Interceptor InterceptorConfig `json:"interceptor"`
    Profiler    ProfilerConfig    `json:"profiler"`
    API         APIConfig         `json:"api"`
    Cloud       CloudConfig       `json:"cloud"`
    Logging     LoggingConfig     `json:"logging"`
}

type DatabaseConfig struct {
    Path            string `json:"path"`
    GCInterval      int    `json:"gc_interval_minutes"`
}

type NetworkConfig struct {
    Interface       string `json:"interface"`
    AutoDetect      bool   `json:"auto_detect"`
}

type DiscoveryConfig struct {
    ARPScanInterval     int  `json:"arp_scan_interval_seconds"`
    MDNSEnabled         bool `json:"mdns_enabled"`
    InactiveTimeout     int  `json:"inactive_timeout_minutes"`
}

type InterceptorConfig struct {
    Enabled             bool     `json:"enabled"`
    SpoofInterval       int      `json:"spoof_interval_seconds"`
    TargetMACs          []string `json:"target_macs"`
}

type ProfilerConfig struct {
    PersistInterval     int `json:"persist_interval_seconds"`
    MaxDestinations     int `json:"max_destinations"`
}

type APIConfig struct {
    Port                int    `json:"port"`
    Host                string `json:"host"`
    RateLimitPerMinute  int    `json:"rate_limit_per_minute"`
}

type CloudConfig struct {
    Enabled     bool              `json:"enabled"`
    Provider    string            `json:"provider"`
    AWS         AWSConfig         `json:"aws"`
    GCP         GCPConfig         `json:"gcp"`
}

type LoggingConfig struct {
    Level       string `json:"level"`
    File        string `json:"file"`
}
```

### Default Configuration
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

## Error Handling

### Component-Level Error Handling

Each component implements error handling following these principles:

1. **Retry with Backoff**: Transient errors retry with exponential backoff (max 3 attempts)
2. **Graceful Degradation**: Component failures don't crash the application
3. **Error Propagation**: Critical errors propagate to orchestrator for logging
4. **Recovery**: Components automatically restart after failure (max 5 restarts/hour)

### Specific Error Scenarios

**Network Auto-Config Failure**:
- Retry every 5 seconds indefinitely
- Block application startup until network detected
- Log each retry attempt

**Device Discovery Failure**:
- Log error and continue with other discovery methods
- If ARP fails, rely on mDNS
- If both fail, retry after 60 seconds

**Traffic Interceptor Failure**:
- Verify IP forwarding is enabled
- Check for required capabilities (CAP_NET_RAW, CAP_NET_ADMIN)
- Restore ARP tables before restart
- Exponential backoff: 1s, 2s, 4s, 8s, 16s (max)
- Alert via API if spoofing fails repeatedly

**Packet Analyzer Failure**:
- Reopen pcap handle
- Verify interface is up
- Check for permission issues
- Continue profiling with existing data if capture fails

**Database Failure**:
- Switch to in-memory buffer (max 1000 records)
- Retry database connection every 30 seconds
- Flush buffer when database available
- Log warning if buffer reaches capacity

**Web API Failure**:
- Log error but don't crash application
- Retry binding to port (may be in use)
- Provide health endpoint even if dashboard fails

**Cloud Connector Failure**:
- Log error and continue local operations
- Retry connection every 5 minutes
- Queue failed transmissions (max 100)
- Drop oldest queued data if queue full

### Logging Strategy

Use structured logging with levels:
- **DEBUG**: Detailed packet information, component state changes
- **INFO**: Startup, shutdown, device discovery, profile updates
- **WARN**: Retry attempts, degraded functionality, buffer usage
- **ERROR**: Component failures, database errors, critical issues

Log to both file (`/var/log/heimdal/heimdal.log`) and stdout for systemd journal.

## Testing Strategy

### Unit Tests

Test each component in isolation:
- **Network Auto-Config**: Mock network interfaces and routing tables
- **Device Discovery**: Mock ARP/mDNS responses
- **Traffic Interceptor**: Mock packet handles, verify ARP packet construction
- **Packet Analyzer**: Test packet parsing with sample pcap files
- **Behavioral Profiler**: Test aggregation logic with synthetic packet data
- **Database**: Test CRUD operations, error handling, buffering
- **Web API**: Test endpoints with mock database

### Integration Tests

Test component interactions:
- Device discovery → Database persistence
- Packet analyzer → Profiler → Database
- Database → Web API responses
- Orchestrator shutdown sequence

### System Tests

Test on actual Raspberry Pi:
- Full deployment via Ansible
- Network detection on real network
- ARP spoofing with test devices
- Resource usage monitoring (RAM, CPU)
- Long-running stability test (24+ hours)

### Performance Tests

- Packet processing throughput (target: 10,000 pps)
- Database write performance (target: 1,000 writes/sec)
- API response time (target: < 500ms)
- Memory usage under load (target: < 200MB)
- CPU usage under load (target: < 25%)

## Deployment Architecture

### Directory Structure

```
/opt/heimdal/
├── bin/
│   └── heimdal              # Statically compiled binary
├── web/
│   └── dashboard/
│       ├── index.html
│       ├── app.js
│       └── styles.css

/etc/heimdal/
├── config.json              # Main configuration
└── certs/                   # Cloud connector certificates (optional)
    ├── device.crt
    └── device.key

/var/lib/heimdal/
└── db/                      # BadgerDB data directory

/var/log/heimdal/
└── heimdal.log              # Application logs
```

### Systemd Service

**File**: `/etc/systemd/system/heimdal.service`

```ini
[Unit]
Description=Heimdal Network Security Sensor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=heimdal
Group=heimdal
ExecStart=/opt/heimdal/bin/heimdal --config /etc/heimdal/config.json
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/heimdal /var/log/heimdal

# Capabilities
AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN
CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
```

### Ansible Deployment

**Project Structure**:
```
ansible/
├── inventory.ini
├── playbook.yml
├── group_vars/
│   └── all.yml
├── roles/
│   └── heimdal_sensor/
│       ├── tasks/
│       │   └── main.yml
│       ├── templates/
│       │   ├── config.json.j2
│       │   └── heimdal.service.j2
│       ├── files/
│       │   └── heimdal              # Pre-compiled binary
│       └── handlers/
│           └── main.yml
```

**Inventory** (`inventory.ini`):
```ini
[heimdal_sensors]
heimdal-sensor-01 ansible_host=10.100.102.131 ansible_user=cortexa ansible_password=Password1

[heimdal_sensors:vars]
ansible_python_interpreter=/usr/bin/python3
```

**Playbook** (`playbook.yml`):
```yaml
---
- name: Deploy Heimdal Sensor
  hosts: heimdal_sensors
  become: yes
  roles:
    - heimdal_sensor
```

**Role Tasks** (`roles/heimdal_sensor/tasks/main.yml`):
```yaml
---
- name: Install system dependencies
  apt:
    name:
      - libpcap0.8
      - ca-certificates
    state: present
    update_cache: yes

- name: Enable IP forwarding
  ansible.posix.sysctl:
    name: net.ipv4.ip_forward
    value: '1'
    state: present
    reload: yes

- name: Create heimdal user
  user:
    name: heimdal
    system: yes
    shell: /usr/sbin/nologin
    create_home: no

- name: Create directory structure
  file:
    path: "{{ item }}"
    state: directory
    owner: heimdal
    group: heimdal
    mode: '0755'
  loop:
    - /opt/heimdal/bin
    - /opt/heimdal/web/dashboard
    - /etc/heimdal
    - /var/lib/heimdal
    - /var/log/heimdal

- name: Copy heimdal binary
  copy:
    src: heimdal
    dest: /opt/heimdal/bin/heimdal
    owner: heimdal
    group: heimdal
    mode: '0755'
  notify: restart heimdal

- name: Set capabilities on binary
  capabilities:
    path: /opt/heimdal/bin/heimdal
    capability: cap_net_raw,cap_net_admin=eip
    state: present

- name: Deploy configuration file
  template:
    src: config.json.j2
    dest: /etc/heimdal/config.json
    owner: heimdal
    group: heimdal
    mode: '0644'
  notify: restart heimdal

- name: Deploy systemd service
  template:
    src: heimdal.service.j2
    dest: /etc/systemd/system/heimdal.service
    owner: root
    group: root
    mode: '0644'
  notify:
    - reload systemd
    - restart heimdal

- name: Enable and start heimdal service
  systemd:
    name: heimdal
    enabled: yes
    state: started
```

**Handlers** (`roles/heimdal_sensor/handlers/main.yml`):
```yaml
---
- name: reload systemd
  systemd:
    daemon_reload: yes

- name: restart heimdal
  systemd:
    name: heimdal
    state: restarted
```

## Build Process

### Cross-Compilation for Raspberry Pi

**Build Script** (`build.sh`):
```bash
#!/bin/bash
set -e

# Configuration
GOOS=linux
GOARCH=arm64
OUTPUT=ansible/roles/heimdal_sensor/files/heimdal

# Build
echo "Building Heimdal for Raspberry Pi (ARM64)..."
CGO_ENABLED=1 \
CC=aarch64-linux-gnu-gcc \
GOOS=$GOOS \
GOARCH=$GOARCH \
go build -a \
  -ldflags="-s -w -extldflags '-static'" \
  -tags netgo \
  -o $OUTPUT \
  ./cmd/heimdal

echo "Build complete: $OUTPUT"
ls -lh $OUTPUT
```

**Dependencies**:
- Go 1.21+
- Cross-compiler: `aarch64-linux-gnu-gcc`
- Static linking for libpcap

**Build Commands**:
```bash
# Install cross-compiler (Ubuntu/Debian)
sudo apt-get install gcc-aarch64-linux-gnu

# Build
./build.sh

# Verify binary
file ansible/roles/heimdal_sensor/files/heimdal
```

## Go Module Dependencies

**go.mod**:
```go
module github.com/mosiko1234/heimdal/sensor

go 1.21

require (
    github.com/dgraph-io/badger/v4 v4.2.0
    github.com/google/gopacket v1.1.19
    github.com/gorilla/mux v1.8.1
    github.com/hashicorp/mdns v1.0.5
    github.com/aws/aws-sdk-go-v2 v1.24.0
    cloud.google.com/go/pubsub v1.33.0
    golang.org/x/time v0.5.0
)
```

**Key Libraries**:
- **BadgerDB**: Embedded database with high write performance
- **gopacket**: Packet capture and manipulation
- **gorilla/mux**: HTTP router for API
- **hashicorp/mdns**: mDNS/DNS-SD discovery
- **aws-sdk-go-v2**: AWS IoT connectivity
- **cloud.google.com/go/pubsub**: Google Cloud Pub/Sub
- **golang.org/x/time**: Rate limiting

## Security Considerations

### Privilege Management

- Run as non-root user (`heimdal`)
- Use Linux capabilities instead of root:
  - `CAP_NET_RAW`: Packet capture
  - `CAP_NET_ADMIN`: ARP manipulation
- Systemd security hardening:
  - `NoNewPrivileges=true`
  - `ProtectSystem=strict`
  - `ProtectHome=true`

### Network Security

- ARP spoofing is inherently invasive:
  - Only enable on trusted networks
  - Provide configuration to limit target devices
  - Restore ARP tables on shutdown
- Web API:
  - No authentication (local network trust model)
  - Rate limiting to prevent abuse
  - Input validation on all endpoints

### Data Privacy

- All data stored locally by default
- Cloud transmission requires explicit configuration
- No telemetry or external communication without consent
- Device names may contain PII (handle appropriately)

## Performance Optimization

### Memory Management

- Use sync.Pool for packet buffers
- Limit in-memory profile storage (max 1000 devices)
- Implement LRU cache for frequently accessed data
- Run garbage collection during low-traffic periods

### CPU Optimization

- Use buffered channels to reduce goroutine blocking
- Batch database writes (every 60 seconds)
- Implement packet sampling during high traffic
- Use efficient data structures (maps for O(1) lookups)

### Disk I/O

- BadgerDB's LSM-tree minimizes write amplification
- Batch writes to reduce fsync calls
- Use SSD if available (Raspberry Pi 4/5 support USB 3.0)
- Implement log rotation to prevent disk fill

## Monitoring and Observability

### Health Checks

- `/api/v1/health` endpoint returns:
  - Component status (running/stopped/error)
  - Database connectivity
  - Last successful operations
  - Resource usage (memory, CPU)

### Metrics

Expose via API:
- Total devices discovered
- Active devices
- Packets processed per second
- Database size
- Uptime
- Error counts per component

### Logging

Structured logs with fields:
- Timestamp
- Level (DEBUG/INFO/WARN/ERROR)
- Component name
- Message
- Context (device MAC, error details)

## Future Enhancements

### Phase 2 Features

1. **Anomaly Detection**:
   - Machine learning for behavioral analysis
   - Alert on unusual communication patterns
   - Baseline establishment period

2. **Advanced Dashboard**:
   - Real-time WebSocket updates
   - Interactive network graph visualization
   - Historical trend analysis

3. **Multi-Sensor Support**:
   - Sensor-to-sensor communication
   - Distributed monitoring
   - Centralized management

4. **Enhanced Cloud Integration**:
   - Full Asgard platform integration
   - Encrypted data transmission
   - Remote configuration management

5. **Mobile App**:
   - iOS/Android companion app
   - Push notifications for alerts
   - Remote monitoring

## Conclusion

This design provides a comprehensive architecture for Heimdal 2.0, a high-performance Go-based network security sensor. The system prioritizes zero-touch provisioning, resource efficiency, and reliability while maintaining extensibility for future cloud integration. The concurrent goroutine-based architecture ensures efficient operation on Raspberry Pi hardware, and the Ansible-based deployment enables consistent, repeatable provisioning.
