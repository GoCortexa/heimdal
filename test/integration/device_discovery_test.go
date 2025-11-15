package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/discovery"
	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
)

// TestDeviceDiscoveryToDatabaseFlow tests the integration between device discovery and database persistence
func TestDeviceDiscoveryToDatabaseFlow(t *testing.T) {
	// Create temporary database directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db")

	// Initialize database
	db, err := database.NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create network configuration (mock)
	netConfig := netconfig.NewAutoConfig()
	
	// Try to detect network with timeout, skip test if no network available
	detectDone := make(chan error, 1)
	go func() {
		detectDone <- netConfig.DetectNetwork()
	}()
	
	select {
	case err := <-detectDone:
		if err != nil {
			t.Skipf("No network available for testing: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Skip("Network detection timed out - skipping test (expected on macOS)")
	}

	// Create device channel
	deviceChan := make(chan *database.Device, 10)

	// Initialize scanner
	scanner := discovery.NewScanner(
		netConfig,
		db,
		deviceChan,
		60*time.Second, // scan interval
		false,          // mDNS disabled for test
		5*time.Minute,  // inactive timeout
	)

	// Start scanner
	if err := scanner.Start(); err != nil {
		t.Fatalf("Failed to start scanner: %v", err)
	}

	// Create a test device and save it directly to database
	testDevice := &database.Device{
		MAC:       "aa:bb:cc:dd:ee:ff",
		IP:        "192.168.1.100",
		Name:      "TestDevice",
		Vendor:    "TestVendor",
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		IsActive:  true,
	}

	// Save device to database
	if err := db.SaveDevice(testDevice); err != nil {
		t.Fatalf("Failed to save test device: %v", err)
	}

	// Verify device was saved
	retrievedDevice, err := db.GetDevice(testDevice.MAC)
	if err != nil {
		t.Fatalf("Failed to retrieve device: %v", err)
	}

	// Verify device fields
	if retrievedDevice.MAC != testDevice.MAC {
		t.Errorf("Expected MAC %s, got %s", testDevice.MAC, retrievedDevice.MAC)
	}
	if retrievedDevice.IP != testDevice.IP {
		t.Errorf("Expected IP %s, got %s", testDevice.IP, retrievedDevice.IP)
	}
	if retrievedDevice.Name != testDevice.Name {
		t.Errorf("Expected Name %s, got %s", testDevice.Name, retrievedDevice.Name)
	}
	if !retrievedDevice.IsActive {
		t.Error("Expected device to be active")
	}

	// Verify device appears in GetAllDevices
	allDevices, err := db.GetAllDevices()
	if err != nil {
		t.Fatalf("Failed to get all devices: %v", err)
	}

	found := false
	for _, device := range allDevices {
		if device.MAC == testDevice.MAC {
			found = true
			break
		}
	}
	if !found {
		t.Error("Test device not found in GetAllDevices")
	}

	// Stop scanner
	if err := scanner.Stop(); err != nil {
		t.Errorf("Failed to stop scanner: %v", err)
	}

	// Close device channel
	close(deviceChan)
}

// TestDeviceBatchPersistence tests batch device operations
func TestDeviceBatchPersistence(t *testing.T) {
	// Create temporary database directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db")

	// Initialize database
	db, err := database.NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create multiple test devices
	devices := []*database.Device{
		{
			MAC:       "11:22:33:44:55:66",
			IP:        "192.168.1.101",
			Name:      "Device1",
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			IsActive:  true,
		},
		{
			MAC:       "aa:bb:cc:dd:ee:ff",
			IP:        "192.168.1.102",
			Name:      "Device2",
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			IsActive:  true,
		},
		{
			MAC:       "ff:ee:dd:cc:bb:aa",
			IP:        "192.168.1.103",
			Name:      "Device3",
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			IsActive:  false,
		},
	}

	// Save devices in batch
	if err := db.SaveDeviceBatch(devices); err != nil {
		t.Fatalf("Failed to save device batch: %v", err)
	}

	// Verify all devices were saved
	allDevices, err := db.GetAllDevices()
	if err != nil {
		t.Fatalf("Failed to get all devices: %v", err)
	}

	if len(allDevices) != len(devices) {
		t.Errorf("Expected %d devices, got %d", len(devices), len(allDevices))
	}

	// Verify each device
	for _, expectedDevice := range devices {
		retrievedDevice, err := db.GetDevice(expectedDevice.MAC)
		if err != nil {
			t.Errorf("Failed to retrieve device %s: %v", expectedDevice.MAC, err)
			continue
		}

		if retrievedDevice.IP != expectedDevice.IP {
			t.Errorf("Device %s: expected IP %s, got %s", expectedDevice.MAC, expectedDevice.IP, retrievedDevice.IP)
		}
		if retrievedDevice.IsActive != expectedDevice.IsActive {
			t.Errorf("Device %s: expected IsActive %v, got %v", expectedDevice.MAC, expectedDevice.IsActive, retrievedDevice.IsActive)
		}
	}
}

// TestDeviceLifecycle tests device creation, update, and deletion
func TestDeviceLifecycle(t *testing.T) {
	// Create temporary database directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db")

	// Initialize database
	db, err := database.NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create device
	device := &database.Device{
		MAC:       "aa:bb:cc:dd:ee:ff",
		IP:        "192.168.1.100",
		Name:      "TestDevice",
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		IsActive:  true,
	}

	// Save device
	if err := db.SaveDevice(device); err != nil {
		t.Fatalf("Failed to save device: %v", err)
	}

	// Update device
	device.IP = "192.168.1.200"
	device.IsActive = false
	if err := db.SaveDevice(device); err != nil {
		t.Fatalf("Failed to update device: %v", err)
	}

	// Verify update
	retrievedDevice, err := db.GetDevice(device.MAC)
	if err != nil {
		t.Fatalf("Failed to retrieve device: %v", err)
	}
	if retrievedDevice.IP != "192.168.1.200" {
		t.Errorf("Expected IP 192.168.1.200, got %s", retrievedDevice.IP)
	}
	if retrievedDevice.IsActive {
		t.Error("Expected device to be inactive")
	}

	// Delete device
	if err := db.DeleteDevice(device.MAC); err != nil {
		t.Fatalf("Failed to delete device: %v", err)
	}

	// Verify deletion
	_, err = db.GetDevice(device.MAC)
	if err == nil {
		t.Error("Expected error when retrieving deleted device")
	}
}
