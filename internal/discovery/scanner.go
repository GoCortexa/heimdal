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
	"github.com/mosiko1234/heimdal/sensor/internal/discovery/classifier"
	"github.com/mosiko1234/heimdal/sensor/internal/discovery/hostname"
	"github.com/mosiko1234/heimdal/sensor/internal/discovery/oui"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
)

// StatusLevel indicates the severity of a discovery status update.
type StatusLevel string

const (
	StatusLevelInfo    StatusLevel = "info"
	StatusLevelWarning StatusLevel = "warning"
	StatusLevelError   StatusLevel = "error"
)

// StatusUpdate captures scanner health or progress information.
type StatusUpdate struct {
	Level     StatusLevel
	Component string
	Message   string
	Time      time.Time
}

// StatusSink receives scanner status updates.
type StatusSink func(StatusUpdate)

// ScannerOptions exposes tuning knobs for discovery behavior.
type ScannerOptions struct {
	ARPReplyTimeout  time.Duration
	ARPMaxAttempts   int
	RetryDelay       time.Duration
	MDNSQueryTimeout time.Duration
}

func defaultScannerOptions() *ScannerOptions {
	return &ScannerOptions{
		ARPReplyTimeout:  3 * time.Second,
		ARPMaxAttempts:   2,
		RetryDelay:       1 * time.Second,
		MDNSQueryTimeout: 10 * time.Second,
	}
}

// Scanner manages device discovery operations
type NetworkConfigProvider interface {
	GetConfig() *netconfig.NetworkConfig
}

type Scanner struct {
	netConfig       NetworkConfigProvider
	db              database.DeviceStore
	deviceChan      chan<- *database.Device
	scanInterval    time.Duration
	mdnsEnabled     bool
	inactiveTimeout time.Duration
	logger          *logger.Logger
	options         *ScannerOptions
	statusSink      StatusSink

	// Device enrichment
	ouiLookup        *oui.OUILookup
	classifier       *classifier.Classifier
	hostnameResolver *hostname.Resolver

	// Internal state
	devices        map[string]*database.Device // MAC -> Device
	deviceServices map[string][]string         // MAC -> mDNS services
	devicesMu      sync.RWMutex

	// Lifecycle management
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	runningMu sync.Mutex
}

// NewScanner creates a new Scanner instance with the provided dependencies
func NewScanner(netConfig NetworkConfigProvider, db database.DeviceStore, deviceChan chan<- *database.Device, scanInterval time.Duration, mdnsEnabled bool, inactiveTimeout time.Duration, options *ScannerOptions, statusSink StatusSink) *Scanner {
	ctx, cancel := context.WithCancel(context.Background())

	if options == nil {
		options = defaultScannerOptions()
	}

	// Initialize OUI lookup
	ouiLookup := oui.NewOUILookup()
	if err := ouiLookup.Load(); err != nil {
		// Log error but continue - OUI lookup is optional
		log := logger.NewComponentLogger("Scanner")
		log.Warn("Failed to load OUI database: %v", err)
	}

	// Initialize device classifier
	deviceClassifier := classifier.NewClassifier()

	// Initialize hostname resolver
	hostnameResolver := hostname.NewResolver(2 * time.Second)

	return &Scanner{
		netConfig:        netConfig,
		db:               db,
		deviceChan:       deviceChan,
		scanInterval:     scanInterval,
		mdnsEnabled:      mdnsEnabled,
		inactiveTimeout:  inactiveTimeout,
		logger:           logger.NewComponentLogger("Scanner"),
		options:          options,
		statusSink:       statusSink,
		ouiLookup:        ouiLookup,
		classifier:       deviceClassifier,
		hostnameResolver: hostnameResolver,
		devices:          make(map[string]*database.Device),
		deviceServices:   make(map[string][]string),
		ctx:              ctx,
		cancel:           cancel,
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
		s.reportStatus(StatusLevelWarning, "Failed to load existing devices: %v", err)
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

	// Start hostname enrichment goroutine
	s.wg.Add(1)
	go s.hostnameEnrichmentLoop()

	s.logger.Info("Device discovery scanner started (scan interval: %v)", s.scanInterval)
	if cfg := s.netConfig.GetConfig(); cfg != nil {
		s.reportStatus(StatusLevelInfo, "Discovery scanner running on %s (%s)", cfg.Interface, cfg.CIDR)
	} else {
		s.reportStatus(StatusLevelInfo, "Discovery scanner started (interval %v)", s.scanInterval)
	}
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
		s.reportStatus(StatusLevelInfo, "Discovery scanner stopped")
		return nil
	case <-time.After(5 * time.Second):
		// Even if timed out, we consider it stopped as context is cancelled
		s.logger.Warn("Device discovery scanner shutdown timed out, forcing stop")
		return nil
	}
}

