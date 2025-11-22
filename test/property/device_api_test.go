// +build property

package property

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/visualizer"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// Feature: monorepo-architecture, Property 4: Device API Response Completeness
// Validates: Requirements 4.2
//
// For any set of discovered devices, the LocalVisualizer API should return JSON
// responses containing all required fields (IP, MAC, device name) for each device.
func TestProperty_DeviceAPIResponseCompleteness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Device API returns complete device information", prop.ForAll(
		func(devices []*database.Device) bool {
			// Create mock storage with test devices
			storage := &MockStorageForDevices{devices: devices}

			// Create visualizer with mock storage
			vis, err := visualizer.NewVisualizer(&visualizer.Config{
				Port:        8080,
				Storage:     storage,
				FeatureGate: nil, // No feature gate for testing
			})
			if err != nil {
				t.Logf("Failed to create visualizer: %v", err)
				return false
			}

			// Create test HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
			w := httptest.NewRecorder()

			// Call the handler directly
			vis.HandleDevices(w, req)

			// Check response status
			if w.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d", w.Code)
				return false
			}

			// Parse response
			var response []visualizer.DeviceResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Logf("Failed to decode response: %v", err)
				return false
			}

			// Verify response has same number of devices
			if len(response) != len(devices) {
				t.Logf("Expected %d devices, got %d", len(devices), len(response))
				return false
			}

			// Verify each device has all required fields
			for i, deviceResp := range response {
				// Find corresponding input device
				var inputDevice *database.Device
				for _, d := range devices {
					if d.MAC == deviceResp.MAC {
						inputDevice = d
						break
					}
				}

				if inputDevice == nil {
					t.Logf("Response device %d not found in input", i)
					return false
				}

				// Check all required fields are present and non-empty
				if deviceResp.MAC == "" {
					t.Logf("Device %d missing MAC address", i)
					return false
				}

				if deviceResp.IP == "" {
					t.Logf("Device %d missing IP address", i)
					return false
				}

				if deviceResp.Name == "" {
					t.Logf("Device %d missing name", i)
					return false
				}

				// Verify fields match input
				if deviceResp.MAC != inputDevice.MAC {
					t.Logf("Device %d MAC mismatch: expected %s, got %s", i, inputDevice.MAC, deviceResp.MAC)
					return false
				}

				if deviceResp.IP != inputDevice.IP {
					t.Logf("Device %d IP mismatch: expected %s, got %s", i, inputDevice.IP, deviceResp.IP)
					return false
				}

				if deviceResp.Name != inputDevice.Name {
					t.Logf("Device %d Name mismatch: expected %s, got %s", i, inputDevice.Name, deviceResp.Name)
					return false
				}
			}

			return true
		},
		genDeviceList(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: monorepo-architecture, Property 6: API Endpoint JSON Validity
// Validates: Requirements 4.5
//
// For any API endpoint in the LocalVisualizer, responses should be valid JSON
// and include expected fields for the endpoint type.
func TestProperty_APIEndpointJSONValidity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("All API endpoints return valid JSON", prop.ForAll(
		func(device *database.Device, profile *database.BehavioralProfile) bool {
			// Create mock storage with test data
			storage := &MockStorageForDevices{
				devices:  []*database.Device{device},
				profiles: []*database.BehavioralProfile{profile},
			}

			// Create visualizer with mock storage
			vis, err := visualizer.NewVisualizer(&visualizer.Config{
				Port:        8080,
				Storage:     storage,
				FeatureGate: nil,
			})
			if err != nil {
				t.Logf("Failed to create visualizer: %v", err)
				return false
			}

			// Test /api/v1/devices endpoint
			req1 := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
			w1 := httptest.NewRecorder()
			vis.HandleDevices(w1, req1)

			if w1.Code != http.StatusOK {
				t.Logf("Devices endpoint returned status %d", w1.Code)
				return false
			}

			// Verify valid JSON
			var devicesResp []visualizer.DeviceResponse
			if err := json.NewDecoder(w1.Body).Decode(&devicesResp); err != nil {
				t.Logf("Devices endpoint returned invalid JSON: %v", err)
				return false
			}

			// Test /api/v1/devices/:mac endpoint
			req2 := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+device.MAC, nil)
			w2 := httptest.NewRecorder()
			vis.HandleDeviceByMAC(w2, req2)

			if w2.Code != http.StatusOK {
				t.Logf("Device by MAC endpoint returned status %d", w2.Code)
				return false
			}

			// Verify valid JSON
			var deviceResp visualizer.DeviceResponse
			if err := json.NewDecoder(w2.Body).Decode(&deviceResp); err != nil {
				t.Logf("Device by MAC endpoint returned invalid JSON: %v", err)
				return false
			}

			// Verify expected fields are present
			if deviceResp.MAC == "" || deviceResp.IP == "" || deviceResp.Name == "" {
				t.Logf("Device response missing required fields")
				return false
			}

			// Test /api/v1/profiles/:mac endpoint
			req3 := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+profile.MAC, nil)
			w3 := httptest.NewRecorder()
			vis.HandleProfileByMAC(w3, req3)

			if w3.Code != http.StatusOK {
				t.Logf("Profile endpoint returned status %d", w3.Code)
				return false
			}

			// Verify valid JSON
			var profileResp visualizer.ProfileResponse
			if err := json.NewDecoder(w3.Body).Decode(&profileResp); err != nil {
				t.Logf("Profile endpoint returned invalid JSON: %v", err)
				return false
			}

			// Verify expected fields are present
			if profileResp.MAC == "" {
				t.Logf("Profile response missing MAC field")
				return false
			}

			if profileResp.Destinations == nil {
				t.Logf("Profile response missing Destinations field")
				return false
			}

			if profileResp.Ports == nil {
				t.Logf("Profile response missing Ports field")
				return false
			}

			if profileResp.Protocols == nil {
				t.Logf("Profile response missing Protocols field")
				return false
			}

			// Test /api/v1/tier endpoint
			req4 := httptest.NewRequest(http.MethodGet, "/api/v1/tier", nil)
			w4 := httptest.NewRecorder()
			vis.HandleTierInfo(w4, req4)

			if w4.Code != http.StatusOK {
				t.Logf("Tier endpoint returned status %d", w4.Code)
				return false
			}

			// Verify valid JSON
			var tierResp visualizer.TierInfoResponse
			if err := json.NewDecoder(w4.Body).Decode(&tierResp); err != nil {
				t.Logf("Tier endpoint returned invalid JSON: %v", err)
				return false
			}

			// Verify expected fields are present
			if tierResp.Tier == "" {
				t.Logf("Tier response missing Tier field")
				return false
			}

			if tierResp.Features == nil {
				t.Logf("Tier response missing Features field")
				return false
			}

			return true
		},
		genDevice(),
		genBehavioralProfile(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genDevice generates a random Device for testing
func genDevice() gopter.Gen {
	return gopter.CombineGens(
		genMAC(),
		genIP(),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Bool(),
	).Map(func(values []interface{}) *database.Device {
		mac := values[0].(net.HardwareAddr)
		ip := values[1].(net.IP)
		name := values[2].(string)
		vendor := values[3].(string)
		isActive := values[4].(bool)

		// Ensure non-empty strings
		if name == "" {
			name = "TestDevice"
		}
		if vendor == "" {
			vendor = "TestVendor"
		}

		return &database.Device{
			MAC:       mac.String(),
			IP:        ip.String(),
			Name:      name,
			Vendor:    vendor,
			FirstSeen: time.Now().Add(-24 * time.Hour),
			LastSeen:  time.Now(),
			IsActive:  isActive,
		}
	})
}

// genDeviceList generates a list of random devices
func genDeviceList() gopter.Gen {
	return gen.SliceOfN(5, genDevice()).SuchThat(func(devices []*database.Device) bool {
		// Ensure all devices have unique MAC addresses
		seen := make(map[string]bool)
		for _, d := range devices {
			if seen[d.MAC] {
				return false
			}
			seen[d.MAC] = true
		}
		return true
	})
}

// genBehavioralProfile generates a random BehavioralProfile for testing
func genBehavioralProfile() gopter.Gen {
	return gopter.CombineGens(
		genMAC(),
		gen.IntRange(1, 10),
		gen.IntRange(1, 10),
	).Map(func(values []interface{}) *database.BehavioralProfile {
		mac := values[0].(net.HardwareAddr)
		numDests := values[1].(int)
		numPorts := values[2].(int)

		// Generate destinations
		destinations := make(map[string]*database.DestInfo)
		for i := 0; i < numDests; i++ {
			ip := fmt.Sprintf("192.168.1.%d", i+1)
			destinations[ip] = &database.DestInfo{
				IP:       ip,
				Count:    int64(i + 1),
				LastSeen: time.Now(),
			}
		}

		// Generate ports
		ports := make(map[uint16]int)
		for i := 0; i < numPorts; i++ {
			ports[uint16(80+i)] = i + 1
		}

		// Generate protocols
		protocols := map[string]int{
			"TCP":  10,
			"UDP":  5,
			"ICMP": 2,
		}

		return &database.BehavioralProfile{
			MAC:            mac.String(),
			Destinations:   destinations,
			Ports:          ports,
			Protocols:      protocols,
			TotalPackets:   100,
			TotalBytes:     10000,
			FirstSeen:      time.Now().Add(-24 * time.Hour),
			LastSeen:       time.Now(),
			HourlyActivity: [24]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
		}
	})
}

// MockStorageForDevices is a mock storage provider for testing
type MockStorageForDevices struct {
	devices  []*database.Device
	profiles []*database.BehavioralProfile
}

func (m *MockStorageForDevices) Open(path string, options *platform.StorageOptions) error {
	return nil
}

func (m *MockStorageForDevices) Close() error {
	return nil
}

func (m *MockStorageForDevices) Get(key string) ([]byte, error) {
	// Handle device keys
	if len(key) > 7 && key[:7] == "device:" {
		mac := key[7:]
		for _, device := range m.devices {
			if device.MAC == mac {
				return json.Marshal(device)
			}
		}
		return nil, fmt.Errorf("device not found")
	}

	// Handle profile keys
	if len(key) > 8 && key[:8] == "profile:" {
		mac := key[8:]
		for _, profile := range m.profiles {
			if profile.MAC == mac {
				return json.Marshal(profile)
			}
		}
		return nil, fmt.Errorf("profile not found")
	}

	return nil, fmt.Errorf("key not found")
}

func (m *MockStorageForDevices) Set(key string, value []byte) error {
	return nil
}

func (m *MockStorageForDevices) Delete(key string) error {
	return nil
}

func (m *MockStorageForDevices) List(prefix string) ([]string, error) {
	keys := []string{}

	if prefix == "device:" {
		for _, device := range m.devices {
			keys = append(keys, "device:"+device.MAC)
		}
	}

	if prefix == "profile:" {
		for _, profile := range m.profiles {
			keys = append(keys, "profile:"+profile.MAC)
		}
	}

	return keys, nil
}

func (m *MockStorageForDevices) Batch(ops []platform.BatchOp) error {
	return nil
}
