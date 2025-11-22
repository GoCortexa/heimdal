//go:build linux
// +build linux

package desktop_linux

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// LinuxStorage implements StorageProvider for Linux using BadgerDB
type LinuxStorage struct {
	db *badger.DB
}

// NewLinuxStorage creates a new Linux storage provider
func NewLinuxStorage() *LinuxStorage {
	return &LinuxStorage{}
}

// GetDefaultStoragePath returns the default storage path for Linux (~/.local/share/heimdal/db)
func GetDefaultStoragePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	dbPath := filepath.Join(homeDir, ".local", "share", "heimdal", "db")
	return dbPath, nil
}

// Open initializes the storage backend
func (l *LinuxStorage) Open(path string, options *platform.StorageOptions) error {
	// If path is empty, use default path
	if path == "" {
		defaultPath, err := GetDefaultStoragePath()
		if err != nil {
			return fmt.Errorf("failed to get default storage path: %w", err)
		}
		path = defaultPath
	}

	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Configure BadgerDB options
	opts := badger.DefaultOptions(path)

	if options != nil {
		opts = opts.WithReadOnly(options.ReadOnly)
		opts = opts.WithSyncWrites(options.SyncWrites)

		if options.CacheSize > 0 {
			opts = opts.WithBlockCacheSize(options.CacheSize)
		}
	}

	// Set Linux-specific options
	opts = opts.WithLoggingLevel(badger.WARNING)

	// Open the database
	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	l.db = db
	return nil
}

// Close releases storage resources
func (l *LinuxStorage) Close() error {
	if l.db == nil {
		return nil
	}

	err := l.db.Close()
	l.db = nil
	return err
}

// Get retrieves a value by key
func (l *LinuxStorage) Get(key string) ([]byte, error) {
	if l.db == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	var value []byte
	err := l.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		value, err = item.ValueCopy(nil)
		return err
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return value, err
}

// Set stores a value with the given key
func (l *LinuxStorage) Set(key string, value []byte) error {
	if l.db == nil {
		return fmt.Errorf("storage not initialized")
	}

	return l.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// Delete removes a key-value pair
func (l *LinuxStorage) Delete(key string) error {
	if l.db == nil {
		return fmt.Errorf("storage not initialized")
	}

	return l.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// List returns all keys matching the prefix
func (l *LinuxStorage) List(prefix string) ([]string, error) {
	if l.db == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	var keys []string
	prefixBytes := []byte(prefix)

	err := l.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // We only need keys
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())
			keys = append(keys, key)
		}

		return nil
	})

	return keys, err
}

// Batch performs multiple operations atomically
func (l *LinuxStorage) Batch(ops []platform.BatchOp) error {
	if l.db == nil {
		return fmt.Errorf("storage not initialized")
	}

	return l.db.Update(func(txn *badger.Txn) error {
		for _, op := range ops {
			switch op.Type {
			case platform.BatchOpSet:
				if err := txn.Set([]byte(op.Key), op.Value); err != nil {
					return fmt.Errorf("batch set failed for key %s: %w", op.Key, err)
				}
			case platform.BatchOpDelete:
				if err := txn.Delete([]byte(op.Key)); err != nil {
					return fmt.Errorf("batch delete failed for key %s: %w", op.Key, err)
				}
			default:
				return fmt.Errorf("unknown batch operation type: %v", op.Type)
			}
		}
		return nil
	})
}

// RunGarbageCollection runs BadgerDB garbage collection
// This should be called periodically to reclaim disk space
func (l *LinuxStorage) RunGarbageCollection() error {
	if l.db == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Run GC with discard ratio of 0.5
	// This means GC will run if at least 50% of the file can be reclaimed
	err := l.db.RunValueLogGC(0.5)
	if err != nil && err != badger.ErrNoRewrite {
		return fmt.Errorf("garbage collection failed: %w", err)
	}

	return nil
}

// GetDatabaseSize returns the approximate size of the database in bytes
func (l *LinuxStorage) GetDatabaseSize() (int64, error) {
	if l.db == nil {
		return 0, fmt.Errorf("storage not initialized")
	}

	lsm, vlog := l.db.Size()
	return lsm + vlog, nil
}
