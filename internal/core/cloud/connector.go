// Package cloud provides shared cloud communication functionality for both
// hardware and desktop products.
//
// This module handles cloud connectivity, message queuing, and transmission
// with support for multiple cloud providers (AWS IoT Core, Google Cloud Pub/Sub).
package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

// DeviceType distinguishes hardware vs desktop deployments
type DeviceType string

const (
	DeviceTypeHardware DeviceType = "hardware"
	DeviceTypeDesktop  DeviceType = "desktop"
)

// Connector defines the interface for cloud platform connectivity
type Connector interface {
	// Connect establishes connection to the cloud platform
	Connect() error

	// Disconnect closes the connection to the cloud platform
	Disconnect() error

	// SendProfile transmits a behavioral profile to the cloud
	SendProfile(profile *database.BehavioralProfile, deviceType DeviceType) error

	// SendDevice transmits device information to the cloud
	SendDevice(device *database.Device, deviceType DeviceType) error

	// SendAnomaly transmits an anomaly alert to the cloud
	SendAnomaly(anomaly *AnomalyData, deviceType DeviceType) error

	// IsConnected returns the current connection status
	IsConnected() bool
}

// AnomalyData represents an anomaly alert to be sent to the cloud
type AnomalyData struct {
	DeviceMAC   string                 `json:"device_mac"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Evidence    map[string]interface{} `json:"evidence"`
}

// ProfileData wraps a behavioral profile with device type metadata
type ProfileData struct {
	Profile    *database.BehavioralProfile `json:"profile"`
	DeviceType DeviceType                  `json:"device_type"`
	Timestamp  time.Time                   `json:"timestamp"`
}

// DeviceData wraps device information with device type metadata
type DeviceData struct {
	Device     *database.Device `json:"device"`
	DeviceType DeviceType       `json:"device_type"`
	Timestamp  time.Time        `json:"timestamp"`
}

// TransmissionItem represents an item in the transmission queue
type TransmissionItem struct {
	Type       string      // "profile", "device", or "anomaly"
	Data       interface{} // *ProfileData, *DeviceData, or *AnomalyData
	DeviceType DeviceType
	Timestamp  time.Time
	Retries    int
}

// BaseConnector provides common functionality for cloud connectors
type BaseConnector struct {
	connected        bool
	mu               sync.RWMutex
	queue            []*TransmissionItem
	queueMu          sync.Mutex
	maxQueueSize     int
	maxRetries       int
	retryDelay       time.Duration
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	transmitTicker   *time.Ticker
	transmitInterval time.Duration
}

// NewBaseConnector creates a new base connector with common functionality
func NewBaseConnector(transmitInterval time.Duration) *BaseConnector {
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
func (bc *BaseConnector) EnqueueProfile(profile *database.BehavioralProfile, deviceType DeviceType) error {
	profileData := &ProfileData{
		Profile:    profile,
		DeviceType: deviceType,
		Timestamp:  time.Now(),
	}
	return bc.enqueue("profile", profileData, deviceType)
}

// EnqueueDevice adds a device to the transmission queue
func (bc *BaseConnector) EnqueueDevice(device *database.Device, deviceType DeviceType) error {
	deviceData := &DeviceData{
		Device:     device,
		DeviceType: deviceType,
		Timestamp:  time.Now(),
	}
	return bc.enqueue("device", deviceData, deviceType)
}

// EnqueueAnomaly adds an anomaly to the transmission queue
func (bc *BaseConnector) EnqueueAnomaly(anomaly *AnomalyData, deviceType DeviceType) error {
	return bc.enqueue("anomaly", anomaly, deviceType)
}

// enqueue adds an item to the transmission queue
func (bc *BaseConnector) enqueue(itemType string, data interface{}, deviceType DeviceType) error {
	bc.queueMu.Lock()
	defer bc.queueMu.Unlock()

	// Check if queue is full
	if len(bc.queue) >= bc.maxQueueSize {
		// Remove oldest item to make room
		log.Printf("[CloudConnector] Queue full, dropping oldest item")
		bc.queue = bc.queue[1:]
	}

	item := &TransmissionItem{
		Type:       itemType,
		Data:       data,
		DeviceType: deviceType,
		Timestamp:  time.Now(),
		Retries:    0,
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
func (bc *BaseConnector) StartTransmissionLoop(connector Connector) {
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
func (bc *BaseConnector) processQueue(connector Connector) {
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
			if profileData, ok := item.Data.(*ProfileData); ok {
				err = connector.SendProfile(profileData.Profile, item.DeviceType)
			} else {
				log.Printf("[CloudConnector] Invalid profile data type")
				bc.RemoveFromQueue()
				continue
			}
		case "device":
			if deviceData, ok := item.Data.(*DeviceData); ok {
				err = connector.SendDevice(deviceData.Device, item.DeviceType)
			} else {
				log.Printf("[CloudConnector] Invalid device data type")
				bc.RemoveFromQueue()
				continue
			}
		case "anomaly":
			if anomalyData, ok := item.Data.(*AnomalyData); ok {
				err = connector.SendAnomaly(anomalyData, item.DeviceType)
			} else {
				log.Printf("[CloudConnector] Invalid anomaly data type")
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

// SerializeProfileData converts ProfileData to JSON
func SerializeProfileData(data *ProfileData) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("profile data is nil")
	}
	return json.Marshal(data)
}

// SerializeDeviceData converts DeviceData to JSON
func SerializeDeviceData(data *DeviceData) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("device data is nil")
	}
	return json.Marshal(data)
}

// SerializeAnomalyData converts AnomalyData to JSON
func SerializeAnomalyData(data *AnomalyData) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("anomaly data is nil")
	}
	return json.Marshal(data)
}