func (s *Scanner) reportStatus(level StatusLevel, format string, args ...interface{}) {
	if s.statusSink == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	s.statusSink(StatusUpdate{
		Level:     level,
		Component: "discovery",
		Message:   msg,
		Time:      time.Now(),
	})
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

		// Re-enrich if vendor/manufacturer missing (for existing devices)
		if device.Vendor == "" || device.Manufacturer == "" {
			if s.ouiLookup != nil && s.ouiLookup.IsLoaded() {
				vendorName, manufacturerName, found := s.ouiLookup.Lookup(mac)
				if found {
					if device.Vendor == "" {
						device.Vendor = vendorName
					}
					if device.Manufacturer == "" {
						device.Manufacturer = manufacturerName
					}
					s.logger.Debug("Re-enriched existing device %s with vendor: %s", mac, vendorName)
				}
			}
		}

		// Re-classify if device type missing
		if device.DeviceType == "" && s.classifier != nil {
			services := s.deviceServices[mac]
			classInfo := s.classifier.ClassifyDevice(device.Vendor, device.Manufacturer, device.Name, services)
			device.DeviceType = string(classInfo.Type)
			device.Services = services
			s.logger.Debug("Re-classified existing device %s as %s", mac, classInfo.Type)
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

		// Enrich with OUI lookup if available
		if s.ouiLookup != nil && s.ouiLookup.IsLoaded() {
			vendorName, manufacturerName, found := s.ouiLookup.Lookup(mac)
			if found {
				if device.Vendor == "" {
					device.Vendor = vendorName
				}
				device.Manufacturer = manufacturerName
				s.logger.Debug("Enriched device %s with vendor: %s", mac, vendorName)
			}
		}

		// Classify device type
		if s.classifier != nil {
			// Get services for this device
			services := s.deviceServices[mac]

			classInfo := s.classifier.ClassifyDevice(device.Vendor, device.Manufacturer, device.Name, services)
			device.DeviceType = string(classInfo.Type)
			device.Services = services
			s.logger.Debug("Classified device %s as %s (confidence: %.2f, signals: %v)",
				mac, classInfo.Type, classInfo.Confidence, classInfo.Signals)
		}
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
	if s.deviceChan != nil {
		select {
		case s.deviceChan <- &deviceCopy:
			s.logger.Debug("Device %s sent to channel", mac)
		default:
			s.logger.Warn("Device channel full, dropping device update for %s", mac)
		}
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
	s.runARPScanWithRetry()

	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runARPScanWithRetry()
		}
	}
}

func (s *Scanner) runARPScanWithRetry() {
	attempts := s.options.ARPMaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		count, err := s.scanARP()
		if err == nil {
			s.reportStatus(StatusLevelInfo, "ARP scan completed (%d devices)", count)
			return
		}

		lastErr = err
		level := StatusLevelWarning
		if isPermissionError(err) {
			level = StatusLevelError
		}
		s.reportStatus(level, "ARP scan attempt %d/%d failed: %v", attempt, attempts, err)

		if attempt < attempts {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(s.options.RetryDelay):
			}
		}
	}

	if lastErr != nil {
		s.logger.Warn("ARP scan failed after %d attempts: %v", s.options.ARPMaxAttempts, lastErr)
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

// hostnameEnrichmentLoop periodically enriches devices with hostname information
func (s *Scanner) hostnameEnrichmentLoop() {
	defer s.wg.Done()

	// Wait a bit before starting to let initial discovery complete
	time.Sleep(30 * time.Second)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.enrichHostnames()
		}
	}
}

// enrichHostnames resolves hostnames for devices that don't have one
func (s *Scanner) enrichHostnames() {
	if s.hostnameResolver == nil {
		return
	}

	s.devicesMu.RLock()
	devicesToEnrich := make([]*database.Device, 0)
	for _, device := range s.devices {
		// Only enrich active devices without hostname
		if device.IsActive && device.Hostname == "" {
			devicesToEnrich = append(devicesToEnrich, device)
		}
	}
	s.devicesMu.RUnlock()

	if len(devicesToEnrich) == 0 {
		return
	}

	s.logger.Info("Enriching hostnames for %d devices", len(devicesToEnrich))

	// Resolve hostnames concurrently (but rate-limited)
	for _, device := range devicesToEnrich {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Launch async resolution
		go func(dev *database.Device) {
			resultCh := s.hostnameResolver.ResolveAsync(dev.IP, dev.Name)
			result := <-resultCh

			if result.Error == nil && result.Hostname != "" {
				// Update device with hostname
				s.devicesMu.Lock()
				if storedDev, exists := s.devices[dev.MAC]; exists {
					storedDev.Hostname = result.Hostname
					// Save to database
					if err := s.db.SaveDevice(storedDev); err != nil {
						s.logger.Error("Failed to save hostname for device %s: %v", dev.MAC, err)
					} else {
						s.logger.Info("Resolved hostname for %s: %s (method: %s)", dev.MAC, result.Hostname, result.Method)
					}
				}
				s.devicesMu.Unlock()
			}
		}(device)

		// Rate limit: 1 device per second to avoid overwhelming DNS
		time.Sleep(1 * time.Second)
	}
}

// scanARP is implemented in arp.go

// scanMDNS is implemented in mdns.go
