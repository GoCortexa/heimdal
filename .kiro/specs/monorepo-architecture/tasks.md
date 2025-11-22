# Implementation Plan

- [x] 1. Repository restructuring and interface foundation
  - Create new monorepo directory structure (cmd/, internal/core/, internal/platform/, internal/hardware/, internal/desktop/)
  - Define core platform interfaces in internal/platform/interfaces.go (PacketCaptureProvider, SystemIntegrator, StorageProvider)
  - Create mock implementations for testing in test/mocks/
  - _Requirements: 1.1, 1.2, 6.1, 6.2, 6.7_

- [x] 1.1 Write property test for packet parser interface compatibility
  - **Property 1: Packet Parser Interface Compatibility**
  - **Validates: Requirements 2.4**

- [x] 2. Extract and migrate shared core modules
  - [x] 2.1 Move packet analysis logic to internal/core/packet/
    - Extract packet parsing from internal/analyzer/sniffer.go
    - Refactor to use PacketCaptureProvider interface
    - Implement Analyzer struct that works with any provider
    - _Requirements: 1.3, 2.4, 2.5_

- [x] 2.2 Write property test for packet metadata extraction
  - **Property 2: Packet Metadata Extraction Completeness**
  - **Validates: Requirements 2.5**

  - [x] 2.3 Move cloud communication to internal/core/cloud/
    - Extract cloud connector logic from internal/cloud/
    - Add DeviceType metadata to all cloud messages
    - Ensure both AWS and GCP connectors support device type
    - _Requirements: 1.4, 9.1, 9.3, 9.4, 9.5_

- [x] 2.4 Write property test for cloud message type support
  - **Property 9: Cloud Message Type Support**
  - **Validates: Requirements 9.4**

- [x] 2.5 Write property test for cloud metadata inclusion
  - **Property 10: Cloud Metadata Inclusion**
  - **Validates: Requirements 9.5**

- [x] 2.6 Write property test for cloud authentication consistency
  - **Property 11: Cloud Authentication Consistency**
  - **Validates: Requirements 9.6**

  - [x] 2.7 Create anomaly detection module in internal/core/detection/
    - Implement Detector struct with configurable sensitivity
    - Implement detection algorithms for unexpected destinations, unusual ports, traffic spikes
    - Implement Anomaly struct with severity levels
    - _Requirements: 1.5, 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 2.8 Write property test for anomaly detection pattern recognition
  - **Property 12: Anomaly Detection Pattern Recognition**
  - **Validates: Requirements 10.2, 10.3**

- [x] 2.9 Write property test for anomaly alert structure
  - **Property 13: Anomaly Alert Structure**
  - **Validates: Requirements 10.4**

- [x] 2.10 Write property test for anomaly detection sensitivity
  - **Property 14: Anomaly Detection Sensitivity**
  - **Validates: Requirements 10.5**

  - [x] 2.11 Move behavioral profiler to internal/core/profiler/
    - Extract profiler logic from internal/profiler/
    - Refactor to use StorageProvider interface
    - Maintain existing profile aggregation logic
    - _Requirements: 4.4_

- [x] 3. Implement hardware platform implementations
  - [x] 3.1 Create linux_embedded PacketCaptureProvider
    - Implement raw socket or AF_PACKET packet capture
    - Implement Open, ReadPacket, Close, GetStats methods
    - Place in internal/platform/linux_embedded/packet_capture.go
    - _Requirements: 1.6, 2.2, 6.1_

  - [x] 3.2 Create linux_embedded SystemIntegrator
    - Implement systemd service management
    - Implement Install, Uninstall, Start, Stop, GetStatus methods
    - Place in internal/platform/linux_embedded/system_integrator.go
    - _Requirements: 1.6, 6.2, 6.3_

  - [x] 3.3 Refactor hardware orchestrator to use interfaces
    - Update internal/hardware/orchestrator/ to use platform interfaces
    - Inject platform implementations at initialization
    - Maintain existing component coordination logic
    - _Requirements: 9.1_

