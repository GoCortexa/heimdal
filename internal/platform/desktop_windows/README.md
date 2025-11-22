# Windows Desktop Platform Implementation

This package provides Windows-specific implementations of the Heimdal platform interfaces.

## Components

### PacketCaptureProvider (`packet_capture.go`)

Implements packet capture for Windows using gopacket with Npcap.

**Features:**
- Npcap installation detection
- Automatic installation guidance
- Support for promiscuous mode
- BPF filtering support
- Comprehensive packet parsing (Ethernet, IPv4, IPv6, TCP, UDP, ICMP)
- Capture statistics tracking

**Requirements:**
- Npcap must be installed (https://npcap.com/)
- Administrator privileges for packet capture
- WinPcap API-compatible mode enabled during Npcap installation

**Usage:**
```go
capture := desktop_windows.NewWindowsPacketCapture()

// Check if Npcap is installed
if !desktop_windows.IsNpcapInstalled() {
    fmt.Println(desktop_windows.GetNpcapInstallationGuidance())
    return
}

// Open capture on interface
err := capture.Open("\\Device\\NPF_{GUID}", true, "tcp port 80")
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

Implements Windows Service management using the Windows Service API.

**Features:**
- Service installation and uninstallation
- Service start, stop, and restart
- Service status queries
- Auto-start configuration
- Recovery actions (automatic restart on failure)
- Administrator privilege detection

**Requirements:**
- Administrator privileges for service management
- Windows Service API access

**Usage:**
```go
integrator := desktop_windows.NewWindowsSystemIntegrator()

// Check administrator privileges
if !desktop_windows.IsAdministrator() {
    log.Fatal("Administrator privileges required")
}

// Install service
config := &platform.InstallConfig{
    ServiceName:    "HeimdалDesktop",
    DisplayName:    "Heimdal Desktop Agent",
    Description:    "Network monitoring and security agent",
    ExecutablePath: "C:\\Program Files\\Heimdal\\heimdal-desktop.exe",
    Arguments:      []string{"--service"},
    WorkingDir:     "C:\\Program Files\\Heimdal",
}

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

Implements persistent storage for Windows using BadgerDB with Windows-specific paths.

**Features:**
- BadgerDB integration
- Windows-specific default path (%APPDATA%/Heimdal/db)
- Atomic batch operations
- Key prefix listing
- Garbage collection support
- Database size tracking

**Default Storage Location:**
- `%APPDATA%\Heimdal\db` (typically `C:\Users\<username>\AppData\Roaming\Heimdal\db`)

**Usage:**
```go
storage := desktop_windows.NewWindowsStorage()

// Open with default path
err := storage.Open("", &platform.StorageOptions{
    ReadOnly:   false,
    SyncWrites: true,
    CacheSize:  100 * 1024 * 1024, // 100MB cache
})
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
data, err := storage.Get("device:mac1")
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

// Run garbage collection periodically
err = storage.RunGarbageCollection()
if err != nil {
    log.Println("GC warning:", err)
}
```

## Build Tags

All files in this package use the `// +build windows` build tag to ensure they are only compiled on Windows systems.

## Dependencies

- `github.com/google/gopacket` - Packet capture and parsing
- `github.com/dgraph-io/badger/v4` - Embedded key-value database
- `golang.org/x/sys/windows` - Windows system APIs

## Testing

Due to platform-specific nature, these implementations should be tested on actual Windows systems. Mock implementations are available in `test/mocks/` for testing core logic without platform dependencies.

## Requirements Validation

This implementation satisfies the following requirements from the specification:

- **1.6**: Platform-specific implementations in internal/platform/desktop_windows
- **2.3**: Desktop packet capture using gopacket with Npcap
- **6.1**: PacketCaptureProvider interface implementation
- **6.2**: SystemIntegrator interface implementation
- **6.4**: Windows Service implementation
- **6.7**: StorageProvider interface implementation
- **7.1**: Npcap installation verification and guidance
- **13.4**: Windows-specific storage paths (%APPDATA%)

## Security Considerations

- **Administrator Privileges**: Required for service management and packet capture
- **Npcap Security**: Npcap requires administrator installation and provides secure packet capture
- **Storage Permissions**: Database files are stored in user's AppData with appropriate permissions
- **Service Security**: Service runs as LocalSystem by default (configurable via InstallConfig.User)

## Troubleshooting

### Npcap Not Found
- Ensure Npcap is installed from https://npcap.com/
- Verify "WinPcap API-compatible Mode" is enabled
- Restart the application after Npcap installation

### Service Installation Fails
- Verify administrator privileges
- Check if service name is already in use
- Ensure executable path is correct and accessible

### Storage Access Errors
- Verify %APPDATA% environment variable is set
- Check disk space availability
- Ensure write permissions to AppData directory

## Future Enhancements

- Support for multiple network interfaces
- Enhanced service recovery options
- Storage encryption at rest
- Performance monitoring and metrics
- Integration with Windows Event Log
