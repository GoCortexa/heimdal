// +build darwin

package desktop_macos

import (
	"testing"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// TestPacketCaptureInterface verifies MacOSPacketCapture implements PacketCaptureProvider
func TestPacketCaptureInterface(t *testing.T) {
	var _ platform.PacketCaptureProvider = (*MacOSPacketCapture)(nil)
	
	capture := NewMacOSPacketCapture()
	if capture == nil {
		t.Fatal("NewMacOSPacketCapture returned nil")
	}
}

// TestSystemIntegratorInterface verifies MacOSSystemIntegrator implements SystemIntegrator
func TestSystemIntegratorInterface(t *testing.T) {
	var _ platform.SystemIntegrator = (*MacOSSystemIntegrator)(nil)
	
	integrator := NewMacOSSystemIntegrator()
	if integrator == nil {
		t.Fatal("NewMacOSSystemIntegrator returned nil")
	}
	
	// Test daemon variant
	daemon := NewMacOSSystemIntegratorDaemon()
	if daemon == nil {
		t.Fatal("NewMacOSSystemIntegratorDaemon returned nil")
	}
	
	if daemon.IsUserAgent() {
		t.Error("Daemon should not be a user agent")
	}
	
	if !integrator.IsUserAgent() {
		t.Error("Default integrator should be a user agent")
	}
}

// TestStorageInterface verifies MacOSStorage implements StorageProvider
func TestStorageInterface(t *testing.T) {
	var _ platform.StorageProvider = (*MacOSStorage)(nil)
	
	storage := NewMacOSStorage()
	if storage == nil {
		t.Fatal("NewMacOSStorage returned nil")
	}
}

// TestGetDefaultStoragePath verifies the default storage path is correct
func TestGetDefaultStoragePath(t *testing.T) {
	path, err := GetDefaultStoragePath()
	if err != nil {
		t.Fatalf("GetDefaultStoragePath failed: %v", err)
	}
	
	if path == "" {
		t.Error("GetDefaultStoragePath returned empty path")
	}
	
	// Should contain Library/Application Support
	if !contains(path, "Library") || !contains(path, "Application Support") {
		t.Errorf("Path %s doesn't contain expected macOS directories", path)
	}
}

// TestLibpcapAvailability tests libpcap availability check
func TestLibpcapAvailability(t *testing.T) {
	// This should always be true on macOS as libpcap is built-in
	available := IsLibpcapAvailable()
	if !available {
		t.Log("Warning: libpcap not available - this is unexpected on macOS")
	}
}

// TestPermissionGuidance verifies guidance strings are not empty
func TestPermissionGuidance(t *testing.T) {
	guidance := GetLibpcapPermissionGuidance()
	if guidance == "" {
		t.Error("GetLibpcapPermissionGuidance returned empty string")
	}
	
	// Should mention System Preferences
	if !contains(guidance, "System Preferences") && !contains(guidance, "sudo") {
		t.Error("Guidance should mention System Preferences or sudo")
	}
}

// TestListInterfaces verifies we can list network interfaces
func TestListInterfaces(t *testing.T) {
	interfaces, err := ListInterfaces()
	if err != nil {
		t.Logf("ListInterfaces failed: %v (may require permissions)", err)
		return
	}
	
	if len(interfaces) == 0 {
		t.Log("Warning: No network interfaces found")
	}
	
	for _, iface := range interfaces {
		t.Logf("Found interface: %s", iface.Name)
	}
}

// TestSetServiceName verifies service name can be set
func TestSetServiceName(t *testing.T) {
	integrator := NewMacOSSystemIntegrator()
	
	serviceName := "com.test.service"
	integrator.SetServiceName(serviceName)
	
	if integrator.GetServiceName() != serviceName {
		t.Errorf("Expected service name %s, got %s", serviceName, integrator.GetServiceName())
	}
	
	plistPath := integrator.GetPlistPath()
	if plistPath == "" {
		t.Error("Plist path should not be empty after setting service name")
	}
	
	if !contains(plistPath, serviceName) {
		t.Errorf("Plist path %s should contain service name %s", plistPath, serviceName)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
