# Logging and Error Handling Implementation

## Overview

This document describes the comprehensive logging and error handling system implemented for the Heimdal sensor application.

## Components Implemented

### 1. Structured Logging System (`internal/logger/`)

A custom logging package that provides:

- **Multiple Log Levels**: DEBUG, INFO, WARN, ERROR
- **Component-Specific Loggers**: Each component gets its own logger with automatic component identification
- **Dual Output**: Logs to both file and stdout simultaneously
- **Thread-Safe**: Safe for concurrent use across goroutines
- **Contextual Information**: Includes timestamp, log level, component name, and caller location

**Key Features:**
- Structured log format: `TIMESTAMP [LEVEL] [COMPONENT] CALLER: MESSAGE`
- Global logger initialization for application-wide configuration
- Component loggers for fine-grained control
- Configurable log levels via configuration file

### 2. Error Handling System (`internal/errors/`)

A comprehensive error handling package that provides:

- **Retry Logic with Exponential Backoff**: Automatic retry for transient errors
- **Error Wrapping**: Add context while preserving original errors
- **Component Errors**: Track which component generated errors
- **Recoverable Errors**: Distinguish between recoverable and fatal errors
- **Safe Resource Cleanup**: Utilities for safely closing resources

**Key Features:**
- Configurable retry behavior (max attempts, delays, backoff factor)
- Automatic logging of retry attempts
- Error context preservation
- Type-safe error checking

## Integration Points

### Main Application (`cmd/heimdal/main.go`)

- Replaced standard `log` package with structured logger
- Initialize logging system at startup
- Proper error handling with context

### Orchestrator (`internal/orchestrator/orchestrator.go`)

- Component-specific logger
- Retry logic for database initialization
- Improved error messages with context
- Safe resource cleanup on shutdown

### Database Manager (`internal/database/badger_db.go`)

- Component logger for all database operations
- Retry logic for database writes (3 attempts with exponential backoff)
- Enhanced error messages with device/profile context
- Logging of buffer usage and errors

### Network Auto-Config (`internal/netconfig/autoconfig.go`)

- Component logger for network detection
- Detailed logging of network discovery process
- Error wrapping with context

### Device Discovery Scanner (`internal/discovery/scanner.go`)

- Component logger for discovery operations
- Logging of device lifecycle events
- Debug logging for detailed diagnostics
- Warning logs for channel overflow

## Configuration

Logging is configured via the main configuration file:

```json
{
  "logging": {
    "level": "info",
    "file": "/var/log/heimdal/heimdal.log"
  }
}
```

Valid log levels: `debug`, `info`, `warn`, `error`

## Usage Examples

### Component Logger

```go
type MyComponent struct {
    logger *logger.Logger
}

func NewMyComponent() *MyComponent {
    return &MyComponent{
        logger: logger.NewComponentLogger("MyComponent"),
    }
}

func (c *MyComponent) DoWork() {
    c.logger.Info("Starting work...")
    c.logger.Debug("Processing item %d", itemID)
    c.logger.Warn("Retry attempt %d", attempt)
    c.logger.Error("Operation failed: %v", err)
}
```

### Error Handling with Retry

```go
err := errors.RetryWithBackoff("database write", errors.DefaultRetryConfig(), func() error {
    return db.Write(data)
})
if err != nil {
    return errors.Wrap(err, "failed to save data")
}
```

### Safe Resource Cleanup

```go
defer errors.SafeClose(connection, "network connection")
```

## Log Output Examples

```
2024-01-15 10:30:45.123 [INFO] [main] main.go:45: === Heimdal Sensor v2.0.0 ===
2024-01-15 10:30:45.124 [INFO] [main] main.go:46: Configuration loaded successfully
2024-01-15 10:30:45.125 [INFO] [Orchestrator] orchestrator.go:67: === Heimdal Sensor Starting ===
2024-01-15 10:30:45.126 [INFO] [Orchestrator] orchestrator.go:89: Initializing components...
2024-01-15 10:30:45.127 [INFO] [Database] badger_db.go:75: Opening database at /var/lib/heimdal/db
2024-01-15 10:30:45.234 [INFO] [Database] badger_db.go:95: Database initialized successfully
2024-01-15 10:30:45.235 [INFO] [NetConfig] autoconfig.go:42: Starting network detection...
2024-01-15 10:30:45.345 [DEBUG] [NetConfig] autoconfig.go:67: Found primary interface: eth0 with IP 192.168.1.100
2024-01-15 10:30:45.346 [INFO] [NetConfig] autoconfig.go:52: Network detected successfully: interface=eth0, ip=192.168.1.100, gateway=192.168.1.1
2024-01-15 10:30:45.347 [INFO] [Scanner] scanner.go:67: Starting device discovery scanner...
2024-01-15 10:30:45.348 [INFO] [Scanner] scanner.go:141: Loaded 5 existing devices from database
2024-01-15 10:30:45.349 [INFO] [Scanner] scanner.go:85: Device discovery scanner started (scan interval: 1m0s)
```

## Benefits

1. **Improved Debugging**: Structured logs with component identification make it easy to trace issues
2. **Operational Visibility**: Clear logging of all major operations and state changes
3. **Resilience**: Automatic retry logic handles transient failures
4. **Error Context**: Error wrapping preserves full error chain with context
5. **Production Ready**: Dual output (file + stdout) works with systemd journal
6. **Performance**: Configurable log levels allow reducing verbosity in production

## Testing

The implementation has been verified to:
- Compile without errors
- Maintain backward compatibility with existing code
- Provide consistent logging across all components
- Handle errors gracefully with proper context

## Documentation

Comprehensive documentation is provided in:
- `internal/logger/README.md` - Logging system documentation
- `internal/errors/README.md` - Error handling system documentation

## Future Enhancements

Potential improvements for future iterations:
- Log rotation support
- Metrics collection from logs
- Remote log shipping (syslog, etc.)
- Performance profiling integration
- Structured JSON logging option
