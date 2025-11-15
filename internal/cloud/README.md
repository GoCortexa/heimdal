# Cloud Connector Package

This package provides cloud connectivity for the Heimdal sensor, enabling transmission of device and behavioral profile data to cloud platforms.

## Architecture

The cloud connector system consists of:

1. **CloudConnector Interface** (`connector.go`) - Defines the contract for cloud platform implementations
2. **BaseConnector** (`connector.go`) - Provides common functionality including:
   - Connection state management
   - Transmission queue (max 100 items)
   - Retry logic with exponential backoff
   - Periodic transmission loop
3. **AWS IoT Connector** (`aws/iot_connector.go`) - Stub implementation for AWS IoT Core
4. **Google Cloud Connector** (`gcp/pubsub_connector.go`) - Stub implementation for Google Cloud Pub/Sub
5. **Orchestrator** (`orchestrator.go`) - Manages connector lifecycle and data transmission

## Usage

### Configuration

Cloud connectivity is configured in `/etc/heimdal/config.json`:

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

### Initialization

```go
import (
    "github.com/mosiko1234/heimdal/sensor/internal/cloud"
    "github.com/mosiko1234/heimdal/sensor/internal/cloud/aws"
    "github.com/mosiko1234/heimdal/sensor/internal/cloud/gcp"
    "github.com/mosiko1234/heimdal/sensor/internal/config"
    "github.com/mosiko1234/heimdal/sensor/internal/database"
)

// Load configuration
cfg, err := config.LoadConfig("/etc/heimdal/config.json")
if err != nil {
    log.Fatal(err)
}

// Initialize database
db, err := database.NewDatabaseManager(cfg.Database.Path)
if err != nil {
    log.Fatal(err)
}

// Create orchestrator
orchestrator, err := cloud.NewOrchestrator(&cfg.Cloud, db)
if err != nil {
    log.Fatal(err)
}

// Create connector based on provider
var connector cloud.CloudConnector
if cfg.Cloud.Enabled {
    switch cfg.Cloud.Provider {
    case "aws":
        connector, err = aws.NewAWSIoTConnector(&cfg.Cloud.AWS, db)
    case "gcp":
        connector, err = gcp.NewGoogleCloudConnector(&cfg.Cloud.GCP, db)
    default:
        log.Fatalf("Unsupported provider: %s", cfg.Cloud.Provider)
    }
    
    if err != nil {
        log.Fatal(err)
    }
    
    orchestrator.SetConnector(connector)
}

// Start orchestrator
if err := orchestrator.Start(); err != nil {
    log.Printf("Warning: Failed to start cloud connector: %v", err)
    // Local operations continue even if cloud fails
}

// ... application runs ...

// Graceful shutdown
orchestrator.Stop()
```

## Features

### Automatic Retry with Exponential Backoff

Failed transmissions are automatically retried with exponential backoff:
- Initial retry: 1 second delay
- Second retry: 2 seconds delay
- Third retry: 4 seconds delay
- Maximum retries: 3 attempts

### Transmission Queue

- Maximum queue size: 100 items
- When full, oldest items are dropped
- Queue persists during temporary disconnections
- Items include both profiles and devices

### Periodic Transmission

- Default interval: 5 minutes
- Transmits all profiles and devices from database
- Continues local operations if cloud unavailable

### Graceful Degradation

- Cloud failures don't affect local operations
- Connection attempts continue in background
- Automatic reconnection on network recovery

## Stub Implementations

The current AWS and GCP connectors are **stub implementations** that:
- Log connection attempts and data transmissions
- Simulate successful operations
- Provide detailed comments showing full implementation approach
- Allow testing of orchestration logic without cloud dependencies

### Converting Stubs to Full Implementation

#### AWS IoT Core

To implement full AWS IoT connectivity:

1. Add MQTT client dependency:
   ```bash
   go get github.com/eclipse/paho.mqtt.golang
   ```

2. Uncomment and implement the MQTT client code in `aws/iot_connector.go`
3. Load TLS certificates for authentication
4. Configure MQTT topics for device and profile data
5. Implement QoS 1 for reliable delivery

#### Google Cloud Pub/Sub

To implement full Google Cloud connectivity:

1. Add Pub/Sub client dependency:
   ```bash
   go get cloud.google.com/go/pubsub
   ```

2. Uncomment and implement the Pub/Sub client code in `gcp/pubsub_connector.go`
3. Configure service account authentication
4. Create or reference Pub/Sub topics
5. Configure batching and compression settings

## Requirements Satisfied

This implementation satisfies the following requirements:

- **8.1**: CloudConnector interface for transmitting data to cloud platforms
- **8.2**: Stub implementations for AWS IoT Core and Google Cloud IoT Core
- **8.3**: Transmission of behavioral profiles when cloud connectivity is enabled
- **8.4**: Cloud connectivity disabled by default
- **8.5**: Local operations continue when cloud transmission fails

## Testing

To test the cloud connector:

```go
// Create test profile
profile := &database.BehavioralProfile{
    MAC: "aa:bb:cc:dd:ee:ff",
    Destinations: make(map[string]*database.DestInfo),
    Ports: make(map[uint16]int),
    Protocols: make(map[string]int),
    FirstSeen: time.Now(),
    LastSeen: time.Now(),
}

// Send via connector
if err := connector.SendProfile(profile); err != nil {
    log.Printf("Failed to send profile: %v", err)
}

// Check connection status
if connector.IsConnected() {
    log.Println("Connected to cloud")
}
```

## Future Enhancements

- Add compression for large payloads
- Implement delta updates (only changed data)
- Add message signing for integrity verification
- Support for bidirectional communication (commands from cloud)
- Metrics and monitoring integration
- Support for additional cloud providers (Azure, etc.)
