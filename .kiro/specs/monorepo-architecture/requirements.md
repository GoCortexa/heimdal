# Requirements Document

## Introduction

Heimdal is expanding from a single hardware-focused product (Raspberry Pi sensor) to a dual-product platform supporting both dedicated hardware sensors and desktop software agents. This expansion requires restructuring the codebase into a monorepo architecture that maximizes code reuse for shared functionality (packet analysis, cloud communication, anomaly detection) while maintaining strict separation of platform-specific concerns (deployment, OS integration, UI). The Desktop product serves as a SaaS MVP offering "Free Tier" network visibility through ARP spoofing from the user's endpoint, with potential upgrade paths to "Pro" features.

## Glossary

- **Heimdal Hardware**: Dedicated physical sensor running on Raspberry Pi (Linux) using promiscuous mode and ARP spoofing at the router level to protect entire subnets
- **Heimdal Desktop**: Software-only agent for Windows, macOS, and Linux endpoints that runs on user computers and uses ARP spoofing from the host for network visualization
- **Monorepo**: Single repository containing both product lines with shared core logic and separated platform-specific implementations
- **Core Logic**: Shared functionality including packet analysis, cloud communication, anomaly detection, and behavioral profiling
- **Platform Abstraction Layer**: Interface-based architecture allowing different implementations for hardware vs desktop platforms
- **DesktopTrafficInterceptor**: Component performing ARP spoofing from user's PC to gain visibility into network devices without crashing the OS network stack
- **LocalVisualizer**: Lightweight HTTP server providing React or Vue dashboard for network map and traffic flow visualization
- **FeatureGating**: Module enforcing tier-based limitations (Free Tier read-only visibility vs Pro active blocking)
- **PacketCaptureProvider**: Interface abstracting packet capture mechanisms (raw sockets for hardware, gopacket with PCAP/Npcap for desktop)
- **SystemIntegrator**: Interface abstracting OS-level integration (systemd for hardware, Windows Service/macOS LaunchAgent/System Tray for desktop)
- **Asgard Platform**: Cloud-based analytics platform for aggregating sensor data from both hardware and desktop deployments

## Requirements

### Requirement 1: Monorepo Project Structure

**User Story:** As a software architect, I want a clear directory structure that separates hardware and desktop applications while sharing core logic, so that the codebase remains maintainable as both products evolve.

#### Acceptance Criteria

1.1 THE Heimdal Monorepo SHALL organize code into three top-level internal directories: core (shared logic), platform (OS abstractions), and product-specific directories (hardware, desktop)

1.2 THE Heimdal Monorepo SHALL maintain separate entry points in cmd/heimdal-hardware and cmd/heimdal-desktop

1.3 THE Heimdal Monorepo SHALL place shared packet analysis logic in internal/core/packet_analysis

1.4 THE Heimdal Monorepo SHALL place shared cloud communication logic in internal/core/cloud

1.5 THE Heimdal Monorepo SHALL place shared anomaly detection logic in internal/core/detection

1.6 THE Heimdal Monorepo SHALL place platform-specific implementations in internal/platform with subdirectories for linux_embedded, desktop_windows, desktop_macos, and desktop_linux

### Requirement 2: Core Packet Analysis Abstraction

**User Story:** As a developer, I want packet analysis logic to be shared between hardware and desktop products, so that improvements benefit both platforms without code duplication.

#### Acceptance Criteria

2.1 THE Heimdal Monorepo SHALL define a PacketCaptureProvider interface with methods for opening capture handles, reading packets, and closing handles

2.2 WHEN the Hardware product initializes packet capture, THE Hardware product SHALL use a raw socket or AF_PACKET implementation of PacketCaptureProvider

2.3 WHEN the Desktop product initializes packet capture, THE Desktop product SHALL use a gopacket with PCAP (Linux/macOS) or Npcap (Windows) implementation of PacketCaptureProvider