- [x] 4. Checkpoint - Verify hardware product still works
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement desktop platform implementations for Windows
  - [x] 5.1 Create desktop_windows PacketCaptureProvider
    - Implement gopacket with Npcap
    - Add Npcap installation detection
    - Implement Open, ReadPacket, Close, GetStats methods
    - Place in internal/platform/desktop_windows/packet_capture.go
    - _Requirements: 1.6, 2.3, 6.1, 7.1_

  - [x] 5.2 Create desktop_windows SystemIntegrator
    - Implement Windows Service API integration
    - Implement Install, Uninstall, Start, Stop, GetStatus methods
    - Place in internal/platform/desktop_windows/system_integrator.go
    - _Requirements: 1.6, 6.2, 6.4_

  - [x] 5.3 Create desktop_windows StorageProvider
    - Implement BadgerDB wrapper with Windows-specific paths (%APPDATA%)
    - Implement Open, Close, Get, Set, Delete, List, Batch methods
    - Place in internal/platform/desktop_windows/storage.go
    - _Requirements: 6.7, 13.4_

- [x] 6. Implement desktop platform implementations for macOS
  - [x] 6.1 Create desktop_macos PacketCaptureProvider
    - Implement gopacket with libpcap
    - Add libpcap permission detection and request
    - Implement Open, ReadPacket, Close, GetStats methods
    - Place in internal/platform/desktop_macos/packet_capture.go
    - _Requirements: 1.6, 2.3, 6.1, 7.3_

  - [x] 6.2 Create desktop_macos SystemIntegrator
    - Implement LaunchAgent plist management
    - Implement Install, Uninstall, Start, Stop, GetStatus methods
    - Place in internal/platform/desktop_macos/system_integrator.go
    - _Requirements: 1.6, 6.2, 6.5_

  - [x] 6.3 Create desktop_macos StorageProvider
    - Implement BadgerDB wrapper with macOS-specific paths (~/Library/Application Support)
    - Implement Open, Close, Get, Set, Delete, List, Batch methods
    - Place in internal/platform/desktop_macos/storage.go
    - _Requirements: 6.7, 13.4_

- [-] 7. Implement desktop platform implementations for Linux
  - [x] 7.1 Create desktop_linux PacketCaptureProvider
    - Implement gopacket with libpcap
    - Add capability detection (CAP_NET_RAW, CAP_NET_ADMIN)
    - Implement Open, ReadPacket, Close, GetStats methods
    - Place in internal/platform/desktop_linux/packet_capture.go
    - _Requirements: 1.6, 2.3, 6.1, 7.5_

  - [x] 7.2 Create desktop_linux SystemIntegrator
    - Implement systemd user service management
    - Implement Install, Uninstall, Start, Stop, GetStatus methods
    - Place in internal/platform/desktop_linux/system_integrator.go
    - _Requirements: 1.6, 6.2, 6.6_

  - [x] 7.3 Create desktop_linux StorageProvider
    - Implement BadgerDB wrapper with Linux-specific paths (~/.local/share)
    - Implement Open, Close, Get, Set, Delete, List, Batch methods
    - Place in internal/platform/desktop_linux/storage.go
    - _Requirements: 6.7, 13.4_

- [x] 8. Implement desktop-specific feature gate module
  - [x] 8.1 Create FeatureGate core logic
    - Implement Tier enum (Free, Pro, Enterprise)
    - Implement Feature enum (NetworkVisibility, TrafficBlocking, etc.)
    - Implement CanAccess method with tier-based logic
    - Place in internal/desktop/featuregate/feature_gate.go
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 8.2 Write property test for feature gate access control
  - **Property 7: Feature Gate Access Control**
  - **Validates: Requirements 5.4**

- [x] 8.3 Write property test for feature gate error messages
  - **Property 8: Feature Gate Error Messages**
  - **Validates: Requirements 5.5**

  - [x] 8.4 Implement license validation
    - Implement LicenseValidator interface
    - Implement local license key validation
    - Implement cloud-based license validation
    - Place in internal/desktop/featuregate/license.go
    - _Requirements: 5.6_

  - [x] 8.5 Implement configuration loading for feature gates
    - Load tier from configuration file
    - Load license key from configuration
    - Validate on startup
    - Place in internal/desktop/featuregate/config.go
    - _Requirements: 5.6, 13.2_

- [x] 9. Implement desktop traffic interceptor
  - [x] 9.1 Create DesktopTrafficInterceptor
    - Implement ARP spoofing from host endpoint
    - Implement IP forwarding verification
    - Implement safety checks for network stack
    - Place in internal/desktop/interceptor/arp_spoofer.go
    - _Requirements: 3.1, 3.2, 3.4_

- [x] 9.2 Write property test for ARP spoofing packet correctness
  - **Property 3: ARP Spoofing Packet Correctness**
  - **Validates: Requirements 3.1**

  - [x] 9.3 Implement ARP table restoration
    - Implement graceful cleanup on shutdown
    - Implement cleanup on crash (signal handlers)
    - Store original ARP entries before spoofing
    - _Requirements: 3.5_

  - [x] 9.4 Implement platform-specific permission handling
    - Check for administrator rights on Windows
    - Check for sudo/capabilities on Linux/macOS
    - Provide clear error messages for missing permissions
    - _Requirements: 3.6_

