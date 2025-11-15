package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/analyzer"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/profiler"
)

// TestPacketAnalyzerToProfilerToDatabaseFlow tests the integration between packet analyzer, profiler, and database
func TestPacketAnalyzerToProfilerToDatabaseFlow(t *testing.T) {
	// Create temporary database directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db")

	// Initialize database
	db, err := database.NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create packet channel
	packetChan := make(chan analyzer.PacketInfo, 100)

	// Initialize profiler with short persist interval for testing
	prof, err := profiler.NewProfiler(db, packetChan, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to initialize profiler: %v", err)
	}

	// Start profiler
	if err := prof.Start(); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Send test packets
	testMAC := "aa:bb:cc:dd:ee:ff"
	testPackets := []analyzer.PacketInfo{
		{
			Timestamp: time.Now(),
			SrcMAC:    testMAC,
			DstIP:     "8.8.8.8",
			DstPort:   443,
			Protocol:  "TCP",
			Size:      1500,
		},
		{
			Timestamp: time.Now(),
			SrcMAC:    testMAC,
			DstIP:     "8.8.8.8",
			DstPort:   443,
			Protocol:  "TCP",
			Size:      1200,
		},
		{
			Timestamp: time.Now(),
			SrcMAC:    testMAC,
			DstIP:     "1.1.1.1",
			DstPort:   80,
			Protocol:  "TCP",
			Size:      800,
		},
	}

	// Send packets to profiler
	for _, packet := range testPackets {
		packetChan <- packet
	}

	// Wait for profiler to process packets
	time.Sleep(500 * time.Millisecond)

	// Verify profile was created in memory
	profile, err := prof.GetProfile(testMAC)
	if err != nil {
		t.Fatalf("Failed to get profile from profiler: %v", err)
	}

	// Verify profile data
	if profile.MAC != testMAC {
		t.Errorf("Expected MAC %s, got %s", testMAC, profile.MAC)
	}
	if profile.TotalPackets != int64(len(testPackets)) {
		t.Errorf("Expected %d packets, got %d", len(testPackets), profile.TotalPackets)
	}

	expectedBytes := int64(1500 + 1200 + 800)
	if profile.TotalBytes != expectedBytes {
		t.Errorf("Expected %d bytes, got %d", expectedBytes, profile.TotalBytes)
	}

	// Verify destinations
	if len(profile.Destinations) != 2 {
		t.Errorf("Expected 2 destinations, got %d", len(profile.Destinations))
	}
	if destInfo, exists := profile.Destinations["8.8.8.8"]; exists {
		if destInfo.Count != 2 {
			t.Errorf("Expected 2 packets to 8.8.8.8, got %d", destInfo.Count)
		}
	} else {
		t.Error("Expected destination 8.8.8.8 not found")
	}

	// Verify ports
	if profile.Ports[443] != 2 {
		t.Errorf("Expected 2 packets to port 443, got %d", profile.Ports[443])
	}
	if profile.Ports[80] != 1 {
		t.Errorf("Expected 1 packet to port 80, got %d", profile.Ports[80])
	}

	// Verify protocols
	if profile.Protocols["TCP"] != 3 {
		t.Errorf("Expected 3 TCP packets, got %d", profile.Protocols["TCP"])
	}

	// Wait for persistence interval to trigger
	time.Sleep(3 * time.Second)

	// Verify profile was persisted to database
	dbProfile, err := db.GetProfile(testMAC)
	if err != nil {
		t.Fatalf("Failed to get profile from database: %v", err)
	}

	// Verify persisted profile data
	if dbProfile.MAC != testMAC {
		t.Errorf("Expected MAC %s, got %s", testMAC, dbProfile.MAC)
	}
	if dbProfile.TotalPackets != int64(len(testPackets)) {
		t.Errorf("Expected %d packets in DB, got %d", len(testPackets), dbProfile.TotalPackets)
	}

	// Stop profiler
	if err := prof.Stop(); err != nil {
		t.Errorf("Failed to stop profiler: %v", err)
	}

	// Close packet channel
	close(packetChan)
}