2.4 THE Heimdal Monorepo SHALL implement shared packet parsing logic that accepts packets from any PacketCaptureProvider implementation

2.5 THE Heimdal Monorepo SHALL extract protocol information (TCP, UDP, ICMP), source/destination addresses, and ports in the shared packet analysis module

### Requirement 3: Desktop Traffic Interceptor

**User Story:** As a desktop user, I want the software to perform ARP spoofing from my computer to visualize network traffic, so that I can see what devices are communicating on my network without purchasing dedicated hardware.

#### Acceptance Criteria

3.1 THE DesktopTrafficInterceptor SHALL perform ARP spoofing from the user's endpoint to intercept traffic destined for other network devices

3.2 THE DesktopTrafficInterceptor SHALL verify that IP forwarding is enabled before starting ARP spoofing operations

3.3 WHEN ARP spoofing is active, THE DesktopTrafficInterceptor SHALL forward intercepted packets to maintain network connectivity for other devices

3.4 THE DesktopTrafficInterceptor SHALL implement safety checks to prevent OS network stack crashes during ARP spoofing

3.5 THE DesktopTrafficInterceptor SHALL gracefully restore original ARP tables when the application stops or crashes

3.6 THE DesktopTrafficInterceptor SHALL handle platform-specific permission requirements (administrator on Windows, sudo on Linux/macOS)

### Requirement 4: Local Visualization Dashboard

**User Story:** As a desktop user, I want a web-based dashboard showing my network map and traffic flows, so that I can understand what devices are on my network and how they communicate.

#### Acceptance Criteria

4.1 THE LocalVisualizer SHALL serve a web dashboard on a configurable local port (default 8080)

4.2 THE LocalVisualizer SHALL display a network map showing all discovered devices with their IP addresses, MAC addresses, and device names

4.3 THE LocalVisualizer SHALL visualize traffic flows between devices using a graph or flow diagram

4.4 THE LocalVisualizer SHALL update the dashboard in real-time as new traffic is detected

4.5 THE LocalVisualizer SHALL provide API endpoints for retrieving device information and traffic statistics in JSON format

4.6 THE LocalVisualizer SHALL use a modern frontend framework (React or Vue) for the dashboard UI

### Requirement 5: Feature Gating and Tier Management

**User Story:** As a product manager, I want to enforce feature limitations based on user subscription tier, so that we can offer a free tier with basic visibility and paid tiers with advanced features.

#### Acceptance Criteria

5.1 THE FeatureGating module SHALL define tier levels including Free, Pro, and Enterprise

5.2 WHEN a user is on the Free tier, THE FeatureGating module SHALL enable read-only network visibility without active blocking capabilities

5.3 WHEN a user is on the Pro tier, THE FeatureGating module SHALL enable active traffic blocking and advanced filtering capabilities

5.4 THE FeatureGating module SHALL check tier permissions before executing protected operations

5.5 THE FeatureGating module SHALL provide clear error messages when users attempt to access features outside their tier

5.6 THE FeatureGating module SHALL support tier configuration through local configuration files and cloud-based license validation

### Requirement 6: Platform Abstraction Interfaces

**User Story:** As a developer, I want well-defined interfaces for platform-specific operations, so that I can implement different behaviors for hardware and desktop without modifying core logic.

#### Acceptance Criteria

6.1 THE Heimdal Monorepo SHALL define a PacketCaptureProvider interface with methods: Open(interfaceName string) error, ReadPacket() (Packet, error), Close() error

6.2 THE Heimdal Monorepo SHALL define a SystemIntegrator interface with methods: Install() error, Uninstall() error, Start() error, Stop() error, GetStatus() (Status, error)

6.3 WHEN the Hardware product initializes, THE Hardware product SHALL use a systemd implementation of SystemIntegrator

6.4 WHEN the Desktop product initializes on Windows, THE Desktop product SHALL use a Windows Service implementation of SystemIntegrator

