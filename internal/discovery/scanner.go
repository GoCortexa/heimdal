// Package discovery provides network device discovery functionality using ARP and mDNS protocols.
//
// The Scanner component continuously scans the local network to discover connected devices,
// tracking their IP addresses, MAC addresses, and names. It uses two complementary methods:
//
// ARP Scanning (arp.go):
//   - Sends ARP requests to all IPs in the subnet CIDR range
//   - Parses ARP responses to extract IP and MAC addresses
//   - Runs periodically (default: every 60 seconds)
//   - Uses gopacket library for packet crafting and parsing
//
// mDNS Discovery (mdns.go):
//   - Passive listener for mDNS service announcements
//   - Active mDNS queries at longer intervals (default: every 5 minutes)
//   - Extracts device names from mDNS responses
//   - Uses hashicorp/mdns library for DNS-SD
//
// Device Lifecycle:
//   - Tracks LastSeen timestamp for each device
//   - Marks devices inactive if not seen within timeout period (default: 5 minutes)
//   - Updates database immediately on discovery or status change
//   - Sends discovered devices to deviceChan for traffic interception
//
// The Scanner implements the Component interface for lifecycle management by the orchestrator.
package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
)

// Scanner manages device discovery operations
type Scanner struct {
	netConfig       *netconfig.AutoConfig
	db              *database.DatabaseManager
	deviceChan      chan<- *database.Device
	scanInterval    time.Duration
	mdnsEnabled     bool
	inactiveTimeout time.Duration
	logger          *logger.Logger
	
	// Internal state
	devices         map[string]*database.Device // MAC -> Device
	devicesMu       sync.RWMutex
	
	// Lifecycle management
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
	runningMu       sync.Mutex
}

// NewScanner creates a new Scanner instance with the provided dependencies
func NewScanner(netConfig *netconfig.AutoConfig, db *database.DatabaseManager, deviceChan chan<- *database.Device, scanInterval time.Duration, mdnsEnabled bool, inactiveTimeout time.Duration) *Scanner {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Scanner{
		netConfig:       netConfig,
		db:              db,
		deviceChan:      deviceChan,
		scanInterval:    scanInterval,
		mdnsEnabled:     mdnsEnabled,
		inactiveTimeout: inactiveTimeout,
		logger:          logger.NewComponentLogger("Scanner"),
		devices:         make(map[string]*database.Device),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start launches the discovery goroutines
func (s *Scanner) Start() error {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return fmt.Errorf("scanner already running")
	}
	s.running = true
	s.runningMu.Unlock()

	s.logger.Info("Starting device discovery scanner...")

	// Load existing devices from database
	if err := s.loadExistingDevices(); err != nil {
		s.logger.Warn("Failed to load existing devices: %v", err)
	}

	// Start ARP scanning goroutine
	s.wg.Add(1)
	go s.arpScanLoop()

	// Start mDNS scanning goroutine if enabled
	if s.mdnsEnabled {
		s.wg.Add(1)
		go s.mdnsScanLoop()
		s.logger.Info("mDNS discovery enabled")
	} else {
		s.logger.Info("mDNS discovery disabled")
	}

	// Start device lifecycle management goroutine
	s.wg.Add(1)
	go s.lifecycleLoop()

	s.logger.Info("Device discovery scanner started (scan interval: %v)", s.scanInterval)
	return nil
}

// Stop gracefully shuts down the scanner
func (s *Scanner) Stop() error {
	s.runningMu.Lock()
	if !s.running {
		s.runningMu.Unlock()
		return fmt.Errorf("scanner not running")
	}
	s.running = false
	s.runningMu.Unlock()

	// Cancel context to signal all goroutines to stop
	s.cancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Device discovery scanner stopped gracefully")
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("scanner shutdown timeout")
	}
}

// Name returns the component name
func (s *Scanner) Name() string {
	return "DeviceDiscoveryScanner"
}

// loadExistingDevices loads devices from the database into memory
func (s *Scanner) loadExistingDevices() error {
	devices, err := s.db.GetAllDevices()
	if err != nil {
		return fmt.Errorf("failed to load devices from database: %w", err)
	}

	s.devicesMu.Lock()
	defer s.devicesMu.Unlock()

	for _, device := range devices {
		s.devices[device.MAC] = device
	}

	s.logger.Info("Loaded %d existing devices from database", len(devices))
	return nil
}

// updateDevice updates or creates a device entry
func (s *Scanner) updateDevice(mac, ip, name, vendor string) {
	now := time.Now()

	s.devicesMu.Lock()
	device, exists := s.devices[mac]
	
	if exists {
		// Update existing device
		device.IP = ip
		device.LastSeen = now
		device.IsActive = true
		
		// Update name if we got a better one (non-empty)
		if name != "" && device.Name == "" {
			device.Name = name
		}
		
		// Update vendor if we got one
		if vendor != "" && device.Vendor == "" {
			device.Vendor = vendor
		}
	} else {
		// Create new device
		device = &database.Device{
			MAC:       mac,
			IP:        ip,
			Name:      name,
			Vendor:    vendor,
			FirstSeen: now,
			LastSeen:  now,
			IsActive:  true,
		}
		s.devices[mac] = device
	}
	
	// Make a copy for sending
	deviceCopy := *device
	s.devicesMu.Unlock()

	// Save to database immediately
	if err := s.db.SaveDevice(&deviceCopy); err != nil {
		s.logger.Error("Error saving device %s to database: %v", mac, err)
	} else {
		s.logger.Debug("Device %s saved to database", mac)
	}

	// Send to device channel (non-blocking)
	select {
	case s.deviceChan <- &deviceCopy:
		s.logger.Debug("Device %s sent to channel", mac)
	default:
		s.logger.Warn("Device channel full, dropping device update for %s", mac)
	}
}

// lifecycleLoop manages device lifecycle (marking inactive devices)
func (s *Scanner) lifecycleLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkInactiveDevices()
		}
	}
}

// checkInactiveDevices marks devices as inactive if not seen recently
func (s *Scanner) checkInactiveDevices() {
	now := time.Now()
	inactiveThreshold := now.Add(-s.inactiveTimeout)

	s.devicesMu.Lock()
	defer s.devicesMu.Unlock()

	for mac, device := range s.devices {
		if device.IsActive && device.LastSeen.Before(inactiveThreshold) {
			device.IsActive = false
			
			// Save updated status to database
			if err := s.db.SaveDevice(device); err != nil {
				s.logger.Error("Error updating inactive status for device %s: %v", mac, err)
			} else {
				s.logger.Info("Device %s (%s) marked as inactive", mac, device.IP)
			}
		}
	}
}

// arpScanLoop runs ARP scanning at regular intervals
func (s *Scanner) arpScanLoop() {
	defer s.wg.Done()

	// Run initial scan immediately
	s.scanARP()

	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.scanARP()
		}
	}
}

// mdnsScanLoop runs mDNS scanning at regular intervals
func (s *Scanner) mdnsScanLoop() {
	defer s.wg.Done()

	// Run initial scan after a short delay
	time.Sleep(5 * time.Second)
	s.scanMDNS()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.scanMDNS()
		}
	}
}

// scanARP is implemented in arp.go

// scanMDNS is implemented in mdns.go
