// Package database provides persistent storage for Heimdal sensor data using BadgerDB.
//
// BadgerDB is an embedded, pure-Go key-value database with LSM-tree architecture,
// optimized for high write throughput and excellent concurrency support. It requires
// no external dependencies, making it ideal for cross-compilation to ARM64.
//
// The database stores two primary data types:
//   - Devices: Network devices discovered via ARP and mDNS scanning
//   - Behavioral Profiles: Aggregated traffic patterns per device MAC address
//
// Data is stored with prefixed keys:
//   - device:<MAC_ADDRESS>   → JSON-serialized Device struct
//   - profile:<MAC_ADDRESS>  → JSON-serialized BehavioralProfile struct
//   - meta:config            → System metadata
//
// The DatabaseManager provides CRUD operations, batch operations for efficient writes,
// and in-memory buffering when the database is temporarily unavailable. It implements
// automatic retry with exponential backoff for transient errors and periodic garbage
// collection to reclaim disk space.
package database

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/mosiko1234/heimdal/sensor/internal/errors"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
)

// Key prefix constants
const (
	DevicePrefix  = "device:"
	ProfilePrefix = "profile:"
	MetaPrefix    = "meta:"
)

// Device represents a discovered network device
type Device struct {
	MAC       string    `json:"mac"`
	IP        string    `json:"ip"`
	Name      string    `json:"name"`
	Vendor    string    `json:"vendor"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	IsActive  bool      `json:"is_active"`
}

// BehavioralProfile represents aggregated traffic patterns for a device
type BehavioralProfile struct {
	MAC            string                `json:"mac"`
	Destinations   map[string]*DestInfo  `json:"destinations"`
	Ports          map[uint16]int        `json:"ports"`
	Protocols      map[string]int        `json:"protocols"`
	TotalPackets   int64                 `json:"total_packets"`
	TotalBytes     int64                 `json:"total_bytes"`
	FirstSeen      time.Time             `json:"first_seen"`
	LastSeen       time.Time             `json:"last_seen"`
	HourlyActivity [24]int               `json:"hourly_activity"`
}

// DestInfo contains information about a communication destination
type DestInfo struct {
	IP       string    `json:"ip"`
	Count    int64     `json:"count"`
	LastSeen time.Time `json:"last_seen"`
}

// MemoryBuffer provides in-memory storage when database is unavailable
type MemoryBuffer struct {
	devices  map[string]*Device
	profiles map[string]*BehavioralProfile
	maxSize  int
	mu       sync.RWMutex
}

// DatabaseManager manages BadgerDB operations
type DatabaseManager struct {
	db     *badger.DB
	path   string
	buffer *MemoryBuffer
	logger *logger.Logger
	mu     sync.RWMutex
}

// NewDatabaseManager initializes a new DatabaseManager with BadgerDB
func NewDatabaseManager(path string) (*DatabaseManager, error) {
	log := logger.NewComponentLogger("Database")
	
	// Configure BadgerDB options
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable BadgerDB's default logger

	log.Info("Opening database at %s", path)
	
	// Open BadgerDB
	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open BadgerDB at %s", path)
	}

	// Initialize memory buffer for fallback
	buffer := &MemoryBuffer{
		devices:  make(map[string]*Device),
		profiles: make(map[string]*BehavioralProfile),
		maxSize:  1000,
	}

	dm := &DatabaseManager{
		db:     db,
		path:   path,
		buffer: buffer,
		logger: log,
	}

	log.Info("Database initialized successfully")
	return dm, nil
}

// Close gracefully shuts down the database
func (dm *DatabaseManager) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.logger.Info("Closing database...")
	
	if dm.db != nil {
		if err := dm.db.Close(); err != nil {
			dm.logger.Error("Failed to close database: %v", err)
			return errors.Wrap(err, "failed to close database")
		}
	}

	dm.logger.Info("Database closed successfully")
	return nil
}

// SaveDevice persists a device to the database with JSON serialization
func (dm *DatabaseManager) SaveDevice(device *Device) error {
	if device == nil {
		return fmt.Errorf("device cannot be nil")
	}
	if device.MAC == "" {
		return fmt.Errorf("device MAC address cannot be empty")
	}

	// Serialize device to JSON
	data, err := json.Marshal(device)
	if err != nil {
		return errors.Wrap(err, "failed to serialize device %s", device.MAC)
	}

	// Create key with prefix
	key := []byte(DevicePrefix + device.MAC)

	// Write to database with retry logic
	err = errors.RetryWithBackoff("save device", errors.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}, func() error {
		return dm.db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, data)
		})
	})

	if err != nil {
		// If database write fails, buffer in memory
		dm.buffer.mu.Lock()
		if len(dm.buffer.devices) < dm.buffer.maxSize {
			dm.buffer.devices[device.MAC] = device
			dm.logger.Warn("Device %s buffered in memory due to database error", device.MAC)
		} else {
			dm.logger.Error("Memory buffer full, dropping device %s", device.MAC)
		}
		dm.buffer.mu.Unlock()
		return errors.Wrap(err, "failed to save device to database (buffered in memory)")
	}

	dm.logger.Debug("Device %s saved successfully", device.MAC)
	return nil
}

// GetDevice retrieves a device from the database by MAC address
func (dm *DatabaseManager) GetDevice(mac string) (*Device, error) {
	if mac == "" {
		return nil, fmt.Errorf("MAC address cannot be empty")
	}

	// Check memory buffer first
	dm.buffer.mu.RLock()
	if device, exists := dm.buffer.devices[mac]; exists {
		dm.buffer.mu.RUnlock()
		return device, nil
	}
	dm.buffer.mu.RUnlock()

	// Create key with prefix
	key := []byte(DevicePrefix + mac)

	var device Device
	err := dm.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &device)
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("device not found: %s", mac)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve device: %w", err)
	}

	return &device, nil
}

// GetAllDevices retrieves all devices from the database
func (dm *DatabaseManager) GetAllDevices() ([]*Device, error) {
	devices := make([]*Device, 0)

	// Add devices from memory buffer
	dm.buffer.mu.RLock()
	for _, device := range dm.buffer.devices {
		devices = append(devices, device)
	}
	dm.buffer.mu.RUnlock()

	// Iterate through database
	err := dm.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(DevicePrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				var device Device
				if err := json.Unmarshal(val, &device); err != nil {
					return err
				}
				devices = append(devices, &device)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve devices: %w", err)
	}

	return devices, nil
}

// DeleteDevice removes a device from the database
func (dm *DatabaseManager) DeleteDevice(mac string) error {
	if mac == "" {
		return fmt.Errorf("MAC address cannot be empty")
	}

	// Remove from memory buffer
	dm.buffer.mu.Lock()
	delete(dm.buffer.devices, mac)
	dm.buffer.mu.Unlock()

	// Create key with prefix
	key := []byte(DevicePrefix + mac)

	// Delete from database
	err := dm.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})

	if err != nil && err != badger.ErrKeyNotFound {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// SaveProfile persists a behavioral profile to the database with JSON serialization
func (dm *DatabaseManager) SaveProfile(profile *BehavioralProfile) error {
	if profile == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	if profile.MAC == "" {
		return fmt.Errorf("profile MAC address cannot be empty")
	}

	// Serialize profile to JSON
	data, err := json.Marshal(profile)
	if err != nil {
		return errors.Wrap(err, "failed to serialize profile %s", profile.MAC)
	}

	// Create key with prefix
	key := []byte(ProfilePrefix + profile.MAC)

	// Write to database with retry logic
	err = errors.RetryWithBackoff("save profile", errors.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}, func() error {
		return dm.db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, data)
		})
	})

	if err != nil {
		// If database write fails, buffer in memory
		dm.buffer.mu.Lock()
		if len(dm.buffer.profiles) < dm.buffer.maxSize {
			dm.buffer.profiles[profile.MAC] = profile
			dm.logger.Warn("Profile %s buffered in memory due to database error", profile.MAC)
		} else {
			dm.logger.Error("Memory buffer full, dropping profile %s", profile.MAC)
		}
		dm.buffer.mu.Unlock()
		return errors.Wrap(err, "failed to save profile to database (buffered in memory)")
	}

	return nil
}

// GetProfile retrieves a behavioral profile from the database by MAC address
func (dm *DatabaseManager) GetProfile(mac string) (*BehavioralProfile, error) {
	if mac == "" {
		return nil, fmt.Errorf("MAC address cannot be empty")
	}

	// Check memory buffer first
	dm.buffer.mu.RLock()
	if profile, exists := dm.buffer.profiles[mac]; exists {
		dm.buffer.mu.RUnlock()
		return profile, nil
	}
	dm.buffer.mu.RUnlock()

	// Create key with prefix
	key := []byte(ProfilePrefix + mac)

	var profile BehavioralProfile
	err := dm.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &profile)
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("profile not found: %s", mac)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve profile: %w", err)
	}

	return &profile, nil
}

// GetAllProfiles retrieves all behavioral profiles from the database
func (dm *DatabaseManager) GetAllProfiles() ([]*BehavioralProfile, error) {
	profiles := make([]*BehavioralProfile, 0)

	// Add profiles from memory buffer
	dm.buffer.mu.RLock()
	for _, profile := range dm.buffer.profiles {
		profiles = append(profiles, profile)
	}
	dm.buffer.mu.RUnlock()

	// Iterate through database
	err := dm.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(ProfilePrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				var profile BehavioralProfile
				if err := json.Unmarshal(val, &profile); err != nil {
					return err
				}
				profiles = append(profiles, &profile)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve profiles: %w", err)
	}

	return profiles, nil
}

// SaveDeviceBatch performs bulk device writes to the database
func (dm *DatabaseManager) SaveDeviceBatch(devices []*Device) error {
	if len(devices) == 0 {
		return nil
	}

	// Use a write batch for efficiency
	err := dm.db.Update(func(txn *badger.Txn) error {
		for _, device := range devices {
			if device == nil || device.MAC == "" {
				continue
			}

			// Serialize device to JSON
			data, err := json.Marshal(device)
			if err != nil {
				return fmt.Errorf("failed to serialize device %s: %w", device.MAC, err)
			}

			// Create key with prefix
			key := []byte(DevicePrefix + device.MAC)

			// Set in transaction
			if err := txn.Set(key, data); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		// If batch write fails, buffer devices in memory
		dm.buffer.mu.Lock()
		for _, device := range devices {
			if device != nil && device.MAC != "" {
				if len(dm.buffer.devices) < dm.buffer.maxSize {
					dm.buffer.devices[device.MAC] = device
				}
			}
		}
		dm.buffer.mu.Unlock()
		return fmt.Errorf("failed to save device batch (buffered in memory): %w", err)
	}

	return nil
}

// SaveProfileBatch performs bulk profile writes to the database
func (dm *DatabaseManager) SaveProfileBatch(profiles []*BehavioralProfile) error {
	if len(profiles) == 0 {
		return nil
	}

	// Use a write batch for efficiency
	err := dm.db.Update(func(txn *badger.Txn) error {
		for _, profile := range profiles {
			if profile == nil || profile.MAC == "" {
				continue
			}

			// Serialize profile to JSON
			data, err := json.Marshal(profile)
			if err != nil {
				return fmt.Errorf("failed to serialize profile %s: %w", profile.MAC, err)
			}

			// Create key with prefix
			key := []byte(ProfilePrefix + profile.MAC)

			// Set in transaction
			if err := txn.Set(key, data); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		// If batch write fails, buffer profiles in memory
		dm.buffer.mu.Lock()
		for _, profile := range profiles {
			if profile != nil && profile.MAC != "" {
				if len(dm.buffer.profiles) < dm.buffer.maxSize {
					dm.buffer.profiles[profile.MAC] = profile
				}
			}
		}
		dm.buffer.mu.Unlock()
		return fmt.Errorf("failed to save profile batch (buffered in memory): %w", err)
	}

	return nil
}

// FlushBuffer attempts to write buffered data to the database
func (dm *DatabaseManager) FlushBuffer() error {
	dm.buffer.mu.Lock()
	defer dm.buffer.mu.Unlock()

	// Flush buffered devices
	if len(dm.buffer.devices) > 0 {
		devices := make([]*Device, 0, len(dm.buffer.devices))
		for _, device := range dm.buffer.devices {
			devices = append(devices, device)
		}

		err := dm.db.Update(func(txn *badger.Txn) error {
			for _, device := range devices {
				data, err := json.Marshal(device)
				if err != nil {
					return fmt.Errorf("failed to serialize device %s: %w", device.MAC, err)
				}

				key := []byte(DevicePrefix + device.MAC)
				if err := txn.Set(key, data); err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to flush device buffer: %w", err)
		}

		// Clear buffer after successful flush
		dm.buffer.devices = make(map[string]*Device)
	}

	// Flush buffered profiles
	if len(dm.buffer.profiles) > 0 {
		profiles := make([]*BehavioralProfile, 0, len(dm.buffer.profiles))
		for _, profile := range dm.buffer.profiles {
			profiles = append(profiles, profile)
		}

		err := dm.db.Update(func(txn *badger.Txn) error {
			for _, profile := range profiles {
				data, err := json.Marshal(profile)
				if err != nil {
					return fmt.Errorf("failed to serialize profile %s: %w", profile.MAC, err)
				}

				key := []byte(ProfilePrefix + profile.MAC)
				if err := txn.Set(key, data); err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to flush profile buffer: %w", err)
		}

		// Clear buffer after successful flush
		dm.buffer.profiles = make(map[string]*BehavioralProfile)
	}

	return nil
}

// GetBufferSize returns the current number of items in the memory buffer
func (dm *DatabaseManager) GetBufferSize() (devices int, profiles int) {
	dm.buffer.mu.RLock()
	defer dm.buffer.mu.RUnlock()
	return len(dm.buffer.devices), len(dm.buffer.profiles)
}

// IsBufferFull returns true if the memory buffer has reached capacity
func (dm *DatabaseManager) IsBufferFull() bool {
	dm.buffer.mu.RLock()
	defer dm.buffer.mu.RUnlock()
	return len(dm.buffer.devices) >= dm.buffer.maxSize || len(dm.buffer.profiles) >= dm.buffer.maxSize
}
