package platform

import (
	"net"
	"time"
)

// PacketCaptureProvider abstracts packet capture mechanisms across platforms
type PacketCaptureProvider interface {
	// Open initializes packet capture on the specified interface
	Open(interfaceName string, promiscuous bool, filter string) error

	// ReadPacket returns the next captured packet
	// Returns nil packet when no more packets are available
	ReadPacket() (*Packet, error)

	// Close releases packet capture resources
	Close() error

	// GetStats returns capture statistics (packets captured, dropped, etc.)
	GetStats() (*CaptureStats, error)
}

// Packet represents a captured network packet with parsed metadata
type Packet struct {
	Timestamp   time.Time
	SrcMAC      net.HardwareAddr
	DstMAC      net.HardwareAddr
	SrcIP       net.IP
	DstIP       net.IP
	SrcPort     uint16
	DstPort     uint16
	Protocol    string
	PayloadSize uint32
	RawData     []byte // Optional: full packet data
}

// CaptureStats contains packet capture statistics
type CaptureStats struct {
	PacketsCaptured uint64
	PacketsDropped  uint64
	PacketsFiltered uint64
}

// SystemIntegrator abstracts OS-level service integration
type SystemIntegrator interface {
	// Install registers the application with the OS (service/daemon/LaunchAgent)
	Install(config *InstallConfig) error

	// Uninstall removes the application from OS registration
	Uninstall() error

	// Start begins the service/daemon
	Start() error

	// Stop halts the service/daemon
	Stop() error

	// Restart stops and starts the service/daemon
	Restart() error

	// GetStatus returns the current service status
	GetStatus() (*ServiceStatus, error)

	// EnableAutoStart configures the service to start on boot
	EnableAutoStart(enabled bool) error
}

// InstallConfig contains installation parameters
type InstallConfig struct {
	ServiceName    string
	DisplayName    string
	Description    string
	ExecutablePath string
	Arguments      []string
	WorkingDir     string
	User           string // Optional: run as specific user
}

// ServiceStatus represents the current state of the service
type ServiceStatus struct {
	IsRunning   bool
	IsInstalled bool
	AutoStart   bool
	PID         int
	Uptime      time.Duration
}

// StorageProvider abstracts data persistence across platforms
type StorageProvider interface {
	// Open initializes the storage backend
	Open(path string, options *StorageOptions) error

	// Close releases storage resources
	Close() error

	// Get retrieves a value by key
	Get(key string) ([]byte, error)

	// Set stores a value with the given key
	Set(key string, value []byte) error

	// Delete removes a key-value pair
	Delete(key string) error

	// List returns all keys matching the prefix
	List(prefix string) ([]string, error)

	// Batch performs multiple operations atomically
	Batch(ops []BatchOp) error
}

// StorageOptions contains storage configuration
type StorageOptions struct {
	ReadOnly   bool
	SyncWrites bool
	CacheSize  int64
}

// BatchOp represents a single operation in a batch
type BatchOp struct {
	Type  BatchOpType
	Key   string
	Value []byte
}

// BatchOpType defines the type of batch operation
type BatchOpType int

const (
	// BatchOpSet represents a set operation
	BatchOpSet BatchOpType = iota
	// BatchOpDelete represents a delete operation
	BatchOpDelete
)
