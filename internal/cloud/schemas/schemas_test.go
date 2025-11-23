package schemas

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

func TestDeviceToMessage(t *testing.T) {
	device := &database.Device{
		MAC:          "aa:bb:cc:dd:ee:ff",
		IP:           "192.168.1.100",
		Name:         "Test Device",
		Vendor:       "Apple",
		Manufacturer: "Apple Inc.",
		DeviceType:   "phone",
		Hostname:     "test-iphone",
		Services:     []string{"_airplay._tcp"},
		FirstSeen:    time.Now().Add(-24 * time.Hour),
		LastSeen:     time.Now(),
		IsActive:     true,
	}

	msg := DeviceToMessage(device, "sensor-001", "192.168.1.0/24", "192.168.1.1")

	if msg.MAC != device.MAC {
		t.Errorf("Expected MAC %s, got %s", device.MAC, msg.MAC)
	}
	if msg.DeviceType != device.DeviceType {
		t.Errorf("Expected DeviceType %s, got %s", device.DeviceType, msg.DeviceType)
	}
	if msg.NetworkID != "192.168.1.0/24" {
		t.Errorf("Expected NetworkID 192.168.1.0/24, got %s", msg.NetworkID)
	}
}

func TestProfileToMessage(t *testing.T) {
	profile := &database.BehavioralProfile{
		MAC:          "aa:bb:cc:dd:ee:ff",
		TotalPackets: 1000,
		TotalBytes:   500000,
		FirstSeen:    time.Now().Add(-24 * time.Hour),
		LastSeen:     time.Now(),
		Destinations: map[string]*database.DestInfo{
			"8.8.8.8": {IP: "8.8.8.8", Count: 100, LastSeen: time.Now()},
			"1.1.1.1": {IP: "1.1.1.1", Count: 50, LastSeen: time.Now()},
		},
		Ports: map[uint16]int{
			443: 500,
			80:  300,
			53:  200,
		},
		Protocols: map[string]int{
			"TCP": 800,
			"UDP": 200,
		},
		HourlyActivity: [24]int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240},
		Baseline: &database.ProfileBaseline{
			AvgPacketsPerHour:     100.0,
			StdDevPacketsPerHour:  10.0,
			AvgUniqueDestinations: 5.0,
			ProtocolDistribution: map[string]float64{
				"TCP": 0.8,
				"UDP": 0.2,
			},
			SampleCount: 10,
		},
	}

	msg := ProfileToMessage(profile)

	if msg.MAC != profile.MAC {
		t.Errorf("Expected MAC %s, got %s", profile.MAC, msg.MAC)
	}
	if msg.TotalPackets != profile.TotalPackets {
		t.Errorf("Expected TotalPackets %d, got %d", profile.TotalPackets, msg.TotalPackets)
	}
	if msg.UniqueDestinations != 2 {
		t.Errorf("Expected UniqueDestinations 2, got %d", msg.UniqueDestinations)
	}
	if msg.UniquePorts != 3 {
		t.Errorf("Expected UniquePorts 3, got %d", msg.UniquePorts)
	}
	if len(msg.TopDestinations) != 2 {
		t.Errorf("Expected 2 top destinations, got %d", len(msg.TopDestinations))
	}
	if len(msg.TopPorts) != 3 {
		t.Errorf("Expected 3 top ports, got %d", len(msg.TopPorts))
	}
	if msg.Baseline == nil {
		t.Error("Expected baseline to be included")
	}

	// Verify protocol distribution
	if msg.ProtocolDistribution["TCP"] != 0.8 {
		t.Errorf("Expected TCP distribution 0.8, got %f", msg.ProtocolDistribution["TCP"])
	}
}

func TestWrapMessage(t *testing.T) {
	deviceMsg := &DeviceMessage{
		MAC:        "aa:bb:cc:dd:ee:ff",
		IP:         "192.168.1.100",
		DeviceType: "phone",
	}

	wrapped, err := WrapMessage(MessageTypeDevice, "sensor-001", deviceMsg)
	if err != nil {
		t.Fatalf("WrapMessage failed: %v", err)
	}

	if wrapped.MessageType != MessageTypeDevice {
		t.Errorf("Expected MessageType %s, got %s", MessageTypeDevice, wrapped.MessageType)
	}
	if wrapped.SensorID != "sensor-001" {
		t.Errorf("Expected SensorID sensor-001, got %s", wrapped.SensorID)
	}
	if wrapped.Version != "1.0" {
		t.Errorf("Expected Version 1.0, got %s", wrapped.Version)
	}

	// Verify payload can be unwrapped
	var unwrapped DeviceMessage
	if err := UnwrapMessage(wrapped, &unwrapped); err != nil {
		t.Fatalf("UnwrapMessage failed: %v", err)
	}

	if unwrapped.MAC != deviceMsg.MAC {
		t.Errorf("Expected MAC %s, got %s", deviceMsg.MAC, unwrapped.MAC)
	}
}

func TestSerializeDeserialize(t *testing.T) {
	original := &CloudMessage{
		MessageType: MessageTypeHeartbeat,
		SensorID:    "sensor-001",
		Timestamp:   time.Now(),
		Version:     "1.0",
		Payload:     json.RawMessage(`{"sensor_id":"sensor-001","device_count":10}`),
	}

	// Serialize
	data, err := SerializeMessage(original)
	if err != nil {
		t.Fatalf("SerializeMessage failed: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeMessage(data)
	if err != nil {
		t.Fatalf("DeserializeMessage failed: %v", err)
	}

	if deserialized.MessageType != original.MessageType {
		t.Errorf("Expected MessageType %s, got %s", original.MessageType, deserialized.MessageType)
	}
	if deserialized.SensorID != original.SensorID {
		t.Errorf("Expected SensorID %s, got %s", original.SensorID, deserialized.SensorID)
	}
}

func TestGetTopDestinations(t *testing.T) {
	destinations := map[string]*database.DestInfo{
		"8.8.8.8":  {IP: "8.8.8.8", Count: 100},
		"1.1.1.1":  {IP: "1.1.1.1", Count: 50},
		"10.0.0.1": {IP: "10.0.0.1", Count: 75},
		"10.0.0.2": {IP: "10.0.0.2", Count: 25},
	}

	top := getTopDestinations(destinations, 2)

	if len(top) != 2 {
		t.Errorf("Expected 2 destinations, got %d", len(top))
	}

	// Verify sorted by count
	if top[0].IP != "8.8.8.8" || top[0].Count != 100 {
		t.Errorf("Expected first destination to be 8.8.8.8 with count 100, got %s with count %d", top[0].IP, top[0].Count)
	}
	if top[1].IP != "10.0.0.1" || top[1].Count != 75 {
		t.Errorf("Expected second destination to be 10.0.0.1 with count 75, got %s with count %d", top[1].IP, top[1].Count)
	}
}

func TestGetTopPorts(t *testing.T) {
	ports := map[uint16]int{
		443:  500,
		80:   300,
		53:   200,
		8080: 50,
	}

	top := getTopPorts(ports, 2)

	if len(top) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(top))
	}

	// Verify sorted by count
	if top[0].Port != 443 || top[0].Count != 500 {
		t.Errorf("Expected first port to be 443 with count 500, got %d with count %d", top[0].Port, top[0].Count)
	}

	// Verify percentage calculation
	expectedPercentage := 500.0 / 1050.0
	if top[0].Percentage < expectedPercentage-0.01 || top[0].Percentage > expectedPercentage+0.01 {
		t.Errorf("Expected percentage ~%.3f, got %.3f", expectedPercentage, top[0].Percentage)
	}
}
