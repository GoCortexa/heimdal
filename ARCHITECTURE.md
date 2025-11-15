# Heimdal Architecture

This document provides a detailed explanation of the Heimdal sensor architecture, component interactions, and design decisions.

## Design Principles

1. **Zero-Touch Operation**: Automatic network detection and service startup with no manual configuration
2. **Concurrency-First**: All major components run as independent goroutines communicating via channels
3. **Resilience**: Automatic recovery from component failures without full application restart
4. **Resource Efficiency**: Optimized for Raspberry Pi constraints (< 200MB RAM, < 25% CPU)
5. **Single Binary**: Statically compiled with no external runtime dependencies
6. **Local-First**: Full functionality without cloud connectivity; cloud integration is optional

## High-Level Architecture

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

## Component Communication

Components communicate via typed Go channels, enabling loose coupling and concurrent operation:

- **deviceChan**: Discovered devices flow from Discovery → Interceptor
- **packetChan**: Analyzed packet metadata flows from Analyzer → Profiler
- **shutdownChan**: Orchestrator signals graceful shutdown to all components

This channel-based architecture allows components to operate independently while maintaining data flow integrity.

## Component Details

### 1. Main Orchestrator

**Location**: `internal/orchestrator/orchestrator.go`, `cmd/heimdal/main.go`

The orchestrator is the central coordinator responsible for:
- Loading configuration from `/etc/heimdal/config.json`
- Initializing all components in the correct dependency order
- Starting each component as an independent goroutine
- Monitoring component health and restarting failed components
- Coordinating graceful shutdown on SIGTERM/SIGINT

**Startup Sequence**:
1. Load and validate configuration
2. Initialize DatabaseManager
3. Initialize Network Auto-Config (blocks until network detected)
4. Start Device Discovery
5. Start Traffic Interceptor
6. Start Packet Analyzer
7. Start Behavioral Profiler
8. Start Web API
9. Start Cloud Connector (if enabled)

**Component Interface**:
```go
type Component interface {
    Start(ctx context.Context) error
    Stop() error
    Name() string
}
```

All components implement this interface, allowing the orchestrator to manage them uniformly.

### 2. Network Auto-Config Component

**Location**: `internal/netconfig/autoconfig.go`

Automatically detects the local network configuration at startup:
- Identifies primary network interface (eth0, wlan0, etc.)
- Determines gateway IP by parsing `/proc/net/route`
- Calculates subnet mask and CIDR notation
- Provides thread-safe read access to network configuration

This component blocks application startup until a valid network is detected, retrying every 5 seconds. This ensures all downstream components have valid network information before starting.

### 3. Device Discovery Component

**Location**: `internal/discovery/scanner.go`, `internal/discovery/arp.go`, `internal/discovery/mdns.go`

Continuously scans the local network to discover connected devices using two methods:

**ARP Scanning** (`arp.go`):
- Sends ARP requests to all IPs in the subnet CIDR range
- Parses ARP responses to extract IP and MAC addresses
- Runs every 60 seconds (configurable)
- Uses `gopacket` library for packet crafting and parsing

**mDNS Discovery** (`mdns.go`):
- Passive listener for mDNS service announcements
- Active mDNS queries every 5 minutes
- Extracts device names from mDNS responses
- Uses `hashicorp/mdns` library

**Device Lifecycle**:
- Tracks `LastSeen` timestamp for each device
- Marks devices inactive if not seen for 5 minutes
- Updates database immediately on discovery/update
- Sends discovered devices to `deviceChan` for interception

### 4. Traffic Interceptor Component

**Location**: `internal/interceptor/arp_spoofer.go`

Performs ARP spoofing to intercept network traffic:

**Operation**:
1. Listens on `deviceChan` for newly discovered devices
2. Adds devices to internal targets map
3. Sends spoofed ARP replies every 2 seconds to each target
4. Spoofs both directions: target → gateway and gateway → target
5. Removes inactive devices from spoofing list

**ARP Spoofing Mechanism**:
- Crafts ARP reply packets claiming the sensor's MAC is the gateway
- Sends to both target device and actual gateway
- Causes traffic to flow through the sensor
- Sensor forwards packets (requires IP forwarding enabled)

