package classifier

import (
	"testing"
)

func TestClassifyDevice(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name          string
		vendor        string
		manufacturer  string
		hostname      string
		services      []string
		expectedType  DeviceType
		minConfidence float64
	}{
		{
			name:          "iPhone by hostname",
			vendor:        "Apple",
			hostname:      "Johns-iPhone",
			expectedType:  DeviceTypePhone,
			minConfidence: 0.5,
		},
		{
			name:          "MacBook by hostname",
			vendor:        "Apple",
			hostname:      "MacBook-Pro",
			expectedType:  DeviceTypeLaptop,
			minConfidence: 0.4,
		},
		{
			name:          "Chromecast by service",
			vendor:        "Google",
			services:      []string{"_googlecast._tcp"},
			expectedType:  DeviceTypeStreaming,
			minConfidence: 0.4,
		},
		{
			name:          "Printer by service",
			vendor:        "HP",
			services:      []string{"_printer._tcp", "_ipp._tcp"},
			expectedType:  DeviceTypePrinter,
			minConfidence: 0.6,
		},
		{
			name:          "Raspberry Pi IoT",
			vendor:        "Raspberry Pi Foundation",
			hostname:      "raspberrypi",
			expectedType:  DeviceTypeIoT,
			minConfidence: 0.5,
		},
		{
			name:          "Samsung phone",
			vendor:        "Samsung Electronics",
			hostname:      "Galaxy-S21",
			expectedType:  DeviceTypePhone,
			minConfidence: 0.4,
		},
		{
			name:          "Cisco router",
			vendor:        "Cisco Systems",
			expectedType:  DeviceTypeRouter,
			minConfidence: 0.1,
		},
		{
			name:          "Unknown device",
			vendor:        "Unknown Vendor Inc",
			hostname:      "device-123",
			expectedType:  DeviceTypeUnknown,
			minConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.ClassifyDevice(tt.vendor, tt.manufacturer, tt.hostname, tt.services)

			if result.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, result.Type)
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %.2f, got %.2f", tt.minConfidence, result.Confidence)
			}

			if result.Category != tt.expectedType.GetCategory() {
				t.Errorf("Expected category %s, got %s", tt.expectedType.GetCategory(), result.Category)
			}

			t.Logf("Classification: type=%s, category=%s, confidence=%.2f, signals=%v",
				result.Type, result.Category, result.Confidence, result.Signals)
		})
	}
}

func TestClassifyByVendorOnly(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		vendor       string
		expectedType DeviceType
	}{
		{"Apple", DeviceTypePhone},
		{"Cisco Systems", DeviceTypeRouter},
		{"HP Inc", DeviceTypePrinter},
		{"Unknown Vendor", DeviceTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			result := c.ClassifyByVendorOnly(tt.vendor)
			if result != tt.expectedType {
				t.Errorf("Expected %s, got %s", tt.expectedType, result)
			}
		})
	}
}

func TestGetConfidenceLevel(t *testing.T) {
	tests := []struct {
		confidence float64
		expected   string
	}{
		{0.95, "high"},
		{0.80, "high"},
		{0.75, "medium"},
		{0.50, "medium"},
		{0.30, "low"},
		{0.10, "low"},
		{0.00, "none"},
	}

	for _, tt := range tests {
		result := GetConfidenceLevel(tt.confidence)
		if result != tt.expected {
			t.Errorf("GetConfidenceLevel(%.2f) = %s, want %s", tt.confidence, result, tt.expected)
		}
	}
}

func TestDeviceTypeCategory(t *testing.T) {
	tests := []struct {
		deviceType DeviceType
		expected   DeviceCategory
	}{
		{DeviceTypePhone, CategoryEndpoint},
		{DeviceTypeComputer, CategoryEndpoint},
		{DeviceTypeRouter, CategoryInfrastructure},
		{DeviceTypePrinter, CategoryPeripheral},
		{DeviceTypeTV, CategoryEntertainment},
		{DeviceTypeIoT, CategoryIoT},
		{DeviceTypeUnknown, CategoryUnknown},
	}

	for _, tt := range tests {
		result := tt.deviceType.GetCategory()
		if result != tt.expected {
			t.Errorf("%s.GetCategory() = %s, want %s", tt.deviceType, result, tt.expected)
		}
	}
}

func BenchmarkClassifyDevice(b *testing.B) {
	c := NewClassifier()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.ClassifyDevice("Apple", "Apple Inc.", "Johns-iPhone", []string{"_airplay._tcp"})
	}
}
