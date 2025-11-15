package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/config"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

// CloudConnector defines the interface for cloud platform connectivity
type CloudConnector interface {
	// Connect establishes connection to the cloud platform
	Connect() error

	// Disconnect closes the connection to the cloud platform
	Disconnect() error

	// SendProfile transmits a behavioral profile to the cloud
	SendProfile(profile *database.BehavioralProfile) error

	// SendDevice transmits device information to the cloud
	SendDevice(device *database.Device) error

	// IsConnected returns the current connection status
	IsConnected() bool
}

// TransmissionItem represents an item in the transmission queue
type TransmissionItem struct {
	Type      string      // "profile" or "device"
	Data      interface{} // *BehavioralProfile or *Device
	Timestamp time.Time
	Retries   int
}

// BaseConnector provides common functionality for cloud connectors
type BaseConnector struct {
	connected      bool
	mu             sync.RWMutex
	queue          []*TransmissionItem
	queueMu        sync.Mutex
	maxQueueSize   int
	maxRetries     int
	retryDelay     time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	db             *database.DatabaseManager
	transmitTicker *time.Ticker
	transmitInterval time.Duration
}

// NewBaseConnector creates a new base connector with common functionality
func NewBaseConnector(db *database.DatabaseManager, transmitInterval time.Duration) *BaseConnector {
	ctx, cancel := context.WithCancel(context.Background())

	if transmitInterval <= 0 {
		transmitInterval = 5 * time.Minute // Default to 5 minutes
	}

	return &BaseConnector{
		connected:        false,
		queue:            make([]*TransmissionItem, 0),
		maxQueueSize:     100,
		maxRetries:       3,
		retryDelay:       time.Second,
		ctx:              ctx,
		cancel:           cancel,
		db:               db,
		transmitInterval: transmitInterval,
	}
}

// SetConnected updates the connection status
func (bc *BaseConnector) SetConnected(connected bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.connected = connected
}

// IsConnected returns the current connection status
func (bc *BaseConnector) IsConnected() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.connected
}

// EnqueueProfile adds a profile to the transmission queue
func (bc *BaseConnector) EnqueueProfile(profile *database.BehavioralProfile) error {
	return bc.enqueue("profile", profile)
}

// EnqueueDevice adds a device to the transmission queue
func (bc *BaseConnector) EnqueueDevice(device *database.Device) error {
	return bc.enqueue("device", device)
}

// enqueue adds an item to the transmission queue
func (bc *BaseConnector) enqueue(itemType string, data interface{}) error {
	bc.queueMu.Lock()
	defer bc.queueMu.Unlock()

	// Check if queue is full
	if len(bc.queue) >= bc.maxQueueSize {
		// Remove oldest item to make room
		log.Printf("[CloudConnector] Queue full, dropping oldest item")
		bc.queue = bc.queue[1:]
	}

	item := &TransmissionItem{
		Type:      itemType,
		Data:      data,
		Timestamp: time.Now(),
		Retries:   0,
	}

	bc.queue = append(bc.queue, item)
	return nil
}

// DequeueNext returns the next item from the queue without removing it
func (bc *BaseConnector) DequeueNext() *TransmissionItem {
	bc.queueMu.Lock()
	defer bc.queueMu.Unlock()

	if len(bc.queue) == 0 {
		return nil
	}

	return bc.queue[0]
}

// RemoveFromQueue removes the first item from the queue
func (bc *BaseConnector) RemoveFromQueue() {
	bc.queueMu.Lock()
	defer bc.queueMu.Unlock()

	if len(bc.queue) > 0 {
		bc.queue = bc.queue[1:]
	}
}

// IncrementRetries increments the retry count for the first item in the queue
func (bc *BaseConnector) IncrementRetries() {
	bc.queueMu.Lock()
	defer bc.queueMu.Unlock()

	if len(bc.queue) > 0 {
		bc.queue[0].Retries++
	}
}

