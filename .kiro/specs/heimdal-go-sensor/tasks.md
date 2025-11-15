# Implementation Plan

- [x] 1. Initialize Go project structure and dependencies
  - Create Go module with `go mod init github.com/mosiko1234/heimdal/sensor`
  - Add all required dependencies to go.mod (BadgerDB, gopacket, gorilla/mux, hashicorp/mdns)
  - Create directory structure: cmd/heimdal/, internal/{database,netconfig,discovery,interceptor,analyzer,profiler,api,cloud,orchestrator}/
  - Create web/dashboard/ directory for static files
  - _Requirements: 9.5_

- [x] 2. Implement configuration management
  - Create internal/config/config.go with all configuration structs (Config, DatabaseConfig, NetworkConfig, etc.)
  - Implement LoadConfig() function to read and parse /etc/heimdal/config.json
  - Implement default configuration values
  - Add configuration validation logic
  - _Requirements: 7.6, 9.1_

- [x] 3. Implement BadgerDB database layer
  - [x] 3.1 Create database manager core
    - Create internal/database/badger_db.go with DatabaseManager struct
    - Implement NewDatabaseManager(path string) to initialize BadgerDB
    - Implement Close() method for graceful shutdown
    - Add key prefix constants (device:, profile:)
    - _Requirements: 5.1, 5.2_

  - [x] 3.2 Implement device CRUD operations
    - Implement SaveDevice(device *Device) with JSON serialization
    - Implement GetDevice(mac string) with JSON deserialization
    - Implement GetAllDevices() to retrieve all devices
    - Implement DeleteDevice(mac string)
    - _Requirements: 5.1, 5.4_

  - [x] 3.3 Implement profile CRUD operations
    - Implement SaveProfile(profile *BehavioralProfile) with JSON serialization
    - Implement GetProfile(mac string) with JSON deserialization
    - Implement GetAllProfiles() to retrieve all profiles
    - _Requirements: 4.5, 5.1_

  - [x] 3.4 Implement batch operations and buffering
    - Implement SaveDeviceBatch() for bulk device writes
    - Implement SaveProfileBatch() for bulk profile writes
    - Create MemoryBuffer struct for database unavailability scenarios
    - Implement buffer flush logic when database becomes available
    - _Requirements: 5.3_

- [x] 4. Implement network auto-configuration component
  - Create internal/netconfig/autoconfig.go with NetworkConfig and AutoConfig structs
  - Implement DetectNetwork() to find primary interface (eth0, wlan0)
  - Implement getGateway() by parsing /proc/net/route
  - Implement getSubnet() using net.InterfaceAddrs()
  - Add retry logic (every 5 seconds) until network detected
  - Provide thread-safe read access to network configuration
  - _Requirements: 1.1, 1.2_

- [x] 5. Implement device discovery component
  - [x] 5.1 Create discovery scanner core
    - Create internal/discovery/scanner.go with Scanner struct and Device model
    - Implement NewScanner() constructor with dependencies (netConfig, db, deviceChan)
    - Implement Start() method to launch discovery goroutines
    - Implement Stop() method for graceful shutdown
    - _Requirements: 2.1, 2.3_

  - [x] 5.2 Implement ARP scanning
    - Create scanARP() function using gopacket to send ARP requests
    - Iterate through subnet CIDR range
    - Parse ARP responses to extract IP and MAC
    - Send discovered devices to deviceChan
    - Run scan every 60 seconds
    - _Requirements: 2.1, 2.2_

  - [x] 5.3 Implement mDNS discovery
    - Create scanMDNS() function using hashicorp/mdns library
    - Set up passive mDNS listener for service announcements
    - Perform active mDNS query every 5 minutes
    - Extract device names from mDNS responses
    - Merge mDNS data with ARP-discovered devices
    - _Requirements: 2.1, 2.3_

  - [x] 5.4 Implement device lifecycle management
    - Track LastSeen timestamp for each device
    - Mark devices inactive if not seen for 5 minutes
    - Update database immediately on device discovery/update
    - Handle device reconnection (update existing records)
    - _Requirements: 2.2, 2.4_

- [x] 6. Implement traffic interceptor (ARP spoofer)
  - [x] 6.1 Create ARP spoofer core
    - Create internal/interceptor/arp_spoofer.go with ARPSpoofer struct
    - Implement NewARPSpoofer() constructor
    - Create SpoofTarget struct to track spoofed devices
    - Implement Start() to begin spoofing operations
    - Implement Stop() to restore ARP tables
    - _Requirements: 3.1, 3.2_

  - [x] 6.2 Implement ARP packet crafting
    - Create buildARPReply() function using gopacket layers.ARP
    - Craft ARP reply packets with sensor's MAC as gateway
    - Implement sendARPPacket() to transmit via raw socket
    - _Requirements: 3.1_

  - [x] 6.3 Implement spoofing loop and target management
    - Listen on deviceChan for new devices to spoof
    - Add devices to targets map
    - Send ARP replies every 2 seconds to each target
    - Send spoofed packets to both target device and gateway
    - Remove inactive devices from spoofing list
    - _Requirements: 3.1, 3.2_

  - [x] 6.4 Implement failure recovery and health checks
    - Verify IP forwarding is enabled before starting
    - Implement automatic restart on failure with exponential backoff
    - Add health check to verify packets are being forwarded
    - Implement graceful cleanup to restore original ARP tables
    - _Requirements: 3.3, 3.5_

