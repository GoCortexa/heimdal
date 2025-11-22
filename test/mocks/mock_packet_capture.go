package mocks

import (
	"io"
	"sync"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// MockPacketCaptureProvider is a mock implementation for testing packet analysis
type MockPacketCaptureProvider struct {
	mu       sync.Mutex
	packets  []*platform.Packet
	index    int
	isOpen   bool
	openErr  error
	readErr  error
	closeErr error
	stats    *platform.CaptureStats
}

// NewMockPacketCaptureProvider creates a new mock packet capture provider
func NewMockPacketCaptureProvider(packets []*platform.Packet) *MockPacketCaptureProvider {
	return &MockPacketCaptureProvider{
		packets: packets,
		index:   0,
		isOpen:  false,
		stats: &platform.CaptureStats{
			PacketsCaptured: uint64(len(packets)),
			PacketsDropped:  0,
			PacketsFiltered: 0,
		},
	}
}

// Open initializes the mock packet capture
func (m *MockPacketCaptureProvider) Open(interfaceName string, promiscuous bool, filter string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.openErr != nil {
		return m.openErr
	}

	m.isOpen = true
	m.index = 0
	return nil
}

// ReadPacket returns the next packet from the mock data
func (m *MockPacketCaptureProvider) ReadPacket() (*platform.Packet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isOpen {
		return nil, io.EOF
	}

	if m.readErr != nil {
		return nil, m.readErr
	}

	if m.index >= len(m.packets) {
		return nil, io.EOF
	}

	packet := m.packets[m.index]
	m.index++
	return packet, nil
}

// Close releases mock resources
func (m *MockPacketCaptureProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closeErr != nil {
		return m.closeErr
	}

	m.isOpen = false
	return nil
}

// GetStats returns mock capture statistics
func (m *MockPacketCaptureProvider) GetStats() (*platform.CaptureStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.stats, nil
}

// SetOpenError configures the mock to return an error on Open
func (m *MockPacketCaptureProvider) SetOpenError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.openErr = err
}

// SetReadError configures the mock to return an error on ReadPacket
func (m *MockPacketCaptureProvider) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readErr = err
}

// SetCloseError configures the mock to return an error on Close
func (m *MockPacketCaptureProvider) SetCloseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeErr = err
}

// Reset resets the mock to its initial state
func (m *MockPacketCaptureProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.index = 0
	m.isOpen = false
	m.openErr = nil
	m.readErr = nil
	m.closeErr = nil
}