6.5 WHEN the Desktop product initializes on macOS, THE Desktop product SHALL use a LaunchAgent implementation of SystemIntegrator

6.6 WHEN the Desktop product initializes on Linux, THE Desktop product SHALL use a systemd or system tray implementation of SystemIntegrator

6.7 THE Heimdal Monorepo SHALL define a StorageProvider interface to abstract database operations across platforms

### Requirement 7: Desktop Platform-Specific Implementations

**User Story:** As a desktop user on Windows, I want the application to integrate properly with my operating system, so that it runs reliably and follows OS conventions.

#### Acceptance Criteria

7.1 WHEN the Desktop product runs on Windows, THE Desktop product SHALL verify that Npcap is installed and provide installation guidance if missing

7.2 WHEN the Desktop product runs on Windows, THE Desktop product SHALL integrate with the Windows system tray for status display and quick access

7.3 WHEN the Desktop product runs on macOS, THE Desktop product SHALL verify that libpcap permissions are granted and request them if needed

7.4 WHEN the Desktop product runs on macOS, THE Desktop product SHALL integrate with the macOS menu bar for status display

7.5 WHEN the Desktop product runs on Linux, THE Desktop product SHALL verify that the user has appropriate capabilities or sudo access for packet capture

7.6 THE Desktop product SHALL handle platform-specific permission elevation gracefully with clear user prompts

### Requirement 8: Build and Distribution System

**User Story:** As a DevOps engineer, I want a unified build system that produces binaries for both hardware and desktop platforms, so that I can manage releases efficiently.

#### Acceptance Criteria

8.1 THE Heimdal Monorepo SHALL provide a Makefile with targets for building hardware (ARM64 Linux) and desktop (Windows, macOS, Linux) binaries

8.2 THE Makefile SHALL include targets: build-hardware, build-desktop-windows, build-desktop-macos, build-desktop-linux, build-all

8.3 THE Makefile SHALL support cross-compilation for all target platforms from a single development machine

8.4 THE Heimdal Monorepo SHALL produce statically-linked binaries for hardware deployment with no external dependencies

8.5 THE Heimdal Monorepo SHALL produce desktop installers that bundle required dependencies (Npcap for Windows, libpcap for macOS/Linux)

8.6 THE Desktop product SHALL support packaging as a GUI application using Wails or similar framework for native OS integration

### Requirement 9: Shared Cloud Communication

**User Story:** As a platform engineer, I want both hardware and desktop products to communicate with the Asgard cloud platform using the same protocol, so that the backend can handle both device types uniformly.

#### Acceptance Criteria

9.1 THE Heimdal Monorepo SHALL implement shared cloud connector logic in internal/core/cloud

9.2 THE shared cloud connector SHALL support multiple cloud providers including AWS IoT Core and Google Cloud IoT Core

9.3 WHEN either product transmits data to the cloud, THE product SHALL use the shared CloudConnector interface

9.4 THE shared cloud connector SHALL transmit device discovery events, behavioral profiles, and anomaly alerts

9.5 THE shared cloud connector SHALL include device type metadata (hardware vs desktop) in all cloud transmissions

9.6 THE shared cloud connector SHALL handle authentication and encryption consistently across both products

### Requirement 10: Shared Anomaly Detection

**User Story:** As a security analyst, I want both hardware and desktop products to use the same anomaly detection algorithms, so that threat detection is consistent regardless of deployment type.

#### Acceptance Criteria

10.1 THE Heimdal Monorepo SHALL implement anomaly detection algorithms in internal/core/detection

10.2 THE shared anomaly detection module SHALL analyze behavioral profiles to identify unusual communication patterns

10.3 THE shared anomaly detection module SHALL detect anomalies including unexpected destinations, unusual ports, and abnormal traffic volumes

10.4 WHEN an anomaly is detected, THE shared anomaly detection module SHALL generate an alert with severity level and description

10.5 THE shared anomaly detection module SHALL support configurable sensitivity thresholds for different deployment scenarios