**Safety and Recovery**:
- Verifies IP forwarding is enabled before starting
- Implements health checks to verify packet forwarding
- Automatic restart on failure with exponential backoff
- Graceful cleanup: restores original ARP tables on shutdown

**Security Note**: ARP spoofing is inherently invasive and should only be used on networks you own or have explicit permission to monitor.

### 5. Packet Analyzer Component

**Location**: `internal/analyzer/sniffer.go`

Captures and analyzes network packets:

**Packet Capture**:
- Opens network interface in promiscuous mode using `pcap`
- Applies BPF filter: `not broadcast and not multicast`
- Captures all packets flowing through the interface

**Packet Processing**:
- Parses Ethernet layer for source MAC address
- Parses IP layer for destination IP address
- Parses TCP/UDP layer for destination port and protocol
- Creates `PacketInfo` struct with extracted metadata

**Rate Limiting**:
- Implements rate limiter (10,000 packets/second)
- Uses buffered channel (size 1000) to prevent blocking
- Non-blocking send: drops packets if channel is full
- Prevents goroutine blocking during high traffic periods

**Optimization**:
- Zero-copy packet processing where possible
- Minimal memory allocation per packet
- Efficient filtering to reduce processing load

### 6. Behavioral Profiler Component

**Location**: `internal/profiler/profiler.go`

Aggregates packet metadata into behavioral profiles:

**Profile Structure**:
- **Destinations**: Map of destination IPs with packet counts
- **Ports**: Frequency distribution of destination ports
- **Protocols**: Count of TCP, UDP, ICMP, etc.
- **Volume**: Total packets and bytes
- **Timing**: Hourly activity pattern (24-hour array)

**Aggregation Logic**:
1. Receive `PacketInfo` from `packetChan`
2. Look up or create profile for source MAC
3. Update destination IP counter
4. Update port frequency
5. Update protocol counter
6. Increment total packets and bytes
7. Update hourly activity based on timestamp

**Persistence**:
- Maintains profiles in memory for fast updates
- Persists all profiles to database every 60 seconds
- Uses batch operations for efficient database writes
- Loads existing profiles from database on startup

**Memory Management**:
- Limits maximum destinations per profile (configurable)
- Prunes least-recently-seen destinations when limit reached
- Efficient map-based storage for O(1) lookups

### 7. Database Component

**Location**: `internal/database/badger_db.go`

Provides persistent storage using BadgerDB:

**Why BadgerDB?**
- Pure Go, embedded database (no external dependencies)
- LSM-tree based for high write throughput
- Excellent concurrency support
- No C dependencies (important for cross-compilation)
- Optimized for SSD storage

**Data Model**:
```
device:<MAC_ADDRESS>   → JSON-serialized Device struct
profile:<MAC_ADDRESS>  → JSON-serialized BehavioralProfile struct
```

**Operations**:
- CRUD operations for devices and profiles
- Batch operations for bulk writes
- In-memory buffering when database unavailable
- Automatic buffer flush when database recovers

**Error Handling**:
- Retry failed writes up to 3 times with exponential backoff
- Fall back to memory buffer (max 1000 records) if database unavailable
- Graceful handling of corrupted data (skip and log)
- Periodic garbage collection (every 5 minutes)

### 8. Web API and Dashboard Component

**Location**: `internal/api/server.go`, `web/dashboard/`

Provides REST API and web dashboard:

**API Endpoints**:
- `GET /api/v1/devices` - List all discovered devices
- `GET /api/v1/devices/:mac` - Get device details
- `GET /api/v1/profiles/:mac` - Get behavioral profile
- `GET /api/v1/stats` - System statistics
- `GET /api/v1/health` - Health check
- `GET /` - Dashboard HTML

**Dashboard Features**:
- Device list table with MAC, IP, Name, Status
- Click device to view detailed behavioral profile
- Visual representation of top destinations
- Port usage chart
- Activity timeline (24-hour)
- Auto-refresh every 10 seconds

