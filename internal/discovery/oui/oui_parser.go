// Package oui provides OUI (Organizationally Unique Identifier) lookup functionality
// for identifying device manufacturers from MAC addresses.
//
// The package uses the IEEE OUI database embedded in the binary for offline lookups.
// OUI is the first 24 bits (3 bytes) of a MAC address that identifies the manufacturer.
package oui

import (
	"bufio"
	"fmt"
	"strings"
)

// OUIEntry represents a single OUI database entry
type OUIEntry struct {
	Prefix       string // MAC prefix (e.g., "00:1A:2B" or "001A2B")
	ShortName    string // Short vendor name
	LongName     string // Full organization name
	AddressLines []string
}

// ParseOUIDatabase parses the IEEE OUI database format
// Format examples:
//
//	B8-7C-F2   (hex)		Extreme Networks Headquarters
//	B87CF2     (base 16)	Extreme Networks Headquarters
func ParseOUIDatabase(data string) (map[string]*OUIEntry, error) {
	entries := make(map[string]*OUIEntry)
	scanner := bufio.NewScanner(strings.NewReader(data))

	var currentEntry *OUIEntry
	var currentPrefix string

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if this is a new OUI entry (hex format)
		if strings.Contains(line, "(hex)") {
			parts := strings.SplitN(line, "(hex)", 2)
			if len(parts) != 2 {
				continue
			}

			// Extract MAC prefix (format: "B8-7C-F2" or "B87CF2")
			prefix := strings.TrimSpace(parts[0])
			prefix = strings.ReplaceAll(prefix, "-", "")
			prefix = strings.ReplaceAll(prefix, ":", "")
			prefix = strings.ToUpper(prefix)

			// Extract organization name
			orgName := strings.TrimSpace(parts[1])

			// Create new entry
			currentEntry = &OUIEntry{
				Prefix:       prefix,
				ShortName:    orgName,
				LongName:     orgName,
				AddressLines: make([]string, 0),
			}
			currentPrefix = prefix

			// Store in map (use first 6 hex chars as key)
			if len(prefix) >= 6 {
				key := prefix[:6]
				entries[key] = currentEntry
			}

			continue
		}

		// Check if this is a base 16 entry (full organization name)
		if strings.Contains(line, "(base 16)") {
			parts := strings.SplitN(line, "(base 16)", 2)
			if len(parts) == 2 && currentEntry != nil {
				longName := strings.TrimSpace(parts[1])
				if longName != "" {
					currentEntry.LongName = longName
				}
			}
			continue
		}

		// Otherwise, it's an address line for the current entry
		if currentEntry != nil && currentPrefix != "" {
			// Add address line if it's not empty and not a country code
			if len(line) > 2 {
				currentEntry.AddressLines = append(currentEntry.AddressLines, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading OUI database: %w", err)
	}

	return entries, nil
}

// NormalizeMAC normalizes a MAC address to the format used in OUI lookup
// Accepts formats: "00:1A:2B:3C:4D:5E", "00-1A-2B-3C-4D-5E", "001A2B3C4D5E"
// Returns: "001A2B" (first 6 hex chars, uppercase)
func NormalizeMAC(mac string) string {
	// Remove common separators
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	mac = strings.ToUpper(mac)

	// Return first 6 characters (OUI prefix)
	if len(mac) >= 6 {
		return mac[:6]
	}

	return mac
}
