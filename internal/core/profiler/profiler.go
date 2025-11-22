// Package profiler provides shared behavioral profiling functionality for both
// hardware and desktop products.
//
// The Profiler aggregates packet metadata into behavioral profiles for network
// devices, capturing communication patterns, timing, and volume metrics.
package profiler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/core/packet"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// Profiler aggregates packet data into behavioral profiles
type Profiler struct {
	profiles        map[string]*database.BehavioralProfile
	mu              sync.RWMutex
	packetChan      <-chan packet.PacketInfo
	storage         platform.StorageProvider
	persistTicker   *time.Ticker
	persistInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// Config contains configuration for the profiler
type Config struct {
	// PersistInterval is how often to persist profiles to storage
	PersistInterval time.Duration
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		PersistInterval: 60 * time.Second, // Persist every 60 seconds
	}
}

// NewProfiler creates a new behavioral profiler instance
func NewProfiler(storage platform.StorageProvider, packetChan <-chan packet.PacketInfo, cfg *Config) (*Profiler, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage provider is required")
	}
	if packetChan == nil {
		return nil, fmt.Errorf("packet channel is required")
	}
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.PersistInterval <= 0 {
		cfg.PersistInterval = 60 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	profiler := &Profiler{
		profiles:        make(map[string]*database.BehavioralProfile),
		packetChan:      packetChan,
		storage:         storage,
		persistInterval: cfg.PersistInterval,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Load existing profiles from storage on startup
	if err := profiler.loadProfiles(); err != nil {
		log.Printf("[Profiler] Warning: failed to load existing profiles: %v", err)
		// Don't fail initialization, just log the warning
	}

	return profiler, nil
}

// loadProfiles loads existing behavioral profiles from storage
func (p *Profiler) loadProfiles() error {
	// List all profile keys
	keys, err := p.storage.List("profile:")
	if err != nil {
		return fmt.Errorf("failed to list profiles from storage: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, key := range keys {
		// Get profile data
		data, err := p.storage.Get(key)
		if err != nil {
			log.Printf("[Profiler] Warning: failed to load profile %s: %v", key, err)
			continue
		}

		// Deserialize profile
		var profile database.BehavioralProfile
		if err := json.Unmarshal(data, &profile); err != nil {
			log.Printf("[Profiler] Warning: failed to unmarshal profile %s: %v", key, err)
			continue
		}

		if profile.MAC != "" {
			p.profiles[profile.MAC] = &profile
		}
	}

	log.Printf("[Profiler] Loaded %d existing profiles from storage", len(p.profiles))
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
func (p *Profiler) updateProfile(packetInfo packet.PacketInfo) {
	if packetInfo.SrcMAC == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Get or create profile for this MAC address
	profile, exists := p.profiles[packetInfo.SrcMAC]
	if !exists {
		// Create new profile
		profile = &database.BehavioralProfile{
			MAC:          packetInfo.SrcMAC,
			Destinations: make(map[string]*database.DestInfo),
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
			destInfo = &database.DestInfo{
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
func (p *Profiler) GetProfile(mac string) (*database.BehavioralProfile, error) {
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
func (p *Profiler) GetAllProfiles() []*database.BehavioralProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profiles := make([]*database.BehavioralProfile, 0, len(p.profiles))
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

// persistProfiles saves all profiles to storage using batch operations
func (p *Profiler) persistProfiles() error {
	p.mu.RLock()

	// Create a slice of profiles to persist
	profiles := make([]*database.BehavioralProfile, 0, len(p.profiles))
	for _, profile := range p.profiles {
		// Create a copy to avoid holding the lock during storage operations
		profileCopy := *profile
		profiles = append(profiles, &profileCopy)
	}

	p.mu.RUnlock()

	if len(profiles) == 0 {
		return nil
	}

	// Prepare batch operations
	ops := make([]platform.BatchOp, 0, len(profiles))
	for _, profile := range profiles {
		// Serialize profile to JSON
		data, err := json.Marshal(profile)
		if err != nil {
			log.Printf("[Profiler] Warning: failed to serialize profile %s: %v", profile.MAC, err)
			continue
		}

		// Create batch operation
		op := platform.BatchOp{
			Type:  platform.BatchOpSet,
			Key:   "profile:" + profile.MAC,
			Value: data,
		}
		ops = append(ops, op)
	}

	// Execute batch operation
	if len(ops) > 0 {
		err := p.storage.Batch(ops)
		if err != nil {
			// Retry logic: attempt one more time after a short delay
			log.Printf("[Profiler] First attempt to persist profiles failed, retrying: %v", err)
			time.Sleep(1 * time.Second)

			err = p.storage.Batch(ops)
			if err != nil {
				return fmt.Errorf("failed to persist profiles after retry: %w", err)
			}
		}

		log.Printf("[Profiler] Successfully persisted %d profiles to storage", len(ops))
	}

	return nil
}

// GetProfileCount returns the number of profiles currently tracked
func (p *Profiler) GetProfileCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.profiles)
}

// DeleteProfile removes a profile from memory and storage
func (p *Profiler) DeleteProfile(mac string) error {
	p.mu.Lock()
	delete(p.profiles, mac)
	p.mu.Unlock()

	// Delete from storage
	key := "profile:" + mac
	if err := p.storage.Delete(key); err != nil {
		return fmt.Errorf("failed to delete profile from storage: %w", err)
	}

	return nil
}

// ClearProfiles removes all profiles from memory (does not affect storage)
func (p *Profiler) ClearProfiles() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.profiles = make(map[string]*database.BehavioralProfile)
}
