# Requirements Document

## Introduction

Heimdal 2.0 is a network security sensor designed for zero-touch provisioning on Raspberry Pi hardware. The system performs automated network discovery, traffic interception via ARP spoofing, behavioral profiling of network devices, and provides a local web dashboard for monitoring. The sensor is implemented as a single Go binary, deployed and managed exclusively via Ansible, with optional cloud connectivity for future integration with the Asgard platform.

## Glossary

- **Heimdal Sensor**: The complete Go-based network monitoring application running on Raspberry Pi
- **Zero-Touch Provisioning**: Deployment model requiring only physical connection (power + Ethernet) with no manual configuration
- **Device Map**: Local database of discovered network devices containing name, IP address, and MAC address
- **Behavioral Profile**: Aggregated traffic patterns for each MAC address showing communication targets and timing
- **ARP Spoofing**: Network technique to intercept traffic by manipulating ARP tables
- **Asgard Platform**: Future cloud-based analytics platform for aggregating sensor data
- **Cloud Connector**: Modular interface for transmitting data to cloud platforms (AWS IoT, Google Cloud)
- **Ansible Target**: The Raspberry Pi device (heimdal-sensor-01 at 10.100.102.131)

## Requirements

### Requirement 1: Zero-Touch Provisioning

**User Story:** As a home user, I want to deploy the Heimdal sensor by simply plugging it into power and Ethernet, so that I can monitor my network without technical configuration.

#### Acceptance Criteria

1.1 WHEN the Heimdal Sensor receives power and network connectivity, THE Heimdal Sensor SHALL automatically detect the local network configuration including gateway IP and subnet mask

1.2 WHEN the Heimdal Sensor completes network detection, THE Heimdal Sensor SHALL begin device discovery without user intervention

1.3 THE Heimdal Sensor SHALL start all monitoring services automatically on system boot

1.4 THE Heimdal Sensor SHALL provide a local web dashboard accessible via the device's IP address within 60 seconds of boot completion

### Requirement 2: Automated Network Discovery

**User Story:** As a network administrator, I want the sensor to automatically discover all devices on my network, so that I have visibility into connected devices without manual inventory.

#### Acceptance Criteria

2.1 THE Heimdal Sensor SHALL continuously scan the local network using ARP and mDNS protocols to discover connected devices

2.2 WHEN a new device is detected on the network, THE Heimdal Sensor SHALL add the device to the Device Map within 30 seconds

2.3 THE Heimdal Sensor SHALL store each discovered device's name, IP address, and MAC address in the Device Map

2.4 WHEN a device disconnects from the network, THE Heimdal Sensor SHALL update the device status in the Device Map within 5 minutes

### Requirement 3: Traffic Interception

**User Story:** As a security analyst, I want the sensor to intercept network traffic from IoT devices, so that I can analyze their communication patterns for anomalies.

#### Acceptance Criteria

3.1 THE Heimdal Sensor SHALL perform ARP spoofing to intercept traffic from all devices on the local network

3.2 WHEN ARP spoofing is active, THE Heimdal Sensor SHALL forward intercepted packets to maintain normal network connectivity

3.3 IF the ARP spoofing component fails, THEN THE Heimdal Sensor SHALL automatically restart the component within 10 seconds

3.4 THE Heimdal Sensor SHALL operate with Linux kernel capabilities (cap_net_raw, cap_net_admin) without requiring root privileges

3.5 THE Heimdal Sensor SHALL enable IP forwarding on the host system to route intercepted traffic

### Requirement 4: Packet Analysis and Profiling

**User Story:** As a security researcher, I want the sensor to build behavioral profiles of devices, so that I can identify normal communication patterns and detect anomalies.

#### Acceptance Criteria

4.1 THE Heimdal Sensor SHALL capture and analyze packets from the network interface using the gopacket library

4.2 WHEN a packet is captured, THE Heimdal Sensor SHALL extract source MAC address, destination IP address, and destination port

4.3 THE Heimdal Sensor SHALL aggregate packet data to create Behavioral Profiles for each unique MAC address

4.4 THE Heimdal Sensor SHALL record communication targets and timing patterns in each Behavioral Profile

4.5 THE Heimdal Sensor SHALL persist Behavioral Profiles to the local database with updates occurring at least every 60 seconds

### Requirement 5: Local Data Persistence

