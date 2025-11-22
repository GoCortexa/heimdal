# Platform Implementation Comparison

This document compares the three desktop platform implementations (Windows, macOS, Linux) to ensure consistency and highlight platform-specific differences.

## Implementation Status

| Platform | PacketCaptureProvider | SystemIntegrator | StorageProvider | Status |
|----------|----------------------|------------------|-----------------|--------|
| Windows  | ✅ Npcap             | ✅ Windows Service | ✅ BadgerDB    | Complete |
| macOS    | ✅ libpcap           | ✅ LaunchAgent   | ✅ BadgerDB    | Complete |
| Linux    | ⏳ libpcap           | ⏳ systemd       | ⏳ BadgerDB    | Pending |

## PacketCaptureProvider Comparison

### Common Features
- All use `gopacket` library for packet parsing
- All support BPF filtering
- All parse Ethernet, IPv4/IPv6, TCP/UDP, ICMP
- All provide interface listing functionality
- All implement permission checking

### Platform-Specific Differences

| Feature | Windows | macOS | Linux |
|---------|---------|-------|-------|
| **Backend** | Npcap | libpcap (built-in) | libpcap |
| **Installation** | Requires Npcap installer | Built into OS | Requires libpcap-dev package |
| **Permission Check** | `IsNpcapInstalled()` | `CheckLibpcapPermissions()` | Capability check (CAP_NET_RAW) |
| **Permission Method** | Administrator rights | sudo or Full Disk Access | sudo or capabilities |
| **Service Check** | `sc query npcap` | `pcap.FindAllDevs()` | Capability detection |

### Code Similarities
```go
// All three platforms follow the same structure:
type PlatformPacketCapture struct {
    handle       *pcap.Handle
    packetSource *gopacket.PacketSource
    stats        platform.CaptureStats
}

func (p *PlatformPacketCapture) Open(interfaceName string, promiscuous bool, filter string) error
func (p *PlatformPacketCapture) ReadPacket() (*platform.Packet, error)
func (p *PlatformPacketCapture) Close() error
func (p *PlatformPacketCapture) GetStats() (*platform.CaptureStats, error)
```

## SystemIntegrator Comparison

### Common Features
- All support Install/Uninstall/Start/Stop/Restart
- All support auto-start configuration
- All provide service status monitoring
- All handle service recovery on crash

### Platform-Specific Differences

| Feature | Windows | macOS | Linux |
|---------|---------|-------|-------|
| **Service Type** | Windows Service | LaunchAgent/LaunchDaemon | systemd service |
| **Config Format** | Registry + API | XML plist | systemd unit file |
| **Config Location** | Service Manager | `~/Library/LaunchAgents/` or `/Library/LaunchDaemons/` | `/etc/systemd/system/` |
| **Management Tool** | `sc.exe` / Service API | `launchctl` | `systemctl` |
| **Auto-start** | StartType=Automatic | RunAtLoad=true | WantedBy=multi-user.target |
| **User vs System** | Service runs as LocalSystem | LaunchAgent (user) vs LaunchDaemon (system) | systemd user vs system service |

### Service File Examples

**Windows (API-based):**
```go
svcConfig := mgr.Config{
    DisplayName: "Heimdal Desktop",
    StartType:   mgr.StartManual,
}
s, err := m.CreateService(serviceName, exePath, svcConfig)
```

**macOS (plist):**
```xml
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.heimdal.desktop</string>
    <key>RunAtLoad</key>
    <false/>
</dict>
</plist>
```

**Linux (systemd unit):**
```ini
[Unit]
Description=Heimdal Desktop

[Service]
Type=simple
ExecStart=/opt/heimdal-desktop/bin/heimdal-desktop

[Install]
WantedBy=multi-user.target
```

## StorageProvider Comparison

### Common Features
- All use BadgerDB embedded database
- All implement the same interface methods
- All support batch operations
- All provide garbage collection
- All use platform-specific default paths

### Platform-Specific Differences

