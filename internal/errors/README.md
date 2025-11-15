# Heimdal Error Handling System

## Overview

The Heimdal sensor implements a comprehensive error handling system with retry logic, error wrapping, and component-specific error types. This ensures robust operation and clear error reporting.

## Features

- **Exponential Backoff Retry**: Automatic retry with configurable backoff for transient errors
- **Error Wrapping**: Add context to errors while preserving the original error
- **Component Errors**: Track which component generated an error
- **Recoverable Errors**: Distinguish between recoverable and fatal errors
- **Safe Resource Cleanup**: Utilities for safely closing resources

## Retry Logic

### Basic Retry with Default Configuration

```go
import "github.com/mosiko1234/heimdal/sensor/internal/errors"

err := errors.RetryWithBackoff("database connection", errors.DefaultRetryConfig(), func() error {
    return db.Connect()
})
```

### Custom Retry Configuration

```go
config := errors.RetryConfig{
    MaxAttempts:   5,
    InitialDelay:  500 * time.Millisecond,
    MaxDelay:      10 * time.Second,
    BackoffFactor: 2.0,
}

err := errors.RetryWithBackoff("API request", config, func() error {
    return makeAPIRequest()
})
```

### Default Retry Configuration

The default configuration provides:
- **MaxAttempts**: 3
- **InitialDelay**: 1 second
- **MaxDelay**: 30 seconds
- **BackoffFactor**: 2.0 (exponential)

This means:
- Attempt 1: Immediate
- Attempt 2: Wait 1 second
- Attempt 3: Wait 2 seconds

## Error Wrapping

### Basic Error Wrapping

Add context to errors while preserving the original error:

```go
err := doSomething()
if err != nil {
    return errors.Wrap(err, "failed to process device %s", deviceMAC)
}
```

### Error Wrapping with Logging

Wrap an error and automatically log it:

```go
err := doSomething()
if err != nil {
    return errors.WrapWithLog(err, "failed to initialize component %s", componentName)
}
```

## Component Errors

Track which component generated an error:

```go
err := someOperation()
if err != nil {
    return errors.NewComponentError("Scanner", "device discovery", err)
}
```

This produces errors like:
```
[Scanner] device discovery: connection timeout
```

## Recoverable Errors

Mark errors as recoverable or fatal:

```go
// Recoverable error (can retry)
return errors.NewRecoverableError(err, true)

// Fatal error (should not retry)
return errors.NewRecoverableError(err, false)

// Check if error is recoverable
if errors.IsRecoverable(err) {
    // Retry the operation
}
```

## Safe Resource Cleanup

### Safe Close (Log Errors)

Safely close a resource and log any errors:

```go
defer errors.SafeClose(file, "config file")
defer errors.SafeClose(db, "database connection")
```

### Safe Close (Return Errors)

Safely close a resource and return any errors:

```go
if err := errors.SafeCloseWithError(conn, "network connection"); err != nil {
    return err
}
```

## Usage Patterns

### Database Operations

```go
func (dm *DatabaseManager) SaveDevice(device *Device) error {
    // Serialize with error wrapping
    data, err := json.Marshal(device)
    if err != nil {
        return errors.Wrap(err, "failed to serialize device %s", device.MAC)
    }

    // Retry database write
    err = errors.RetryWithBackoff("save device", errors.RetryConfig{
        MaxAttempts:   3,
        InitialDelay:  100 * time.Millisecond,
        MaxDelay:      1 * time.Second,
        BackoffFactor: 2.0,
    }, func() error {
        return dm.db.Update(func(txn *badger.Txn) error {
            return txn.Set(key, data)
        })
    })

    if err != nil {
        // Buffer in memory and wrap error
        dm.bufferDevice(device)
        return errors.Wrap(err, "failed to save device (buffered in memory)")
    }

    return nil
}
```

### Network Operations

```go
func (c *Connector) Connect() error {
    err := errors.RetryWithBackoff("cloud connection", errors.DefaultRetryConfig(), func() error {
        return c.establishConnection()
    })

    if err != nil {
        return errors.NewComponentError("CloudConnector", "connect", err)
    }

    return nil
}
```

### Component Initialization

```go
func (o *Orchestrator) initializeComponents() error {
    // Initialize database with retry
    var db *database.DatabaseManager
    err := errors.RetryWithBackoff("database initialization", errors.DefaultRetryConfig(), func() error {
        var err error
        db, err = database.NewDatabaseManager(o.config.Database.Path)
        return err
    })
    if err != nil {
        return errors.Wrap(err, "failed to initialize database")
    }
    o.db = db

    // Initialize other components...
    return nil
}
```

### Resource Cleanup

```go
func (c *Component) Stop() error {
    // Close multiple resources safely
    errors.SafeClose(c.connection, "network connection")
    errors.SafeClose(c.file, "log file")
    
    // Close with error checking
    if err := errors.SafeCloseWithError(c.db, "database"); err != nil {
        return err
    }

    return nil
}
```

## Error Logging Integration

The error handling system integrates with the logging system:

```go
// Retry operations automatically log warnings
err := errors.RetryWithBackoff("operation", config, fn)
// Logs: "Operation 'operation' failed (attempt 1/3): error. Retrying in 1s..."

// WrapWithLog automatically logs errors
err = errors.WrapWithLog(err, "context")
// Logs: "[ERROR] context: original error"
```

## Best Practices

1. **Always wrap errors with context**:
   ```go
   return errors.Wrap(err, "failed to process device %s", mac)
   ```

2. **Use retry for transient errors**:
   - Network operations
   - Database operations
   - External service calls

3. **Don't retry for permanent errors**:
   - Invalid input
   - Configuration errors
   - Permission denied

4. **Use component errors for clarity**:
   ```go
   return errors.NewComponentError("Scanner", "ARP scan", err)
   ```

5. **Clean up resources safely**:
   ```go
   defer errors.SafeClose(resource, "resource name")
   ```

6. **Check error types when needed**:
   ```go
   if errors.IsRecoverable(err) {
       // Retry logic
   }
   ```

## Thread Safety

All error handling functions are thread-safe and can be used concurrently from multiple goroutines.