**User Story:** As a system operator, I want all discovered devices and behavioral data stored locally, so that the sensor can operate independently without cloud connectivity.

#### Acceptance Criteria

5.1 THE Heimdal Sensor SHALL use an embedded database to store the Device Map and Behavioral Profiles

5.2 THE Heimdal Sensor SHALL persist data to non-volatile storage to survive system reboots

5.3 WHEN the database becomes unavailable, THE Heimdal Sensor SHALL buffer data in memory for up to 1000 records

5.4 THE Heimdal Sensor SHALL implement database transactions to ensure data consistency

### Requirement 6: Local Web Dashboard

**User Story:** As a home user, I want to view my network devices and their activity through a web browser, so that I can monitor my network without installing additional software.

#### Acceptance Criteria

6.1 THE Heimdal Sensor SHALL serve a web dashboard on port 8080 using the Go net/http package

6.2 THE Heimdal Sensor SHALL display the Device Map showing all discovered devices with their names, IP addresses, and MAC addresses

6.3 THE Heimdal Sensor SHALL provide API endpoints for retrieving device information and behavioral profiles in JSON format

6.4 WHEN a user accesses the dashboard, THE Heimdal Sensor SHALL respond within 500 milliseconds

### Requirement 7: Ansible-Based Deployment

**User Story:** As a DevOps engineer, I want to deploy and update the sensor using Ansible, so that I can manage multiple sensors consistently and reliably.

#### Acceptance Criteria

7.1 THE Ansible playbook SHALL provision the Raspberry Pi target (heimdal-sensor-01 at 10.100.102.131) with all required dependencies

7.2 THE Ansible playbook SHALL enable IP forwarding by setting net.ipv4.ip_forward=1 using the ansible.posix.sysctl module

7.3 THE Ansible playbook SHALL deploy the statically-compiled Go binary to the target device

7.4 THE Ansible playbook SHALL apply kernel capabilities (cap_net_raw, cap_net_admin) to the binary using setcap

7.5 THE Ansible playbook SHALL create and enable a systemd service to run the Heimdal Sensor on boot

7.6 THE Ansible playbook SHALL deploy a configuration file to /etc/heimdal/config.json

### Requirement 8: Modular Cloud Connectivity

**User Story:** As a product manager, I want the sensor to support optional cloud connectivity, so that we can integrate with the Asgard platform in the future without redesigning the core system.

#### Acceptance Criteria

8.1 THE Heimdal Sensor SHALL implement a Cloud Connector interface in Go for transmitting data to cloud platforms

8.2 THE Heimdal Sensor SHALL include stub implementations for AWS IoT Core and Google Cloud IoT Core connectors

8.3 WHERE cloud connectivity is enabled in configuration, THE Heimdal Sensor SHALL transmit Behavioral Profiles to the configured cloud platform

8.4 THE Heimdal Sensor SHALL disable cloud connectivity by default

8.5 WHEN cloud transmission fails, THE Heimdal Sensor SHALL continue local operations without interruption

### Requirement 9: Application Architecture and Concurrency

**User Story:** As a software architect, I want the sensor built with concurrent goroutines and channels, so that it can handle multiple network operations efficiently on resource-constrained hardware.

#### Acceptance Criteria

9.1 THE Heimdal Sensor SHALL implement a main orchestrator that starts all service components as independent goroutines

9.2 THE Heimdal Sensor SHALL use Go channels for inter-component communication

9.3 WHEN a shutdown signal is received, THE Heimdal Sensor SHALL gracefully stop all goroutines within 5 seconds

9.4 THE Heimdal Sensor SHALL implement automatic recovery for failed components without requiring full application restart

9.5 THE Heimdal Sensor SHALL compile to a single statically-linked binary with no external runtime dependencies

### Requirement 10: Resource Efficiency

**User Story:** As a Raspberry Pi user, I want the sensor to use minimal CPU and memory, so that it doesn't impact the device's performance or stability.

#### Acceptance Criteria

10.1 THE Heimdal Sensor SHALL consume less than 200 MB of RAM during normal operation

10.2 THE Heimdal Sensor SHALL use less than 25% CPU on average on a Raspberry Pi 4

10.3 THE Heimdal Sensor SHALL implement rate limiting for network scanning to avoid network congestion

10.4 THE Heimdal Sensor SHALL use buffered channels to prevent goroutine blocking during high traffic periods
