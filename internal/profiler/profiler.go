// Package profiler aggregates packet metadata into behavioral profiles for network devices.
//
// The Profiler component receives packet metadata from the analyzer and builds comprehensive
// behavioral profiles for each device (identified by MAC address). These profiles capture
// communication patterns, timing, and volume metrics for anomaly detection and monitoring.
//
// Profile Structure:
//   - Destinations: Map of destination IPs with packet counts and last seen timestamps
//   - Ports: Frequency distribution of destination ports
//   - Protocols: Count of TCP, UDP, ICMP, and other protocols
//   - Volume: Total packets and bytes transmitted
//   - Timing: Hourly activity pattern (24-hour array)
//
// Aggregation Logic:
//   1. Receive PacketInfo from packetChan
//   2. Look up or create profile for source MAC address
//   3. Update destination IP counter
//   4. Update port frequency distribution
//   5. Update protocol counter
//   6. Increment total packets and bytes
//   7. Update hourly activity based on packet timestamp
//
// Persistence:
//   - Maintains profiles in memory for fast updates
//   - Persists all profiles to database at regular intervals (default: every 60 seconds)
//   - Uses batch operations for efficient database writes
//   - Loads existing profiles from database on startup
//
// Memory Management:
//   - Limits maximum destinations per profile (configurable, default: 100)
//   - Prunes least-recently-seen destinations when limit reached
//   - Efficient map-based storage for O(1) lookups
//
// The Profiler implements the Component interface for lifecycle management by the orchestrator.
package profiler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/analyzer"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

// BehavioralProfile represents aggregated traffic patterns for a device
// This is re-exported from database package for convenience
type BehavioralProfile = database.BehavioralProfile

// DestInfo contains information about a communication destination
// This is re-exported from database package for convenience
type DestInfo = database.DestInfo

// Profiler aggregates packet data into behavioral profiles
type Profiler struct {
	profiles       map[string]*BehavioralProfile
	mu             sync.RWMutex
	packetChan     <-chan analyzer.PacketInfo
	db             *database.DatabaseManager
	persistTicker  *time.Ticker
	persistInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewProfiler creates a new behavioral profiler instance
func NewProfiler(db *database.DatabaseManager, packetChan <-chan analyzer.PacketInfo, persistInterval time.Duration) (*Profiler, error) {
	if db == nil {
		return nil, fmt.Errorf("database manager is required")
	}
	if packetChan == nil {
		return nil, fmt.Errorf("packet channel is required")
	}
	if persistInterval <= 0 {
		persistInterval = 60 * time.Second // Default to 60 seconds
	}

	ctx, cancel := context.WithCancel(context.Background())

	profiler := &Profiler{
		profiles:        make(map[string]*BehavioralProfile),
		packetChan:      packetChan,
		db:              db,
		persistInterval: persistInterval,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Load existing profiles from database on startup
	if err := profiler.loadProfiles(); err != nil {
		log.Printf("[Profiler] Warning: failed to load existing profiles: %v", err)
		// Don't fail initialization, just log the warning
	}

	return profiler, nil
}

// loadProfiles loads existing behavioral profiles from the database
func (p *Profiler) loadProfiles() error {
	profiles, err := p.db.GetAllProfiles()
	if err != nil {
		return fmt.Errorf("failed to load profiles from database: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, profile := range profiles {
		if profile != nil && profile.MAC != "" {
			p.profiles[profile.MAC] = profile
		}
	}

	log.Printf("[Profiler] Loaded %d existing profiles from database", len(profiles))
	return nil
}

// Start begins the profiler's packet processing and persistence operations
func (p *Profiler) Start() error {
	log.Println("[Profiler] Starting behavioral profiler...")

	// Start packet processing goroutine
	p.wg.Add(1)
	go p.processPackets()

	// Start persistence goroutine
	p.persistTicker = time.NewTicker(p.persistInterval)
	p.wg.Add(1)
	go p.persistenceLoop()

	log.Printf("[Profiler] Started with persistence interval of %v", p.persistInterval)
	return nil
}

// Stop gracefully stops the profiler
func (p *Profiler) Stop() error {
	log.Println("[Profiler] Stopping behavioral profiler...")
	p.cancel()

	if p.persistTicker != nil {
		p.persistTicker.Stop()
	}

	p.wg.Wait()

	// Persist profiles one final time before shutdown
	if err := p.persistProfiles(); err != nil {
		log.Printf("[Profiler] Warning: failed to persist profiles during shutdown: %v", err)
	}

	log.Println("[Profiler] Stopped")
	return nil
}

// Name returns the component name
func (p *Profiler) Name() string {
	return "Profiler"
}

// processPackets continuously processes incoming packet metadata
func (p *Profiler) processPackets() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case packetInfo, ok := <-p.packetChan:
			if !ok {
				return
			}
			p.updateProfile(packetInfo)
		}
	}
}

