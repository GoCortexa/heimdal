package oui

import (
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"00:1A:2B:3C:4D:5E", "001A2B"},
		{"00-1A-2B-3C-4D-5E", "001A2B"},
		{"001A2B3C4D5E", "001A2B"},
		{"00:1a:2b:3c:4d:5e", "001A2B"},
		{"B8-7C-F2-11-22-33", "B87CF2"},
		{"b87cf2112233", "B87CF2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeMAC(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeMAC(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseOUIDatabase(t *testing.T) {
	sampleData := `
OUI/MA-L                                                    Organization                                 
company_id                                                  Organization                                 
                                                            Address                                      

B8-7C-F2   (hex)		Extreme Networks Headquarters
B87CF2     (base 16)		Extreme Networks Headquarters
				2121 RDU Center Drive 
				Morrisville  NC  27560
				US

00-1A-2B   (hex)		Apple Inc
001A2B     (base 16)		Apple, Inc.
				1 Infinite Loop
				Cupertino  CA  95014
				US
`

	entries, err := ParseOUIDatabase(sampleData)
	if err != nil {
		t.Fatalf("ParseOUIDatabase failed: %v", err)
	}

	// Check that we parsed entries
	if len(entries) == 0 {
		t.Fatal("Expected entries to be parsed, got 0")
	}

	// Check Extreme Networks entry
	extreme, exists := entries["B87CF2"]
	if !exists {
		t.Error("Expected B87CF2 entry to exist")
	} else {
		if extreme.ShortName != "Extreme Networks Headquarters" {
			t.Errorf("Expected short name 'Extreme Networks Headquarters', got %q", extreme.ShortName)
		}
		if extreme.LongName != "Extreme Networks Headquarters" {
			t.Errorf("Expected long name 'Extreme Networks Headquarters', got %q", extreme.LongName)
		}
	}

	// Check Apple entry
	apple, exists := entries["001A2B"]
	if !exists {
		t.Error("Expected 001A2B entry to exist")
	} else {
		if apple.ShortName != "Apple Inc" {
			t.Errorf("Expected short name 'Apple Inc', got %q", apple.ShortName)
		}
		if apple.LongName != "Apple, Inc." {
			t.Errorf("Expected long name 'Apple, Inc.', got %q", apple.LongName)
		}
	}
}

func TestOUILookup(t *testing.T) {
	lookup := NewOUILookup()

	// Load the embedded database
	if err := lookup.Load(); err != nil {
		t.Fatalf("Failed to load OUI database: %v", err)
	}

	if !lookup.IsLoaded() {
		t.Fatal("Expected database to be loaded")
	}

	// Test lookup with various MAC formats
	testCases := []struct {
		mac       string
		hasVendor bool
	}{
		{"B8:7C:F2:11:22:33", true},  // Extreme Networks
		{"00:1A:2B:3C:4D:5E", true},  // Should find something
		{"FF:FF:FF:FF:FF:FF", false}, // Unlikely to exist
	}

	for _, tc := range testCases {
		t.Run(tc.mac, func(t *testing.T) {
			vendor, manufacturer, found := lookup.Lookup(tc.mac)

			if found != tc.hasVendor {
				t.Errorf("Lookup(%q) found=%v, want %v", tc.mac, found, tc.hasVendor)
			}

			if found {
				if vendor == "" {
					t.Errorf("Lookup(%q) returned empty vendor", tc.mac)
				}
				if manufacturer == "" {
					t.Errorf("Lookup(%q) returned empty manufacturer", tc.mac)
				}
				t.Logf("Lookup(%q) = vendor:%q, manufacturer:%q", tc.mac, vendor, manufacturer)
			}
		})
	}
}

func TestOUILookupStats(t *testing.T) {
	lookup := NewOUILookup()

	if err := lookup.Load(); err != nil {
		t.Fatalf("Failed to load OUI database: %v", err)
	}

	stats := lookup.GetStats()

	if !stats["loaded"].(bool) {
		t.Error("Expected loaded to be true")
	}

	entryCount := stats["entry_count"].(int)
	if entryCount == 0 {
		t.Error("Expected entry_count > 0")
	}

	t.Logf("OUI Database Stats: %d entries, %d bytes", entryCount, stats["memory_bytes"])
}

func BenchmarkOUILookup(b *testing.B) {
	lookup := NewOUILookup()
	if err := lookup.Load(); err != nil {
		b.Fatalf("Failed to load OUI database: %v", err)
	}

	testMAC := "B8:7C:F2:11:22:33"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lookup.Lookup(testMAC)
	}
}

func BenchmarkNormalizeMAC(b *testing.B) {
	testMAC := "B8:7C:F2:11:22:33"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NormalizeMAC(testMAC)
	}
}
