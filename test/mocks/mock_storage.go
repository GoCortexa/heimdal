package mocks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// MockStorageProvider is a mock implementation for testing storage operations
type MockStorageProvider struct {
	mu        sync.RWMutex
	data      map[string][]byte
	isOpen    bool
	openErr   error
	closeErr  error
	getErr    error
	setErr    error
	deleteErr error
	listErr   error
	batchErr  error
	options   *platform.StorageOptions
}

// NewMockStorageProvider creates a new mock storage provider
func NewMockStorageProvider() *MockStorageProvider {
	return &MockStorageProvider{
		data:   make(map[string][]byte),
		isOpen: false,
	}
}

// Open initializes the mock storage backend
func (m *MockStorageProvider) Open(path string, options *platform.StorageOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.openErr != nil {
		return m.openErr
	}

	m.isOpen = true
	m.options = options
	return nil
}

// Close releases mock storage resources
func (m *MockStorageProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closeErr != nil {
		return m.closeErr
	}

	m.isOpen = false
	return nil
}

// Get retrieves a value by key
func (m *MockStorageProvider) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isOpen {
		return nil, &MockError{Message: "storage not open"}
	}

	if m.getErr != nil {
		return nil, m.getErr
	}

	value, exists := m.data[key]
	if !exists {
		return nil, &MockError{Message: fmt.Sprintf("key not found: %s", key)}
	}

	// Return a copy to prevent external modification
	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

// Set stores a value with the given key
func (m *MockStorageProvider) Set(key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isOpen {
		return &MockError{Message: "storage not open"}
	}

	if m.setErr != nil {
		return m.setErr
	}

	// Store a copy to prevent external modification
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	m.data[key] = valueCopy
	return nil
}

// Delete removes a key-value pair
func (m *MockStorageProvider) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isOpen {
		return &MockError{Message: "storage not open"}
	}

	if m.deleteErr != nil {
		return m.deleteErr
	}

	delete(m.data, key)
	return nil
}

// List returns all keys matching the prefix
func (m *MockStorageProvider) List(prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isOpen {
		return nil, &MockError{Message: "storage not open"}
	}

	if m.listErr != nil {
		return nil, m.listErr
	}

	var keys []string
	for key := range m.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Batch performs multiple operations atomically
func (m *MockStorageProvider) Batch(ops []platform.BatchOp) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isOpen {
		return &MockError{Message: "storage not open"}
	}

	if m.batchErr != nil {
		return m.batchErr
	}

	// In a real implementation, this would be atomic
	// For the mock, we just apply operations sequentially
	for _, op := range ops {
		switch op.Type {
		case platform.BatchOpSet:
			valueCopy := make([]byte, len(op.Value))
			copy(valueCopy, op.Value)
			m.data[op.Key] = valueCopy
		case platform.BatchOpDelete:
			delete(m.data, op.Key)
		}
	}

	return nil
}

// SetOpenError configures the mock to return an error on Open
func (m *MockStorageProvider) SetOpenError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.openErr = err
}

// SetCloseError configures the mock to return an error on Close
func (m *MockStorageProvider) SetCloseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeErr = err
}

// SetGetError configures the mock to return an error on Get
func (m *MockStorageProvider) SetGetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getErr = err
}

// SetSetError configures the mock to return an error on Set
func (m *MockStorageProvider) SetSetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setErr = err
}

// SetDeleteError configures the mock to return an error on Delete
func (m *MockStorageProvider) SetDeleteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteErr = err
}

// SetListError configures the mock to return an error on List
func (m *MockStorageProvider) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listErr = err
}

// SetBatchError configures the mock to return an error on Batch
func (m *MockStorageProvider) SetBatchError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batchErr = err
}

// GetData returns a copy of the internal data map for testing
func (m *MockStorageProvider) GetData() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dataCopy := make(map[string][]byte)
	for k, v := range m.data {
		valueCopy := make([]byte, len(v))
		copy(valueCopy, v)
		dataCopy[k] = valueCopy
	}
	return dataCopy
}

// Reset resets the mock to its initial state
func (m *MockStorageProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	m.isOpen = false
	m.openErr = nil
	m.closeErr = nil
	m.getErr = nil
	m.setErr = nil
	m.deleteErr = nil
	m.listErr = nil
	m.batchErr = nil
	m.options = nil
}