**Security**:
- Rate limiting: 100 requests/minute per IP
- Input validation on all endpoints
- CORS enabled for local network access
- No authentication (local network trust model)

### 9. Cloud Connector Component

**Location**: `internal/cloud/connector.go`, `internal/cloud/orchestrator.go`

Optional cloud integration for future Asgard platform:

**Interface Design**:
```go
type CloudConnector interface {
    Connect() error
    Disconnect() error
    SendProfile(profile *BehavioralProfile) error
    SendDevice(device *Device) error
    IsConnected() bool
}
```

**Implementations**:
- **AWS IoT**: Stub implementation using MQTT (`internal/cloud/aws/`)
- **Google Cloud**: Stub implementation using Pub/Sub (`internal/cloud/gcp/`)

**Operation**:
- Disabled by default in configuration
- When enabled, connects on startup
- Transmits profiles every 5 minutes (configurable)
- Retry failed transmissions with exponential backoff
- Continues local operations if cloud unavailable

**Design Note**: These are stub implementations showing the integration pattern. Full implementation requires valid cloud credentials and configuration.

## Data Flow

### Device Discovery Flow

```
Network
  ↓ (ARP/mDNS)
Device Discovery
  ↓ (deviceChan)
Database ← Device Record
  ↓
Traffic Interceptor (adds to spoof targets)
```

### Packet Analysis Flow

```
Network Interface
  ↓ (raw packets)
Packet Analyzer
  ↓ (packetChan - PacketInfo)
Behavioral Profiler
  ↓ (aggregation)
In-Memory Profiles
  ↓ (every 60s)
Database ← Profile Records
  ↓
Web API (on request)
  ↓
Dashboard / API Client
```

### Shutdown Flow

```
OS Signal (SIGTERM/SIGINT)
  ↓
Orchestrator receives signal
  ↓
Broadcasts to shutdownChan
  ↓
Components stop in reverse order:
  1. Cloud Connector
  2. Web API
  3. Behavioral Profiler (flush profiles)
  4. Packet Analyzer
  5. Traffic Interceptor (restore ARP tables)
  6. Device Discovery
  7. Database (close cleanly)
```

## Concurrency Model

### Goroutine Architecture

Each major component runs in its own goroutine:
- **Main goroutine**: Orchestrator and signal handling
- **Network Auto-Config**: Blocking detection loop
- **Device Discovery**: ARP scanner + mDNS listener (2 goroutines)
- **Traffic Interceptor**: Spoofing loop + device listener
- **Packet Analyzer**: Packet capture loop
- **Behavioral Profiler**: Packet processor + persistence ticker
- **Web API**: HTTP server (goroutine per request)
- **Cloud Connector**: Transmission loop

### Synchronization

- **Channels**: Primary communication mechanism (typed, buffered)
- **Mutexes**: Protect shared state within components (RWMutex for read-heavy)
- **WaitGroups**: Coordinate graceful shutdown
- **Context**: Propagate cancellation signals

### Channel Buffering Strategy

- **deviceChan**: Buffered (size 100) - discovery is bursty
- **packetChan**: Buffered (size 1000) - high throughput
- **shutdownChan**: Unbuffered - immediate signal propagation

## Error Handling Strategy

### Component-Level Recovery

Each component implements error handling following these principles:

1. **Retry with Backoff**: Transient errors retry with exponential backoff (max 3 attempts)
2. **Graceful Degradation**: Component failures don't crash the application
3. **Error Propagation**: Critical errors propagate to orchestrator for logging
4. **Automatic Recovery**: Components restart after failure (max 5 restarts/hour)

### Specific Error Scenarios

**Network Auto-Config Failure**:
- Retry every 5 seconds indefinitely
- Block application startup until network detected
- Log each retry attempt

**Device Discovery Failure**:
- Log error and continue with other discovery methods
- If ARP fails, rely on mDNS (and vice versa)
- Retry after configured interval

**Traffic Interceptor Failure**:
- Verify IP forwarding and capabilities
- Restore ARP tables before restart
- Exponential backoff: 1s, 2s, 4s, 8s, 16s (max)

