# Heimdal Logging System

## Overview

The Heimdal sensor uses a structured logging system that provides consistent, contextual logging across all components. The logging system supports multiple log levels, component-specific loggers, and outputs to both file and stdout.

## Log Levels

The system supports four log levels:

- **DEBUG**: Detailed information for debugging purposes (packet details, state changes)
- **INFO**: General informational messages (startup, shutdown, normal operations)
- **WARN**: Warning messages that don't prevent operation (retries, degraded functionality)
- **ERROR**: Error messages for failures that may impact functionality

## Usage

### Initialization

The logging system must be initialized at application startup:

```go
import "github.com/mosiko1234/heimdal/sensor/internal/logger"

// Initialize with log file path and level
err := logger.Initialize("/var/log/heimdal/heimdal.log", "info")
if err != nil {
    log.Fatalf("Failed to initialize logging: %v", err)
}
```

### Component Loggers

Each component should create its own logger instance:

```go
type MyComponent struct {
    logger *logger.Logger
    // ... other fields
}

func NewMyComponent() *MyComponent {
    return &MyComponent{
        logger: logger.NewComponentLogger("MyComponent"),
    }
}
```

### Logging Messages

Use the appropriate method for each log level:

```go
// Debug messages
c.logger.Debug("Processing packet from %s", srcMAC)

// Info messages
c.logger.Info("Component started successfully")

// Warning messages
c.logger.Warn("Retry attempt %d failed: %v", attempt, err)

// Error messages
c.logger.Error("Failed to connect to database: %v", err)

// Error with context
c.logger.ErrorWithContext(err, "failed to save device %s", deviceMAC)
```

### Global Logging Functions

For backward compatibility and simple use cases, global logging functions are available:

```go
logger.Info("Application starting...")
logger.Error("Critical error: %v", err)
```

## Log Format

Log entries follow this format:

```
TIMESTAMP [LEVEL] [COMPONENT] CALLER: MESSAGE
```

Example:
```
2024-01-15 10:30:45.123 [INFO] [Database] badger_db.go:75: Database initialized successfully
2024-01-15 10:30:45.456 [WARN] [Scanner] scanner.go:142: Device channel full, dropping device update for aa:bb:cc:dd:ee:ff
2024-01-15 10:30:46.789 [ERROR] [Orchestrator] orchestrator.go:234: Failed to start component Sniffer: permission denied
```

## Output Destinations

Logs are written to two destinations simultaneously:

1. **File**: Configured log file (default: `/var/log/heimdal/heimdal.log`)
2. **Stdout**: Console output for systemd journal integration

## Configuration

Log level is configured in the main configuration file:

```json
{
  "logging": {
    "level": "info",
    "file": "/var/log/heimdal/heimdal.log"
  }
}
```

Valid log levels: `debug`, `info`, `warn`, `error`

## Best Practices

1. **Use appropriate log levels**:
   - DEBUG for detailed diagnostic information
   - INFO for normal operational messages
   - WARN for recoverable issues
   - ERROR for failures that impact functionality

2. **Include context in messages**:
   ```go
   logger.Info("Device %s discovered at IP %s", mac, ip)
   ```

3. **Use component loggers**:
   - Create a component-specific logger in each component
   - This provides automatic component identification in logs

4. **Avoid logging sensitive data**:
   - Don't log passwords, keys, or personal information
   - Use placeholders or redact sensitive fields

5. **Log at decision points**:
   - Log when entering/exiting major operations
   - Log when making important decisions
   - Log when errors occur or are recovered from

## Thread Safety

The logging system is thread-safe and can be used concurrently from multiple goroutines without additional synchronization.