- [x] 10. Implement local visualizer and web dashboard
  - [x] 10.1 Create LocalVisualizer HTTP server
    - Implement HTTP server with configurable port
    - Implement graceful shutdown
    - Place in internal/desktop/visualizer/server.go
    - _Requirements: 4.1_

  - [x] 10.2 Implement device API endpoints
    - Implement GET /api/v1/devices (list all devices)
    - Implement GET /api/v1/devices/:mac (device details)
    - Implement GET /api/v1/profiles/:mac (behavioral profile)
    - Place in internal/desktop/visualizer/api.go
    - _Requirements: 4.2, 4.5_

- [x] 10.3 Write property test for device API response completeness
  - **Property 4: Device API Response Completeness**
  - **Validates: Requirements 4.2**

- [x] 10.4 Write property test for API endpoint JSON validity
  - **Property 6: API Endpoint JSON Validity**
  - **Validates: Requirements 4.5**

  - [x] 10.5 Implement WebSocket hub for real-time updates
    - Implement WebSocketHub with client management
    - Implement broadcast mechanism for updates
    - Implement connection/disconnection handling
    - Place in internal/desktop/visualizer/websocket.go
    - _Requirements: 4.4_

- [x] 10.6 Write property test for real-time update propagation
  - **Property 5: Real-time Update Propagation**
  - **Validates: Requirements 4.4**

  - [x] 10.7 Create web dashboard frontend
    - Implement network map visualization
    - Implement device list with details
    - Implement real-time updates via WebSocket
    - Place in web/dashboard/ (reuse existing dashboard with enhancements)
    - _Requirements: 4.2, 4.3, 4.4_

  - [x] 10.8 Integrate feature gate with visualizer
    - Check feature access before serving protected endpoints
    - Return appropriate errors for insufficient tier
    - Display tier information in dashboard
    - _Requirements: 5.4, 5.5_

- [x] 11. Implement system tray integration
  - [x] 11.1 Create SystemTray abstraction
    - Define SystemTray interface
    - Define Status enum (Active, Paused, Error)
    - Define menu structure
    - Place in internal/desktop/systray/systray.go
    - _Requirements: 12.1, 12.2, 12.3_

  - [x] 11.2 Implement Windows system tray
    - Use systray library for Windows
    - Implement icon display and menu
    - Implement status icon changes
    - Place in internal/desktop/systray/systray_windows.go
    - _Requirements: 12.1, 12.2, 12.3_

  - [x] 11.3 Implement macOS menu bar
    - Use systray library for macOS
    - Implement menu bar icon and menu
    - Implement status icon changes
    - Place in internal/desktop/systray/systray_darwin.go
    - _Requirements: 12.1, 12.2, 12.3_

  - [x] 11.4 Implement Linux system tray
    - Use systray library for Linux
    - Implement system tray icon and menu
    - Implement status icon changes
    - Place in internal/desktop/systray/systray_linux.go
    - _Requirements: 12.1, 12.2, 12.3_

  - [x] 11.5 Implement desktop notifications
    - Implement notification triggering for new devices
    - Implement notification triggering for anomalies
    - Use platform-specific notification APIs
    - Place in internal/desktop/systray/notifications.go
    - _Requirements: 12.4_

- [x] 11.6 Write property test for event notification triggering
  - **Property 16: Event Notification Triggering**
  - **Validates: Requirements 12.4**

  - [x] 11.7 Implement auto-start configuration
    - Implement auto-start enable/disable
    - Use SystemIntegrator for platform-specific implementation
    - Store preference in configuration
    - _Requirements: 12.5_

- [x] 12. Implement desktop orchestrator
  - [x] 12.1 Create desktop orchestrator
    - Implement component initialization with platform interfaces
    - Implement component lifecycle management
    - Implement graceful shutdown
    - Place in internal/desktop/orchestrator/orchestrator.go
    - _Requirements: 9.1, 9.2, 9.3_

  - [x] 12.2 Integrate all desktop components
    - Initialize packet capture provider
    - Initialize traffic interceptor
    - Initialize feature gate
    - Initialize local visualizer
    - Initialize system tray
    - Wire components together with channels
    - _Requirements: 9.1, 9.2_

  - [x] 12.3 Implement desktop configuration management
    - Define DesktopConfig struct
    - Implement configuration loading from platform-specific paths
    - Implement configuration validation
    - Place in internal/desktop/config/config.go
    - _Requirements: 13.2, 13.4_

