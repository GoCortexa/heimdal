package orchestrator

import (
	"encoding/json"
	"fmt"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// storageDeviceStore adapts a platform.StorageProvider to the database.DeviceStore interface.
type storageDeviceStore struct {
	storage platform.StorageProvider
}

func newStorageDeviceStore(storage platform.StorageProvider) (*storageDeviceStore, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage provider is nil")
	}
	return &storageDeviceStore{storage: storage}, nil
}

func (s *storageDeviceStore) SaveDevice(device *database.Device) error {
	if device == nil {
		return fmt.Errorf("device cannot be nil")
	}
	if device.MAC == "" {
		return fmt.Errorf("device MAC address cannot be empty")
	}

	data, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("failed to serialize device %s: %w", device.MAC, err)
	}

	key := "device:" + device.MAC
	return s.storage.Set(key, data)
}

func (s *storageDeviceStore) GetAllDevices() ([]*database.Device, error) {
	keys, err := s.storage.List("device:")
	if err != nil {
		return nil, fmt.Errorf("failed to list device keys: %w", err)
	}

	devices := make([]*database.Device, 0, len(keys))
	for _, key := range keys {
		data, err := s.storage.Get(key)
		if err != nil {
			continue
		}

		var device database.Device
		if err := json.Unmarshal(data, &device); err != nil {
			continue
		}

		devices = append(devices, &device)
	}

	return devices, nil
}