**Packet Analyzer Failure**:
- Reopen pcap handle
- Verify interface is up
- Continue profiling with existing data

**Database Failure**:
- Switch to in-memory buffer (max 1000 records)
- Retry connection every 30 seconds
- Flush buffer when database available

**Web API Failure**:
- Log error but don't crash application
- Retry binding to port
- Provide health endpoint even if dashboard fails

**Cloud Connector Failure**:
- Log error and continue local operations
- Retry connection every 5 minutes
- Queue failed transmissions (max 100)

For detailed error handling patterns, see [LOGGING_AND_ERROR_HANDLING.md](LOGGING_AND_ERROR_HANDLING.md).

## Performance Characteristics

### Resource Usage

**Target Metrics** (Raspberry Pi 4):
- Memory: < 200MB RAM during normal operation
- CPU: < 25% average utilization
- Disk I/O: Minimal (batch writes every 60s)
- Network: Minimal overhead (ARP every 60s, spoofing every 2s)

**Optimization Techniques**:
- Buffered channels prevent goroutine blocking
- Rate limiting prevents resource exhaustion
- Batch database operations reduce I/O
- Zero-copy packet processing where possible
- Efficient data structures (maps for O(1) lookups)

### Scalability Limits

**Network Size**:
- Tested with up to 254 devices (/24 subnet)
- ARP scan time scales linearly with subnet size
- Memory usage: ~1KB per device, ~10KB per profile

**Packet Throughput**:
- Rate limited to 10,000 packets/second
- Drops packets if rate exceeded (non-blocking)
- Typical home network: 100-1000 packets/second

**Database Performance**:
- BadgerDB handles 1,000+ writes/second
- Batch operations improve throughput
- LSM-tree design optimized for write-heavy workload

## Security Architecture

### Privilege Management

- Runs as non-root user (`heimdal`)
- Uses Linux capabilities instead of root:
  - `CAP_NET_RAW`: Packet capture
  - `CAP_NET_ADMIN`: ARP manipulation
- Systemd security hardening:
  - `NoNewPrivileges=true`
  - `ProtectSystem=strict`
  - `ProtectHome=true`

### Network Security

- ARP spoofing is inherently invasive - only use on trusted networks
- Web API has no authentication (local network trust model)
- Rate limiting prevents abuse
- Input validation on all endpoints

### Data Privacy

- All data stored locally by default
- Cloud transmission requires explicit configuration
- No telemetry or external communication without consent
- Device names may contain PII - handle appropriately

## Deployment Architecture

### Directory Structure

```
/opt/heimdal/
├── bin/heimdal              # Statically compiled binary
└── web/dashboard/           # Static web files

/etc/heimdal/
├── config.json              # Main configuration
└── certs/                   # Cloud connector certificates (optional)

/var/lib/heimdal/
└── db/                      # BadgerDB data directory

/var/log/heimdal/
└── heimdal.log              # Application logs
```

### Systemd Integration

The sensor runs as a systemd service with:
- Automatic restart on failure
- Proper capability management
- Security hardening
- Journal logging integration

See `ansible/roles/heimdal_sensor/templates/heimdal.service.j2` for the complete service definition.

## Future Enhancements

### Planned Features

1. **Anomaly Detection**: Machine learning-based detection of unusual behavior
2. **Full Cloud Integration**: Complete AWS IoT and Google Cloud implementations
3. **Multi-Sensor Coordination**: Distributed deployment with central management
4. **Advanced Profiling**: Application-layer protocol analysis
5. **Alerting**: Real-time notifications for security events

### Extensibility Points

- **Cloud Connector Interface**: Add new cloud providers
- **Discovery Methods**: Add new device discovery protocols
- **Analysis Plugins**: Extend packet analysis capabilities
- **Storage Backends**: Alternative database implementations

## Related Documentation

- [README.md](README.md) - Project overview and quick start
- [CONFIG.md](CONFIG.md) - Configuration reference
- [BUILD.md](BUILD.md) - Build process and cross-compilation
- [LOGGING_AND_ERROR_HANDLING.md](LOGGING_AND_ERROR_HANDLING.md) - Error handling patterns