### Requirement 11: Desktop Installation and Onboarding

**User Story:** As a desktop user, I want a simple installation process that guides me through setup, so that I can start monitoring my network quickly without technical expertise.

#### Acceptance Criteria

11.1 THE Desktop product SHALL provide a graphical installer for Windows (NSIS or WiX) and macOS (DMG or PKG)

11.2 WHEN the Desktop product first launches, THE Desktop product SHALL display an onboarding wizard that explains required permissions

11.3 THE onboarding wizard SHALL guide users through granting necessary permissions (administrator access, packet capture permissions)

11.4 THE onboarding wizard SHALL verify that required dependencies (Npcap, libpcap) are installed and functional

11.5 THE Desktop product SHALL automatically detect the primary network interface and configure itself without user input

11.6 THE Desktop product SHALL provide a "Quick Start" mode that enables monitoring with default settings in under 60 seconds

### Requirement 12: Desktop System Tray Integration

**User Story:** As a desktop user, I want the application to run in the background with a system tray icon, so that it doesn't clutter my desktop while remaining easily accessible.

#### Acceptance Criteria

12.1 THE Desktop product SHALL display a system tray icon (Windows) or menu bar icon (macOS) when running

12.2 WHEN a user clicks the system tray icon, THE Desktop product SHALL display a menu with options: Open Dashboard, Pause Monitoring, Settings, Quit

12.3 THE system tray icon SHALL change appearance to indicate monitoring status (active, paused, error)

12.4 THE Desktop product SHALL display desktop notifications for important events (new device detected, anomaly detected)

12.5 THE Desktop product SHALL start automatically on system boot if configured by the user

### Requirement 13: Configuration Management

**User Story:** As a system administrator, I want both hardware and desktop products to use consistent configuration formats, so that I can manage settings uniformly across deployments.

#### Acceptance Criteria

13.1 THE Heimdal Monorepo SHALL use JSON format for configuration files

13.2 THE configuration format SHALL include sections for: network settings, cloud connectivity, feature gates, and platform-specific options

13.3 THE Hardware product SHALL read configuration from /etc/heimdal/config.json

13.4 THE Desktop product SHALL read configuration from a user-specific location (AppData on Windows, ~/Library on macOS, ~/.config on Linux)

13.5 THE Heimdal Monorepo SHALL validate configuration on startup and provide clear error messages for invalid settings

13.6 THE Heimdal Monorepo SHALL support configuration updates without requiring application restart where possible

### Requirement 14: Testing Strategy for Shared and Platform-Specific Code

**User Story:** As a quality engineer, I want comprehensive tests for both shared and platform-specific code, so that I can ensure reliability across all deployment scenarios.

#### Acceptance Criteria

14.1 THE Heimdal Monorepo SHALL include unit tests for all shared core logic modules

14.2 THE Heimdal Monorepo SHALL include integration tests that verify interface implementations work correctly

14.3 THE Heimdal Monorepo SHALL include platform-specific tests that run only on their target platforms

14.4 THE Heimdal Monorepo SHALL achieve at least 70% code coverage for shared core modules

14.5 THE Heimdal Monorepo SHALL include mock implementations of platform interfaces for testing core logic in isolation

### Requirement 15: Migration Path from Existing Hardware Codebase

**User Story:** As a developer, I want a clear migration strategy from the existing hardware-focused codebase to the new monorepo structure, so that we can transition without breaking existing deployments.

#### Acceptance Criteria

15.1 THE migration plan SHALL identify all existing modules that will move to internal/core

15.2 THE migration plan SHALL identify all existing modules that will move to internal/platform/linux_embedded

15.3 THE migration plan SHALL maintain backward compatibility with existing configuration files during the transition period

15.4 THE migration plan SHALL provide a mapping document showing old package paths to new package paths

15.5 THE migration plan SHALL ensure existing Ansible playbooks continue to work with the new binary structure