- [x] 7. Implement packet analyzer component
  - [x] 7.1 Create packet sniffer core
    - Create internal/analyzer/sniffer.go with Sniffer struct and PacketInfo model
    - Implement NewSniffer() constructor
    - Open network interface in promiscuous mode using pcap
    - Apply BPF filter: "not broadcast and not multicast"
    - _Requirements: 4.1_

  - [x] 7.2 Implement packet processing
    - Create processPacket() function to parse captured packets
    - Extract Ethernet layer for source MAC
    - Extract IP layer for destination IP
    - Extract TCP/UDP layer for destination port and protocol
    - Create PacketInfo struct with extracted metadata
    - _Requirements: 4.2_

  - [x] 7.3 Implement rate limiting and channel management
    - Create rate limiter (10,000 packets/second) using golang.org/x/time/rate
    - Use buffered packetChan (size 1000)
    - Implement non-blocking send to prevent goroutine blocking
    - Drop packets if rate limit exceeded or channel full
    - _Requirements: 10.3, 10.4_

- [x] 8. Implement behavioral profiler component
  - [x] 8.1 Create profiler core
    - Create internal/profiler/profiler.go with Profiler struct
    - Create BehavioralProfile and DestInfo models
    - Implement NewProfiler() constructor with dependencies
    - Initialize profiles map for in-memory storage
    - Load existing profiles from database on startup
    - _Requirements: 4.3, 4.4_

  - [x] 8.2 Implement profile aggregation logic
    - Create updateProfile() function to process PacketInfo
    - Update Destinations map with destination IP and count
    - Update Ports map with destination port frequency
    - Update Protocols map with protocol counts
    - Increment TotalPackets and TotalBytes counters
    - Update HourlyActivity array based on packet timestamp
    - _Requirements: 4.3, 4.4_

  - [x] 8.3 Implement profile persistence
    - Create persistence goroutine with 60-second ticker
    - Implement persistProfiles() to save all profiles to database
    - Use batch operations for efficient database writes
    - Handle database errors with retry logic
    - _Requirements: 4.5_

- [x] 9. Implement web API and dashboard
  - [x] 9.1 Create API server core
    - Create internal/api/server.go with APIServer struct
    - Implement NewAPIServer() constructor with database dependency
    - Set up gorilla/mux router
    - Configure CORS for local network access
    - Implement rate limiting (100 requests/minute per IP)
    - _Requirements: 6.1, 6.4_

  - [x] 9.2 Implement API endpoints
    - Implement GET /api/v1/devices handler to list all devices
    - Implement GET /api/v1/devices/:mac handler for device details
    - Implement GET /api/v1/profiles/:mac handler for behavioral profile
    - Implement GET /api/v1/stats handler for system statistics
    - Implement GET /api/v1/health handler for health check
    - Add JSON response helpers and error handling
    - _Requirements: 6.2, 6.3_

  - [x] 9.3 Create dashboard frontend
    - Create web/dashboard/index.html with device list table
    - Create web/dashboard/app.js with API fetch logic
    - Implement auto-refresh every 10 seconds
    - Add device detail view with behavioral profile visualization
    - Create web/dashboard/styles.css for basic styling
    - Implement static file serving in API server
    - _Requirements: 6.2_

- [x] 10. Implement cloud connector interface and stubs
  - [x] 10.1 Create cloud connector interface
    - Create internal/cloud/connector.go with CloudConnector interface
    - Define Connect(), Disconnect(), SendProfile(), SendDevice(), IsConnected() methods
    - Create base connector struct with common functionality
    - _Requirements: 8.1, 8.5_

  - [x] 10.2 Implement AWS IoT connector stub
    - Create internal/cloud/aws/iot_connector.go with AWSIoTConnector struct
    - Implement stub Connect() method with MQTT client initialization logic
    - Implement stub SendProfile() method showing JSON serialization
    - Implement stub SendDevice() method
    - Add configuration parsing for AWS endpoint, client ID, certificates
    - _Requirements: 8.2_

  - [x] 10.3 Implement Google Cloud connector stub
    - Create internal/cloud/gcp/pubsub_connector.go with GoogleCloudConnector struct
    - Implement stub Connect() method with Pub/Sub client initialization logic
    - Implement stub SendProfile() method showing message publishing
    - Implement stub SendDevice() method
    - Add configuration parsing for GCP project ID and topic ID
    - _Requirements: 8.2_

  - [x] 10.4 Implement cloud connector orchestration
    - Create factory function to instantiate correct connector based on config
    - Implement transmission goroutine with 5-minute interval
    - Add retry logic with exponential backoff for failed transmissions
    - Implement transmission queue (max 100 items)
    - Ensure local operations continue if cloud unavailable
    - _Requirements: 8.3, 8.4, 8.5_

