package oui

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed data/manuf
var ouiDatabaseContent string

// OUILookup provides fast OUI vendor lookups
type OUILookup struct {
	entries map[string]*OUIEntry
	mu      sync.RWMutex
	loaded  bool
}

// NewOUILookup creates a new OUI lookup instance
func NewOUILookup() *OUILookup {
	return &OUILookup{
		entries: make(map[string]*OUIEntry),
		loaded:  false,
	}
}

// Load parses and loads the embedded OUI database into memory
func (o *OUILookup) Load() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.loaded {
		return nil
	}

	entries, err := ParseOUIDatabase(ouiDatabaseContent)
	if err != nil {
		return fmt.Errorf("failed to parse OUI database: %w", err)
	}

	o.entries = entries
	o.loaded = true

	return nil
}

// Lookup finds the vendor information for a given MAC address
// Returns vendor name, manufacturer name, and whether found
func (o *OUILookup) Lookup(mac string) (vendor string, manufacturer string, found bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.loaded {
		return "", "", false
	}

	// Normalize MAC address to OUI prefix format
	prefix := NormalizeMAC(mac)
	if len(prefix) < 6 {
		return "", "", false
	}

	// Look up in database
	entry, exists := o.entries[prefix]
	if !exists {
		return "", "", false
	}

	return entry.ShortName, entry.LongName, true
}

// LookupVendor returns just the vendor name (short name)
func (o *OUILookup) LookupVendor(mac string) string {
	vendor, _, found := o.Lookup(mac)
	if !found {
		return ""
	}
	return vendor
}

// LookupManufacturer returns the full manufacturer name (long name)
func (o *OUILookup) LookupManufacturer(mac string) string {
	_, manufacturer, found := o.Lookup(mac)
	if !found {
		return ""
	}
	return manufacturer
}

// GetStats returns statistics about the loaded database
func (o *OUILookup) GetStats() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return map[string]interface{}{
		"loaded":       o.loaded,
		"entry_count":  len(o.entries),
		"memory_bytes": len(ouiDatabaseContent),
	}
}

// IsLoaded returns whether the database has been loaded
func (o *OUILookup) IsLoaded() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.loaded
}

// SearchByVendor searches for OUI entries matching a vendor name (case-insensitive)
// Returns up to maxResults entries
func (o *OUILookup) SearchByVendor(vendorName string, maxResults int) []*OUIEntry {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.loaded {
		return nil
	}

	vendorName = strings.ToLower(vendorName)
	results := make([]*OUIEntry, 0, maxResults)

	for _, entry := range o.entries {
		if len(results) >= maxResults {
			break
		}

		if strings.Contains(strings.ToLower(entry.ShortName), vendorName) ||
			strings.Contains(strings.ToLower(entry.LongName), vendorName) {
			results = append(results, entry)
		}
	}

	return results
}

// GetAllVendors returns a list of all unique vendor names
func (o *OUILookup) GetAllVendors() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.loaded {
		return nil
	}

	vendorSet := make(map[string]bool)
	for _, entry := range o.entries {
		vendorSet[entry.ShortName] = true
	}

	vendors := make([]string, 0, len(vendorSet))
	for vendor := range vendorSet {
		vendors = append(vendors, vendor)
	}

	return vendors
}
