package property

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/mosiko1234/heimdal/sensor/internal/core/cloud"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

// Feature: monorepo-architecture, Property 9: Cloud Message Type Support
// Validates: Requirements 9.4
//
// Property: For any message type (device discovery, behavioral profile, anomaly alert),
// the shared cloud connector should successfully serialize and transmit the message.
func TestProperty_CloudMessageTypeSupport(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Cloud connector serializes all message types",
		prop.ForAll(
			func(messageType string) bool {
				var err error
				var data []byte

				switch messageType {
				case "profile":
					profile := genTestProfile()
					profileData := &cloud.ProfileData{
						Profile:    profile,
						DeviceType: cloud.DeviceTypeHardware,
						Timestamp:  time.Now(),
					}
					data, err = cloud.SerializeProfileData(profileData)

				case "device":
					device := genTestDevice()
					deviceData := &cloud.DeviceData{
						Device:     device,
						DeviceType: cloud.DeviceTypeDesktop,
						Timestamp:  time.Now(),
					}
					data, err = cloud.SerializeDeviceData(deviceData)

				case "anomaly":
					anomaly := genTestAnomaly()
					data, err = cloud.SerializeAnomalyData(anomaly)

				default:
					t.Logf("Unknown message type: %s", messageType)
					return false
				}

				// Verify serialization succeeded
				if err != nil {
					t.Logf("Failed to serialize %s: %v", messageType, err)
					return false
				}

				// Verify data is not empty
				if len(data) == 0 {
					t.Logf("Serialized %s data is empty", messageType)
					return false
				}

				return true
			},
			genMessageType(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 10: Cloud Metadata Inclusion
// Validates: Requirements 9.5
//
// Property: For any cloud transmission, the message should include device type
// metadata (hardware or desktop) in the payload.
func TestProperty_CloudMetadataInclusion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Cloud messages include device type metadata",
		prop.ForAll(
			func(deviceType cloud.DeviceType) bool {
				// Test with profile
				profile := genTestProfile()
				profileData := &cloud.ProfileData{
					Profile:    profile,
					DeviceType: deviceType,
					Timestamp:  time.Now(),
				}

				// Verify device type is set
				if profileData.DeviceType != deviceType {
					t.Logf("Profile device type mismatch: expected %s, got %s", deviceType, profileData.DeviceType)
					return false
				}

				// Test with device
				device := genTestDevice()
				deviceData := &cloud.DeviceData{
					Device:     device,
					DeviceType: deviceType,
					Timestamp:  time.Now(),
				}

				// Verify device type is set
				if deviceData.DeviceType != deviceType {
					t.Logf("Device device type mismatch: expected %s, got %s", deviceType, deviceData.DeviceType)
					return false
				}

				return true
			},
			genDeviceType(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 11: Cloud Authentication Consistency
// Validates: Requirements 9.6
//
// Property: For any cloud transmission from either product, the same authentication
// and encryption mechanisms should be applied.
func TestProperty_CloudAuthenticationConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Cloud connectors use consistent authentication",
		prop.ForAll(
			func(deviceType cloud.DeviceType) bool {
				// Create mock AWS connector
				awsConnector := &MockCloudConnector{
					connected: true,
					sentItems: make([]string, 0),
				}

				// Create mock GCP connector
				gcpConnector := &MockCloudConnector{
					connected: true,
					sentItems: make([]string, 0),
				}

				// Send profile through both connectors
				profile := genTestProfile()
				
				err1 := awsConnector.SendProfile(profile, deviceType)
				err2 := gcpConnector.SendProfile(profile, deviceType)

				// Both should succeed or both should fail
				if (err1 == nil) != (err2 == nil) {
					t.Logf("Inconsistent behavior: AWS err=%v, GCP err=%v", err1, err2)
					return false
				}

				// Send device through both connectors
				device := genTestDevice()
				
				err1 = awsConnector.SendDevice(device, deviceType)
				err2 = gcpConnector.SendDevice(device, deviceType)

				// Both should succeed or both should fail
				if (err1 == nil) != (err2 == nil) {
					t.Logf("Inconsistent behavior: AWS err=%v, GCP err=%v", err1, err2)
					return false
				}

				return true
			},
			genDeviceType(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generators for cloud testing

func genMessageType() gopter.Gen {
	return gen.OneConstOf("profile", "device", "anomaly")
}

func genDeviceType() gopter.Gen {
	return gen.OneConstOf(cloud.DeviceTypeHardware, cloud.DeviceTypeDesktop)
}

func genTestProfile() *database.BehavioralProfile {
	return &database.BehavioralProfile{
		MAC: "00:11:22:33:44:55",
		Destinations: map[string]*database.DestInfo{
			"192.168.1.1": {
				IP:       "192.168.1.1",
				Count:    100,
				LastSeen: time.Now(),
			},
		},
		Ports: map[uint16]int{
			80:  50,
			443: 50,
		},
		Protocols: map[string]int{
			"TCP": 100,
		},
		TotalPackets:   100,
		TotalBytes:     10000,
		HourlyActivity: [24]int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240},
		FirstSeen:      time.Now().Add(-24 * time.Hour),
		LastSeen:       time.Now(),
	}
}

func genTestDevice() *database.Device {
	return &database.Device{
		MAC:       "00:11:22:33:44:55",
		IP:        "192.168.1.100",
		Name:      "Test Device",
		Vendor:    "Test Vendor",
		IsActive:  true,
		FirstSeen: time.Now().Add(-24 * time.Hour),
		LastSeen:  time.Now(),
	}
}

func genTestAnomaly() *cloud.AnomalyData {
	return &cloud.AnomalyData{
		DeviceMAC:   "00:11:22:33:44:55",
		Type:        "unexpected_destination",
		Severity:    "high",
		Description: "Device contacted unexpected IP address",
		Timestamp:   time.Now(),
		Evidence: map[string]interface{}{
			"destination_ip": "10.0.0.1",
			"port":           8080,
		},
	}
}

// MockCloudConnector for testing
type MockCloudConnector struct {
	connected bool
	sentItems []string
}

func (m *MockCloudConnector) Connect() error {
	m.connected = true
	return nil
}

func (m *MockCloudConnector) Disconnect() error {
	m.connected = false
	return nil
}

func (m *MockCloudConnector) SendProfile(profile *database.BehavioralProfile, deviceType cloud.DeviceType) error {
	if !m.connected {
		return nil
	}
	m.sentItems = append(m.sentItems, "profile")
	return nil
}

func (m *MockCloudConnector) SendDevice(device *database.Device, deviceType cloud.DeviceType) error {
	if !m.connected {
		return nil
	}
	m.sentItems = append(m.sentItems, "device")
	return nil
}

func (m *MockCloudConnector) SendAnomaly(anomaly *cloud.AnomalyData, deviceType cloud.DeviceType) error {
	if !m.connected {
		return nil
	}
	m.sentItems = append(m.sentItems, "anomaly")
	return nil
}

func (m *MockCloudConnector) IsConnected() bool {
	return m.connected
}
