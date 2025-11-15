package cloud

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/config"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

// Orchestrator manages cloud connector lifecycle and data transmission
type Orchestrator struct {
	connector        CloudConnector
	db               *database.DatabaseManager
	cfg              *config.CloudConfig
	transmitInterval time.Duration
	stopChan         chan struct{}
	wg               sync.WaitGroup
	mu               sync.RWMutex
}

// NewOrchestrator creates a new cloud connector orchestrator
func NewOrchestrator(cfg *config.CloudConfig, db *database.DatabaseManager) (*Orchestrator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cloud configuration is required")
	}

	if db == nil {
		return nil, fmt.Errorf("database manager is required")
	}

	transmitInterval := 5 * time.Minute // Default 5-minute interval

	return &Orchestrator{
		db:               db,
		cfg:              cfg,
		transmitInterval: transmitInterval,
		stopChan:         make(chan struct{}),
	}, nil
}

// Start initializes and starts the cloud connector
func (o *Orchestrator) Start() error {
	if !o.cfg.Enabled {
		log.Println("[Cloud Orchestrator] Cloud connector is disabled, skipping")
		return nil
	}

	log.Printf("[Cloud Orchestrator] Starting cloud connector for provider: %s", o.cfg.Provider)

	// Create connector based on provider
	// Note: We can't import aws/gcp packages here due to import cycles
	// The main orchestrator should handle this initialization
	if o.connector == nil {
		return fmt.Errorf("connector not initialized - call SetConnector first")
	}

	// Connect to cloud platform
	if err := o.connector.Connect(); err != nil {
		log.Printf("[Cloud Orchestrator] Failed to connect to cloud: %v", err)
		log.Println("[Cloud Orchestrator] Local operations will continue")
		// Don't return error - local operations should continue
		return nil
	}

	// Start transmission goroutine
	o.wg.Add(1)
	go o.transmissionLoop()

	log.Println("[Cloud Orchestrator] Cloud connector started successfully")
	return nil
}

// Stop gracefully stops the cloud connector
func (o *Orchestrator) Stop() error {
	if !o.cfg.Enabled || o.connector == nil {
		return nil
	}

	log.Println("[Cloud Orchestrator] Stopping cloud connector...")

	// Signal transmission loop to stop
	close(o.stopChan)

	// Wait for goroutines to finish
	o.wg.Wait()

	// Disconnect from cloud platform
	if err := o.connector.Disconnect(); err != nil {
		log.Printf("[Cloud Orchestrator] Error disconnecting: %v", err)
	}

	log.Println("[Cloud Orchestrator] Cloud connector stopped")
	return nil
}

// Name returns the component name
func (o *Orchestrator) Name() string {
	return "Cloud Orchestrator"
}

// SetConnector sets the cloud connector instance
// This is used to avoid import cycles
func (o *Orchestrator) SetConnector(connector CloudConnector) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connector = connector
}

// GetConnector returns the current connector instance
func (o *Orchestrator) GetConnector() CloudConnector {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.connector
}

// transmissionLoop periodically transmits profiles and devices to the cloud
func (o *Orchestrator) transmissionLoop() {
	defer o.wg.Done()

	ticker := time.NewTicker(o.transmitInterval)
	defer ticker.Stop()

	log.Printf("[Cloud Orchestrator] Transmission loop started with interval: %v", o.transmitInterval)

	// Perform initial transmission after a short delay
	time.Sleep(10 * time.Second)
	o.transmitData()

	for {
		select {
		case <-o.stopChan:
			log.Println("[Cloud Orchestrator] Transmission loop stopping")
			return
		case <-ticker.C:
			o.transmitData()
		}
	}
}

// transmitData sends profiles and devices to the cloud
func (o *Orchestrator) transmitData() {
	if o.connector == nil || !o.connector.IsConnected() {
		log.Println("[Cloud Orchestrator] Not connected, skipping transmission")
		return
	}

	log.Println("[Cloud Orchestrator] Starting data transmission...")

	// Transmit all profiles
	profiles, err := o.db.GetAllProfiles()
	if err != nil {
		log.Printf("[Cloud Orchestrator] Failed to get profiles: %v", err)
	} else {
		successCount := 0
		for _, profile := range profiles {
			if profile == nil {
				continue
			}

			// Enqueue profile for transmission with retry logic
			if err := o.enqueueWithRetry("profile", profile); err != nil {
				log.Printf("[Cloud Orchestrator] Failed to enqueue profile %s: %v", profile.MAC, err)
			} else {
				successCount++
			}
		}
		log.Printf("[Cloud Orchestrator] Enqueued %d/%d profiles", successCount, len(profiles))
	}

	// Transmit all devices
	devices, err := o.db.GetAllDevices()
	if err != nil {
		log.Printf("[Cloud Orchestrator] Failed to get devices: %v", err)
	} else {
		successCount := 0
		for _, device := range devices {
			if device == nil {
				continue
			}

			// Enqueue device for transmission with retry logic
			if err := o.enqueueWithRetry("device", device); err != nil {
				log.Printf("[Cloud Orchestrator] Failed to enqueue device %s: %v", device.MAC, err)
			} else {
				successCount++
			}
		}
		log.Printf("[Cloud Orchestrator] Enqueued %d/%d devices", successCount, len(devices))
	}

	log.Println("[Cloud Orchestrator] Data transmission completed")
}

// enqueueWithRetry attempts to enqueue data with exponential backoff
func (o *Orchestrator) enqueueWithRetry(dataType string, data interface{}) error {
	maxRetries := 3
	baseDelay := time.Second

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			log.Printf("[Cloud Orchestrator] Retry attempt %d/%d after %v", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		// Check if still connected
		if !o.connector.IsConnected() {
			return fmt.Errorf("connector disconnected")
		}

		// Attempt to send data
		switch dataType {
		case "profile":
			if profile, ok := data.(*database.BehavioralProfile); ok {
				err = o.connector.SendProfile(profile)
			} else {
				return fmt.Errorf("invalid profile data type")
			}
		case "device":
			if device, ok := data.(*database.Device); ok {
				err = o.connector.SendDevice(device)
			} else {
				return fmt.Errorf("invalid device data type")
			}
		default:
			return fmt.Errorf("unknown data type: %s", dataType)
		}

		if err == nil {
			return nil // Success
		}

		log.Printf("[Cloud Orchestrator] Transmission failed: %v", err)
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}

// IsConnected returns whether the cloud connector is connected
func (o *Orchestrator) IsConnected() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.connector == nil {
		return false
	}

	return o.connector.IsConnected()
}

// GetQueueSize returns the current transmission queue size
func (o *Orchestrator) GetQueueSize() int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.connector == nil {
		return 0
	}

	// Queue size tracking is internal to each connector implementation
	// This would require adding a GetQueueSize method to the CloudConnector interface
	// For now, return 0 as queue management is handled by the connector
	return 0
}
