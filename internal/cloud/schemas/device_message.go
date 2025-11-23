// Package schemas defines the data structures for cloud communication with Asgard platform.
// These schemas ensure consistent data format for device telemetry and behavioral profiles.
package schemas

import (
	"encoding/json"
	"time"
)

// MessageType identifies the type of cloud message
type MessageType string

const (
	MessageTypeDevice    MessageType = "device"
	MessageTypeProfile   MessageType = "profile"
	MessageTypeAnomaly   MessageType = "anomaly"
	MessageTypeHeartbeat MessageType = "heartbeat"
	MessageTypeTelemetry MessageType = "telemetry"
)

// CloudMessage is the envelope for all cloud communications
type CloudMessage struct {
	MessageType MessageType     `json:"message_type"`
	SensorID    string          `json:"sensor_id"` // Unique sensor identifier
	Timestamp   time.Time       `json:"timestamp"`
	Version     string          `json:"version"` // Schema version
	Payload     json.RawMessage `json:"payload"`
}

// DeviceMessage contains device discovery information
type DeviceMessage struct {
	MAC          string    `json:"mac"`
	IP           string    `json:"ip"`
	Name         string    `json:"name,omitempty"`
	Vendor       string    `json:"vendor,omitempty"`
	Manufacturer string    `json:"manufacturer,omitempty"`
	DeviceType   string    `json:"device_type,omitempty"`
	Hostname     string    `json:"hostname,omitempty"`
	Services     []string  `json:"services,omitempty"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	IsActive     bool      `json:"is_active"`

	// Network context
	NetworkID string `json:"network_id,omitempty"` // Subnet or network identifier
	Gateway   string `json:"gateway,omitempty"`    // Gateway IP
}

// ProfileMessage contains behavioral profile data
type ProfileMessage struct {
	MAC          string    `json:"mac"`
	TotalPackets int64     `json:"total_packets"`
	TotalBytes   int64     `json:"total_bytes"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`

	// Aggregated metrics
	UniqueDestinations   int                `json:"unique_destinations"`
	UniquePorts          int                `json:"unique_ports"`
	ProtocolDistribution map[string]float64 `json:"protocol_distribution"` // Protocol -> percentage

	// Top destinations (limited to top 20 for bandwidth)
	TopDestinations []DestinationSummary `json:"top_destinations"`

	// Top ports (limited to top 10)
	TopPorts []PortSummary `json:"top_ports"`

	// Hourly activity pattern
	HourlyActivity [24]int `json:"hourly_activity"`

	// Baseline metrics (if available)
	Baseline *BaselineMetrics `json:"baseline,omitempty"`
}

// DestinationSummary summarizes communication with a destination
type DestinationSummary struct {
	IP       string    `json:"ip"`
	Count    int64     `json:"count"`
	LastSeen time.Time `json:"last_seen"`
}

// PortSummary summarizes port usage
type PortSummary struct {
	Port       uint16  `json:"port"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// BaselineMetrics contains statistical baselines for anomaly detection
type BaselineMetrics struct {
	AvgPacketsPerHour     float64            `json:"avg_packets_per_hour"`
	StdDevPacketsPerHour  float64            `json:"stddev_packets_per_hour"`
	AvgPacketsPerDay      float64            `json:"avg_packets_per_day"`
	StdDevPacketsPerDay   float64            `json:"stddev_packets_per_day"`
	AvgUniqueDestinations float64            `json:"avg_unique_destinations"`
	StdDevDestinations    float64            `json:"stddev_destinations"`
	ProtocolDistribution  map[string]float64 `json:"protocol_distribution"`
	LastCalculated        time.Time          `json:"last_calculated"`
	SampleCount           int                `json:"sample_count"`
}

// AnomalyMessage contains detected anomaly information
type AnomalyMessage struct {
	DeviceMAC   string                 `json:"device_mac"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Evidence    map[string]interface{} `json:"evidence"`

	// Device context
	DeviceType   string `json:"device_type,omitempty"`
	DeviceVendor string `json:"device_vendor,omitempty"`
}

// HeartbeatMessage contains sensor health information
type HeartbeatMessage struct {
	SensorID      string    `json:"sensor_id"`
	Timestamp     time.Time `json:"timestamp"`
	Uptime        int64     `json:"uptime_seconds"`
	DeviceCount   int       `json:"device_count"`
	ActiveDevices int       `json:"active_devices"`
	ProfileCount  int       `json:"profile_count"`

	// Component health
	ComponentStatus map[string]bool `json:"component_status"`
}

// TelemetryMessage contains optional diagnostics data
type TelemetryMessage struct {
	SensorID  string    `json:"sensor_id"`
	Timestamp time.Time `json:"timestamp"`
	Platform  string    `json:"platform"` // "desktop" or "hardware"
	OS        string    `json:"os"`       // "darwin", "linux", "windows"
	Version   string    `json:"version"`

	// Performance metrics
	CPUUsage    float64 `json:"cpu_usage,omitempty"`
	MemoryUsage int64   `json:"memory_usage_bytes,omitempty"`

	// Discovery metrics
	ARPScanSuccess  bool `json:"arp_scan_success"`
	MDNSScanSuccess bool `json:"mdns_scan_success"`

	// Custom metrics
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}