- [x] 11. Implement main orchestrator
  - [x] 11.1 Create orchestrator core
    - Create internal/orchestrator/orchestrator.go with Orchestrator struct
    - Implement Component interface (Start, Stop, Name methods)
    - Create component registry to track all components
    - Implement NewOrchestrator() constructor
    - _Requirements: 9.1_

  - [x] 11.2 Implement component lifecycle management
    - Implement initializeComponents() to create all component instances
    - Implement startComponents() to launch each component as goroutine
    - Implement correct startup sequence (netconfig → discovery → interceptor → analyzer → profiler → api → cloud)
    - Add component health monitoring
    - Implement automatic component restart on failure (max 5 restarts/hour)
    - _Requirements: 9.1, 9.4_

  - [x] 11.3 Implement graceful shutdown
    - Set up signal handling for SIGTERM and SIGINT
    - Implement shutdown() method to stop all components in reverse order
    - Use sync.WaitGroup to wait for all goroutines
    - Implement 5-second shutdown timeout
    - Close database and restore ARP tables before exit
    - _Requirements: 9.3_

- [x] 12. Implement main entry point
  - Create cmd/heimdal/main.go
  - Parse command-line flags (--config path)
  - Load configuration file
  - Initialize logging to file and stdout
  - Create and start orchestrator
  - Block until shutdown signal received
  - Handle panic recovery and logging
  - _Requirements: 9.1, 9.5_

- [x] 13. Create Ansible deployment automation
  - [x] 13.1 Create Ansible project structure
    - Create ansible/inventory.ini with heimdal-sensor-01 host definition
    - Create ansible/playbook.yml as main playbook
    - Create ansible/group_vars/all.yml for shared variables
    - Create ansible/roles/heimdal_sensor/ directory structure
    - _Requirements: 7.1_

  - [x] 13.2 Implement Ansible role tasks
    - Create ansible/roles/heimdal_sensor/tasks/main.yml
    - Add task to install system dependencies (libpcap0.8, ca-certificates)
    - Add task to enable IP forwarding using ansible.posix.sysctl module
    - Add task to create heimdal system user
    - Add task to create directory structure (/opt/heimdal, /etc/heimdal, /var/lib/heimdal, /var/log/heimdal)
    - Add task to copy pre-compiled binary to /opt/heimdal/bin/heimdal
    - Add task to set capabilities (cap_net_raw, cap_net_admin) using capabilities module
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [x] 13.3 Create Ansible templates
    - Create ansible/roles/heimdal_sensor/templates/config.json.j2 with default configuration
    - Create ansible/roles/heimdal_sensor/templates/heimdal.service.j2 for systemd service
    - Add Jinja2 variables for customizable settings (database path, API port, etc.)
    - _Requirements: 7.5, 7.6_

  - [x] 13.4 Implement service deployment
    - Add task to deploy configuration file to /etc/heimdal/config.json
    - Add task to deploy systemd service file to /etc/systemd/system/heimdal.service
    - Create ansible/roles/heimdal_sensor/handlers/main.yml with reload systemd and restart heimdal handlers
    - Add task to enable and start heimdal service
    - _Requirements: 7.5_

- [x] 14. Create build and cross-compilation setup
  - Create build.sh script for cross-compiling to ARM64
  - Configure CGO_ENABLED=1 with aarch64-linux-gnu-gcc
  - Set GOOS=linux and GOARCH=arm64
  - Add static linking flags (-ldflags="-s -w -extldflags '-static'")
  - Output binary to ansible/roles/heimdal_sensor/files/heimdal
  - Add build verification step
  - _Requirements: 9.5_

- [x] 15. Create default configuration file
  - Create config/config.json with all default values
  - Set database path to /var/lib/heimdal/db
  - Configure discovery intervals (ARP: 60s, mDNS: enabled)
  - Configure interceptor (enabled, 2s spoof interval)
  - Configure profiler (60s persist interval)
  - Configure API (port 8080, rate limit 100/min)
  - Disable cloud connector by default
  - Set logging level to info
  - _Requirements: 7.6_

- [x] 16. Implement error handling and logging
  - Add structured logging throughout all components using standard log package or logrus
  - Implement error wrapping with context information
  - Add retry logic with exponential backoff for transient errors
  - Implement component-specific error recovery strategies
  - Add logging to file (/var/log/heimdal/heimdal.log) and stdout
  - _Requirements: 9.4_

- [x] 17. Create integration tests
  - Create test/integration/ directory
  - Write integration test for device discovery → database persistence flow
  - Write integration test for packet analyzer → profiler → database flow
  - Write integration test for database → API response flow
  - Write integration test for orchestrator shutdown sequence
  - _Requirements: 9.3, 9.4_

- [x] 18. Create documentation
  - Create README.md with project overview and quick start guide
  - Document build process and cross-compilation steps
  - Document Ansible deployment process
  - Create ARCHITECTURE.md explaining component interactions
  - Document configuration options in CONFIG.md
  - Add inline code documentation (godoc comments)
  - _Requirements: 9.5_
