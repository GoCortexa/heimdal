# macOS Desktop Platform Implementation

This package provides macOS-specific implementations of the Heimdal platform abstraction interfaces for the desktop product.

## Components

### PacketCaptureProvider (`packet_capture.go`)

Implements packet capture for macOS using `gopacket` with `libpcap`.

**Key Features:**
- Uses native macOS libpcap (built into the OS)
- Permission detection and guidance for packet capture
- Support for promiscuous mode
- BPF filtering support
- Comprehensive packet parsing (Ethernet, IPv4/IPv6, TCP/UDP, ICMP)

**Permission Requirements:**
- Requires administrator privileges for packet capture
- Users can either:
  1. Run with `sudo` (temporary)
  2. Grant "Full Disk Access" in System Preferences (permanent)

**Usage Example:**
```go
capture := desktop_macos.NewMacOSPacketCapture()

// Check permissions before opening
hasPermission, err := desktop_macos.CheckLibpcapPermissions()
if !hasPermission {
    fmt.Println(desktop_macos.GetLibpcapPermissionGuidance())
    return
}

// Open interface
err = capture.Open("en0", true, "tcp or udp")
if err != nil {
    log.Fatal(err)
}
defer capture.Close()

// Read packets
for {
    packet, err := capture.ReadPacket()
    if err != nil {
        break
    }
    // Process packet...
}
```

### SystemIntegrator (`system_integrator.go`)

Implements macOS service management using LaunchAgent/LaunchDaemon plist files.

**Key Features:**
- LaunchAgent support (user-level services)
- LaunchDaemon support (system-level services)
- Auto-start configuration via RunAtLoad
- Service status monitoring
- Automatic restart on crash
- Standard output/error logging

**LaunchAgent vs LaunchDaemon:**
- **LaunchAgent**: Runs when user logs in, stored in `~/Library/LaunchAgents/`
- **LaunchDaemon**: Runs at system boot, stored in `/Library/LaunchDaemons/`, requires root

**Usage Example:**
```go
// Create user-level LaunchAgent
integrator := desktop_macos.NewMacOSSystemIntegrator()

// Or create system-level LaunchDaemon
// integrator := desktop_macos.NewMacOSSystemIntegratorDaemon()

config := &platform.InstallConfig{
    ServiceName:    "com.heimdal.desktop",
    DisplayName:    "Heimdal Desktop",
    Description:    "Heimdal Network Security Monitor",
    ExecutablePath: "/Applications/Heimdal.app/Contents/MacOS/heimdal-desktop",
    Arguments:      []string{"--daemon"},
    WorkingDir:     "/tmp",
}

// Install service
err := integrator.Install(config)
if err != nil {
    log.Fatal(err)
}

// Start service
err = integrator.Start()
if err != nil {
    log.Fatal(err)
}

// Enable auto-start
err = integrator.EnableAutoStart(true)
if err != nil {
    log.Fatal(err)
}
```

### StorageProvider (`storage.go`)

Implements persistent storage for macOS using BadgerDB with macOS-specific paths.

**Key Features:**
- BadgerDB embedded key-value store
- Default path: `~/Library/Application Support/Heimdal/db`
- Atomic batch operations
- Garbage collection support
- Database size monitoring

**Usage Example:**
```go
storage := desktop_macos.NewMacOSStorage()

// Open with default path
err := storage.Open("", nil)
if err != nil {
    log.Fatal(err)
}
defer storage.Close()

// Store data
err = storage.Set("device:mac1", []byte(`{"ip":"192.168.1.100"}`))
if err != nil {
    log.Fatal(err)
}

// Retrieve data
value, err := storage.Get("device:mac1")
if err != nil {
    log.Fatal(err)
}

// List keys with prefix
keys, err := storage.List("device:")
if err != nil {
    log.Fatal(err)
}

// Batch operations
ops := []platform.BatchOp{
    {Type: platform.BatchOpSet, Key: "key1", Value: []byte("value1")},
    {Type: platform.BatchOpSet, Key: "key2", Value: []byte("value2")},
    {Type: platform.BatchOpDelete, Key: "key3"},
}
err = storage.Batch(ops)
if err != nil {
    log.Fatal(err)
}
```

## Build Tags

All files in this package use the `// +build darwin` build tag to ensure they only compile on macOS.

## Dependencies

- `github.com/google/gopacket` - Packet capture and parsing
- `github.com/dgraph-io/badger/v4` - Embedded database
- Standard library packages for system integration

## Testing

To test these implementations on macOS:

```bash
# Run all tests
go test -v ./internal/platform/desktop_macos/...

# Run with build tags
go test -v -tags=darwin ./internal/platform/desktop_macos/...
```

## Platform-Specific Notes

### Packet Capture Permissions

macOS requires special permissions for packet capture:

1. **System Integrity Protection (SIP)**: Enabled by default, restricts low-level system access
2. **Full Disk Access**: Required for packet capture in recent macOS versions
3. **Administrator Privileges**: Required for libpcap operations

### LaunchAgent/LaunchDaemon

- LaunchAgents run in user context and have access to GUI
- LaunchDaemons run at system level and start before user login
- Both use XML plist files for configuration
- `launchctl` command manages loading/unloading services

### File Paths

Standard macOS application paths:
- Application: `/Applications/Heimdal.app/`
- User data: `~/Library/Application Support/Heimdal/`
- User logs: `~/Library/Logs/Heimdal/`
- LaunchAgent: `~/Library/LaunchAgents/`
- LaunchDaemon: `/Library/LaunchDaemons/`
- System logs: `/var/log/heimdal/`

## Requirements Validation

This implementation satisfies the following requirements from the design document:

- **Requirement 1.6**: Platform-specific implementations in `internal/platform/desktop_macos/`
- **Requirement 2.3**: Desktop packet capture using gopacket with libpcap
- **Requirement 6.1**: PacketCaptureProvider interface implementation
- **Requirement 6.2**: SystemIntegrator interface implementation
- **Requirement 6.5**: LaunchAgent implementation for macOS
- **Requirement 6.7**: StorageProvider interface implementation
- **Requirement 7.3**: libpcap permission detection and request
- **Requirement 13.4**: macOS-specific storage paths (`~/Library/Application Support`)

## Future Enhancements

Potential improvements for future versions:

1. **Keychain Integration**: Store sensitive data in macOS Keychain
2. **Notification Center**: Native macOS notifications for alerts
3. **Menu Bar App**: System menu bar integration
4. **Sandboxing**: App Store compatibility with sandboxing
5. **Code Signing**: Proper code signing for distribution
6. **Universal Binary**: Support for both Intel and Apple Silicon