| Feature | Windows | macOS | Linux |
|---------|---------|-------|-------|
| **Default Path** | `%APPDATA%\Heimdal\db` | `~/Library/Application Support/Heimdal/db` | `~/.local/share/heimdal/db` |
| **Example Path** | `C:\Users\John\AppData\Roaming\Heimdal\db` | `/Users/john/Library/Application Support/Heimdal/db` | `/home/john/.local/share/heimdal/db` |
| **Path Function** | `os.Getenv("APPDATA")` | `os.UserHomeDir()` + Library path | `os.UserHomeDir()` + .local/share |

### Code Similarities
```go
// All three platforms follow identical structure:
type PlatformStorage struct {
    db *badger.DB
}

func (s *PlatformStorage) Open(path string, options *platform.StorageOptions) error
func (s *PlatformStorage) Close() error
func (s *PlatformStorage) Get(key string) ([]byte, error)
func (s *PlatformStorage) Set(key string, value []byte) error
func (s *PlatformStorage) Delete(key string) error
func (s *PlatformStorage) List(prefix string) ([]string, error)
func (s *PlatformStorage) Batch(ops []platform.BatchOp) error
```

## Build Tags

All platform implementations use Go build tags to ensure platform-specific compilation:

```go
// Windows
// +build windows

// macOS
// +build darwin

// Linux
// +build linux
```

## Testing Approach

All platforms follow the same testing pattern:

1. **Interface Compliance Tests**: Verify implementation satisfies interface
2. **Constructor Tests**: Verify constructors return non-nil instances
3. **Path Tests**: Verify default paths are correct for platform
4. **Feature Tests**: Test platform-specific features (permission checks, etc.)

## Error Handling Patterns

All platforms follow consistent error handling:

```go
// Check for nil/uninitialized state
if p.handle == nil {
    return fmt.Errorf("not initialized")
}

// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to open interface: %w", err)
}

// Provide user-friendly guidance
if !hasPermission {
    return fmt.Errorf("insufficient permissions. %s", GetPermissionGuidance())
}
```

## Permission Requirements Summary

| Platform | Packet Capture | Service Management | Storage |
|----------|---------------|-------------------|---------|
| **Windows** | Administrator | Administrator | User |
| **macOS** | Administrator or Full Disk Access | Administrator (for install) | User |
| **Linux** | CAP_NET_RAW + CAP_NET_ADMIN or sudo | root (for system service) | User |

## Integration with Desktop Orchestrator

All three platforms will be used by the desktop orchestrator through the same interface:

```go
// Platform selection at runtime
var packetCapture platform.PacketCaptureProvider
var systemIntegrator platform.SystemIntegrator
var storage platform.StorageProvider

switch runtime.GOOS {
case "windows":
    packetCapture = desktop_windows.NewWindowsPacketCapture()
    systemIntegrator = desktop_windows.NewWindowsSystemIntegrator()
    storage = desktop_windows.NewWindowsStorage()
case "darwin":
    packetCapture = desktop_macos.NewMacOSPacketCapture()
    systemIntegrator = desktop_macos.NewMacOSSystemIntegrator()
    storage = desktop_macos.NewMacOSStorage()
case "linux":
    packetCapture = desktop_linux.NewLinuxPacketCapture()
    systemIntegrator = desktop_linux.NewLinuxSystemIntegrator()
    storage = desktop_linux.NewLinuxStorage()
}
```

## Consistency Checklist

✅ All platforms implement the same interfaces
✅ All platforms follow the same struct patterns
✅ All platforms use the same error handling approach
✅ All platforms provide permission checking
✅ All platforms support interface listing (packet capture)
✅ All platforms use BadgerDB for storage
✅ All platforms follow platform conventions for paths
✅ All platforms include comprehensive tests
✅ All platforms include documentation

## Future Considerations

### Cross-Platform Features to Add
- Unified logging across all platforms
- Consistent metrics collection
- Unified configuration validation
- Common utility functions for path handling

### Platform-Specific Enhancements
- **Windows**: MSI installer support, Event Log integration
- **macOS**: Keychain integration, Notification Center support
- **Linux**: AppImage support, multiple init system support (systemd, OpenRC)

## Conclusion

The platform implementations maintain a high degree of consistency while respecting platform-specific conventions and requirements. This design allows the core Heimdal logic to remain platform-agnostic while providing native integration on each supported operating system.