// GetQueueSize returns the current queue size
func (bc *BaseConnector) GetQueueSize() int {
	bc.queueMu.Lock()
	defer bc.queueMu.Unlock()
	return len(bc.queue)
}

// StartTransmissionLoop starts the periodic transmission goroutine
func (bc *BaseConnector) StartTransmissionLoop(connector CloudConnector) {
	bc.transmitTicker = time.NewTicker(bc.transmitInterval)
	bc.wg.Add(1)

	go func() {
		defer bc.wg.Done()
		log.Printf("[CloudConnector] Starting transmission loop with interval %v", bc.transmitInterval)

		for {
			select {
			case <-bc.ctx.Done():
				return
			case <-bc.transmitTicker.C:
				bc.processQueue(connector)
			}
		}
	}()
}

// StopTransmissionLoop stops the transmission goroutine
func (bc *BaseConnector) StopTransmissionLoop() {
	if bc.transmitTicker != nil {
		bc.transmitTicker.Stop()
	}
	bc.cancel()
	bc.wg.Wait()
}

// processQueue processes items in the transmission queue
func (bc *BaseConnector) processQueue(connector CloudConnector) {
	if !connector.IsConnected() {
		log.Printf("[CloudConnector] Not connected, skipping transmission")
		return
	}

	for {
		item := bc.DequeueNext()
		if item == nil {
			break
		}

		var err error
		switch item.Type {
		case "profile":
			if profile, ok := item.Data.(*database.BehavioralProfile); ok {
				err = connector.SendProfile(profile)
			} else {
				log.Printf("[CloudConnector] Invalid profile data type")
				bc.RemoveFromQueue()
				continue
			}
		case "device":
			if device, ok := item.Data.(*database.Device); ok {
				err = connector.SendDevice(device)
			} else {
				log.Printf("[CloudConnector] Invalid device data type")
				bc.RemoveFromQueue()
				continue
			}
		default:
			log.Printf("[CloudConnector] Unknown item type: %s", item.Type)
			bc.RemoveFromQueue()
			continue
		}

		if err != nil {
			log.Printf("[CloudConnector] Failed to transmit %s: %v", item.Type, err)
			bc.IncrementRetries()

			// Check if max retries exceeded
			if item.Retries >= bc.maxRetries {
				log.Printf("[CloudConnector] Max retries exceeded for %s, dropping item", item.Type)
				bc.RemoveFromQueue()
			} else {
				// Exponential backoff
				backoff := bc.retryDelay * time.Duration(1<<uint(item.Retries))
				log.Printf("[CloudConnector] Will retry in %v (attempt %d/%d)", backoff, item.Retries+1, bc.maxRetries)
				time.Sleep(backoff)
			}
		} else {
			log.Printf("[CloudConnector] Successfully transmitted %s", item.Type)
			bc.RemoveFromQueue()
		}
	}
}

// SerializeProfile converts a BehavioralProfile to JSON
func SerializeProfile(profile *database.BehavioralProfile) ([]byte, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}
	return json.Marshal(profile)
}

// SerializeDevice converts a Device to JSON
func SerializeDevice(device *database.Device) ([]byte, error) {
	if device == nil {
		return nil, fmt.Errorf("device is nil")
	}
	return json.Marshal(device)
}

// NewConnector creates a cloud connector based on the configuration
// This factory function instantiates the correct connector based on the provider
func NewConnector(cfg *config.CloudConfig, db *database.DatabaseManager) (CloudConnector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cloud configuration is required")
	}

	if !cfg.Enabled {
		return nil, fmt.Errorf("cloud connector is disabled in configuration")
	}

	// Note: Import cycles prevent direct instantiation here
	// The orchestrator should use the provider-specific constructors directly
	// This function serves as documentation of the factory pattern
	return nil, fmt.Errorf("use provider-specific constructors (aws.NewAWSIoTConnector or gcp.NewGoogleCloudConnector)")
}