- [x] 12.4 Write property test for configuration validation
  - **Property 17: Configuration Validation**
  - **Validates: Requirements 13.5**

- [x] 12.5 Write property test for configuration hot-reload
  - **Property 18: Configuration Hot-Reload**
  - **Validates: Requirements 13.6**

- [x] 13. Create desktop entry point
  - [x] 13.1 Implement cmd/heimdal-desktop/main.go
    - Initialize desktop orchestrator
    - Load platform-specific implementations
    - Handle command-line arguments
    - Implement signal handling
    - _Requirements: 1.2_

  - [x] 13.2 Implement onboarding wizard
    - Create first-run detection
    - Implement permission explanation UI
    - Implement dependency verification (Npcap, libpcap)
    - Implement network interface selection
    - Place in internal/desktop/installer/onboarding.go
    - _Requirements: 11.2, 11.3, 11.4_

- [x] 13.3 Write property test for network interface auto-detection
  - **Property 15: Network Interface Auto-Detection**
  - **Validates: Requirements 11.5**

- [x] 14. Checkpoint - Verify desktop product works on all platforms
  - Ensure all tests pass, ask the user if questions arise.

- [x] 15. Implement build system
  - [x] 15.1 Create Makefile with build targets
    - Implement build-hardware target (ARM64 Linux)
    - Implement build-desktop-windows target
    - Implement build-desktop-macos target (amd64 and arm64)
    - Implement build-desktop-linux target
    - Implement build-all target
    - _Requirements: 8.1, 8.2, 8.3_

  - [x] 15.2 Implement test targets in Makefile
    - Implement test target (all tests)
    - Implement test-property target (property-based tests)
    - Implement test-integration target
    - Implement test-platform targets (Windows, macOS, Linux)
    - _Requirements: 14.1, 14.2, 14.3_

  - [x] 15.3 Configure cross-compilation
    - Set up CGO cross-compilers
    - Configure build flags for static linking (hardware)
    - Configure build flags for GUI (desktop)
    - _Requirements: 8.3, 8.4_

- [x] 16. Implement packaging and installers
  - [x] 16.1 Create Windows installer
    - Create NSIS or WiX installer script
    - Bundle Npcap installer
    - Configure installation paths
    - Create uninstaller
    - Place in build/installers/windows/
    - _Requirements: 8.5, 11.1_

  - [x] 16.2 Create macOS installer
    - Create DMG or PKG installer
    - Configure application bundle
    - Sign application (if certificates available)
    - Place in build/installers/macos/
    - _Requirements: 8.5, 11.1_

  - [x] 16.3 Create Linux packages
    - Create .deb package script
    - Create .rpm package script
    - Configure package dependencies (libpcap-dev)
    - Place in build/package/linux/
    - _Requirements: 8.5_

- [x] 17. Implement backward compatibility
  - [x] 17.1 Support legacy configuration format
    - Implement configuration migration logic
    - Detect old configuration format
    - Convert to new format automatically
    - Log migration warnings
    - Place in internal/config/migration.go
    - _Requirements: 15.3_

- [x] 17.2 Write property test for backward configuration compatibility
  - **Property 19: Backward Configuration Compatibility**
  - **Validates: Requirements 15.3**

  - [x] 17.3 Update Ansible playbooks
    - Verify existing playbooks work with new binary
    - Update playbook for new features (optional)
    - Document migration path
    - _Requirements: 15.5_

- [x] 18. Documentation and final testing
  - [x] 18.1 Update README.md
    - Document monorepo structure
    - Document build process for both products
    - Document desktop installation process
    - Add platform-specific notes

  - [x] 18.2 Create desktop user guide
    - Document installation steps for each platform
    - Document onboarding process
    - Document feature tiers
    - Document troubleshooting

  - [x] 18.3 Create developer guide
    - Document architecture and interfaces
    - Document how to add new platform implementations
    - Document testing strategy
    - Document contribution guidelines

- [x] 18.4 Run full test suite
  - Run all unit tests
  - Run all property-based tests (100 iterations each)
  - Run all integration tests
  - Run platform-specific tests on Windows, macOS, Linux
  - Verify code coverage meets 70% for core modules

- [x] 19. Final checkpoint - Complete system verification
  - Ensure all tests pass, ask the user if questions arise.