// TestProfileBatchPersistence tests batch profile operations
func TestProfileBatchPersistence(t *testing.T) {
	// Create temporary database directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db")

	// Initialize database
	db, err := database.NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create multiple test profiles
	profiles := []*database.BehavioralProfile{
		{
			MAC: "11:22:33:44:55:66",
			Destinations: map[string]*database.DestInfo{
				"8.8.8.8": {IP: "8.8.8.8", Count: 10, LastSeen: time.Now()},
			},
			Ports:        map[uint16]int{443: 10},
			Protocols:    map[string]int{"TCP": 10},
			TotalPackets: 10,
			TotalBytes:   15000,
			FirstSeen:    time.Now(),
			LastSeen:     time.Now(),
		},
		{
			MAC: "aa:bb:cc:dd:ee:ff",
			Destinations: map[string]*database.DestInfo{
				"1.1.1.1": {IP: "1.1.1.1", Count: 5, LastSeen: time.Now()},
			},
			Ports:        map[uint16]int{80: 5},
			Protocols:    map[string]int{"TCP": 5},
			TotalPackets: 5,
			TotalBytes:   4000,
			FirstSeen:    time.Now(),
			LastSeen:     time.Now(),
		},
	}

	// Save profiles in batch
	if err := db.SaveProfileBatch(profiles); err != nil {
		t.Fatalf("Failed to save profile batch: %v", err)
	}

	// Verify all profiles were saved
	allProfiles, err := db.GetAllProfiles()
	if err != nil {
		t.Fatalf("Failed to get all profiles: %v", err)
	}

	if len(allProfiles) != len(profiles) {
		t.Errorf("Expected %d profiles, got %d", len(profiles), len(allProfiles))
	}

	// Verify each profile
	for _, expectedProfile := range profiles {
		retrievedProfile, err := db.GetProfile(expectedProfile.MAC)
		if err != nil {
			t.Errorf("Failed to retrieve profile %s: %v", expectedProfile.MAC, err)
			continue
		}

		if retrievedProfile.TotalPackets != expectedProfile.TotalPackets {
			t.Errorf("Profile %s: expected %d packets, got %d",
				expectedProfile.MAC, expectedProfile.TotalPackets, retrievedProfile.TotalPackets)
		}
		if retrievedProfile.TotalBytes != expectedProfile.TotalBytes {
			t.Errorf("Profile %s: expected %d bytes, got %d",
				expectedProfile.MAC, expectedProfile.TotalBytes, retrievedProfile.TotalBytes)
		}
	}
}

// TestProfileAggregation tests profile aggregation logic
func TestProfileAggregation(t *testing.T) {
	// Create temporary database directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db")

	// Initialize database
	db, err := database.NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create packet channel
	packetChan := make(chan analyzer.PacketInfo, 100)

	// Initialize profiler
	prof, err := profiler.NewProfiler(db, packetChan, 60*time.Second)
	if err != nil {
		t.Fatalf("Failed to initialize profiler: %v", err)
	}

	// Start profiler
	if err := prof.Start(); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Send packets from multiple devices
	testMAC1 := "aa:bb:cc:dd:ee:ff"
	testMAC2 := "11:22:33:44:55:66"

	packets := []analyzer.PacketInfo{
		{Timestamp: time.Now(), SrcMAC: testMAC1, DstIP: "8.8.8.8", DstPort: 443, Protocol: "TCP", Size: 1000},
		{Timestamp: time.Now(), SrcMAC: testMAC1, DstIP: "8.8.8.8", DstPort: 443, Protocol: "TCP", Size: 1000},
		{Timestamp: time.Now(), SrcMAC: testMAC2, DstIP: "1.1.1.1", DstPort: 80, Protocol: "TCP", Size: 500},
	}

	for _, packet := range packets {
		packetChan <- packet
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify profile count
	if prof.GetProfileCount() != 2 {
		t.Errorf("Expected 2 profiles, got %d", prof.GetProfileCount())
	}

	// Verify first profile
	profile1, err := prof.GetProfile(testMAC1)
	if err != nil {
		t.Fatalf("Failed to get profile for %s: %v", testMAC1, err)
	}
	if profile1.TotalPackets != 2 {
		t.Errorf("Expected 2 packets for %s, got %d", testMAC1, profile1.TotalPackets)
	}

	// Verify second profile
	profile2, err := prof.GetProfile(testMAC2)
	if err != nil {
		t.Fatalf("Failed to get profile for %s: %v", testMAC2, err)
	}
	if profile2.TotalPackets != 1 {
		t.Errorf("Expected 1 packet for %s, got %d", testMAC2, profile2.TotalPackets)
	}

	// Stop profiler
	if err := prof.Stop(); err != nil {
		t.Errorf("Failed to stop profiler: %v", err)
	}

	close(packetChan)
}

// TestHourlyActivityTracking tests the hourly activity tracking feature
func TestHourlyActivityTracking(t *testing.T) {
	// Create temporary database directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_db")

	// Initialize database
	db, err := database.NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create packet channel
	packetChan := make(chan analyzer.PacketInfo, 100)

	// Initialize profiler
	prof, err := profiler.NewProfiler(db, packetChan, 60*time.Second)
	if err != nil {
		t.Fatalf("Failed to initialize profiler: %v", err)
	}

	// Start profiler
	if err := prof.Start(); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Send packets with current timestamp
	testMAC := "aa:bb:cc:dd:ee:ff"
	now := time.Now()
	currentHour := now.Hour()

	for i := 0; i < 5; i++ {
		packetChan <- analyzer.PacketInfo{
			Timestamp: now,
			SrcMAC:    testMAC,
			DstIP:     "8.8.8.8",
			DstPort:   443,
			Protocol:  "TCP",
			Size:      1000,
		}
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify hourly activity
	profile, err := prof.GetProfile(testMAC)
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}

	if profile.HourlyActivity[currentHour] != 5 {
		t.Errorf("Expected 5 packets in hour %d, got %d", currentHour, profile.HourlyActivity[currentHour])
	}

	// Stop profiler
	if err := prof.Stop(); err != nil {
		t.Errorf("Failed to stop profiler: %v", err)
	}

	close(packetChan)
}