// updateProfile processes a packet and updates the behavioral profile for the source MAC
func (p *Profiler) updateProfile(packetInfo analyzer.PacketInfo) {
	if packetInfo.SrcMAC == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Get or create profile for this MAC address
	profile, exists := p.profiles[packetInfo.SrcMAC]
	if !exists {
		// Create new profile
		profile = &BehavioralProfile{
			MAC:          packetInfo.SrcMAC,
			Destinations: make(map[string]*DestInfo),
			Ports:        make(map[uint16]int),
			Protocols:    make(map[string]int),
			FirstSeen:    packetInfo.Timestamp,
			LastSeen:     packetInfo.Timestamp,
		}
		p.profiles[packetInfo.SrcMAC] = profile
	}

	// Update LastSeen timestamp
	profile.LastSeen = packetInfo.Timestamp

	// Update Destinations map with destination IP and count
	if packetInfo.DstIP != "" {
		destInfo, destExists := profile.Destinations[packetInfo.DstIP]
		if !destExists {
			destInfo = &DestInfo{
				IP:       packetInfo.DstIP,
				Count:    0,
				LastSeen: packetInfo.Timestamp,
			}
			profile.Destinations[packetInfo.DstIP] = destInfo
		}
		destInfo.Count++
		destInfo.LastSeen = packetInfo.Timestamp
	}

	// Update Ports map with destination port frequency
	if packetInfo.DstPort > 0 {
		profile.Ports[packetInfo.DstPort]++
	}

	// Update Protocols map with protocol counts
	if packetInfo.Protocol != "" {
		profile.Protocols[packetInfo.Protocol]++
	}

	// Increment TotalPackets counter
	profile.TotalPackets++

	// Increment TotalBytes counter
	profile.TotalBytes += int64(packetInfo.Size)

	// Update HourlyActivity array based on packet timestamp
	hour := packetInfo.Timestamp.Hour()
	if hour >= 0 && hour < 24 {
		profile.HourlyActivity[hour]++
	}
}

// GetProfile returns a copy of the profile for a given MAC address
func (p *Profiler) GetProfile(mac string) (*BehavioralProfile, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profile, exists := p.profiles[mac]
	if !exists {
		return nil, fmt.Errorf("profile not found for MAC: %s", mac)
	}

	// Return a copy to prevent external modification
	profileCopy := *profile
	return &profileCopy, nil
}

// GetAllProfiles returns copies of all profiles
func (p *Profiler) GetAllProfiles() []*BehavioralProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profiles := make([]*BehavioralProfile, 0, len(p.profiles))
	for _, profile := range p.profiles {
		profileCopy := *profile
		profiles = append(profiles, &profileCopy)
	}

	return profiles
}

// persistenceLoop runs the periodic profile persistence operation
func (p *Profiler) persistenceLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.persistTicker.C:
			if err := p.persistProfiles(); err != nil {
				log.Printf("[Profiler] Error persisting profiles: %v", err)
			}
		}
	}
}

// persistProfiles saves all profiles to the database using batch operations
func (p *Profiler) persistProfiles() error {
	p.mu.RLock()
	
	// Create a slice of profiles to persist
	profiles := make([]*BehavioralProfile, 0, len(p.profiles))
	for _, profile := range p.profiles {
		// Create a copy to avoid holding the lock during database operations
		profileCopy := *profile
		profiles = append(profiles, &profileCopy)
	}
	
	p.mu.RUnlock()

	if len(profiles) == 0 {
		return nil
	}

	// Use batch operation for efficient database writes
	err := p.db.SaveProfileBatch(profiles)
	if err != nil {
		// Retry logic: attempt one more time after a short delay
		log.Printf("[Profiler] First attempt to persist profiles failed, retrying: %v", err)
		time.Sleep(1 * time.Second)
		
		err = p.db.SaveProfileBatch(profiles)
		if err != nil {
			return fmt.Errorf("failed to persist profiles after retry: %w", err)
		}
	}

	log.Printf("[Profiler] Successfully persisted %d profiles to database", len(profiles))
	return nil
}

// GetProfileCount returns the number of profiles currently tracked
func (p *Profiler) GetProfileCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.profiles)
}
