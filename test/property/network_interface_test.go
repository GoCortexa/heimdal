// +build property

package property

import (
	"net"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/installer"
)

// Feature: monorepo-architecture, Property 15: Network Interface Auto-Detection
// Validates: Requirements 11.5
//
// For any system with at least one active network interface, the Desktop product
// should successfully detect and select a primary interface.
func TestProperty_NetworkInterfaceAutoDetection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Network interface detection returns at least one interface on systems with active interfaces", prop.ForAll(
		func() bool {
			// Detect network interfaces
			interfaces, err := installer.DetectNetworkInterfaces()
			if err != nil {
				t.Logf("Failed to detect network interfaces: %v", err)
				return false
			}

			// Get all system interfaces to check if any are active
			systemInterfaces, err := net.Interfaces()
			if err != nil {
				t.Logf("Failed to get system interfaces: %v", err)
				return false
			}

			// Count active non-loopback interfaces with addresses
			activeCount := 0
			for _, iface := range systemInterfaces {
				// Skip loopback
				if iface.Flags&net.FlagLoopback != 0 {
					continue
				}
				// Skip down interfaces
				if iface.Flags&net.FlagUp == 0 {
					continue
				}

				// Check if it has addresses
				addrs, err := iface.Addrs()
				if err != nil || len(addrs) == 0 {
					continue
				}

				// Check for valid non-link-local addresses
				hasValidAddr := false
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok {
						if !ipnet.IP.IsLinkLocalUnicast() {
							hasValidAddr = true
							break
						}
					}
				}

				if hasValidAddr {
					activeCount++
				}
			}

			// Property: If there are active interfaces, detection should find at least one
			if activeCount > 0 {
				if len(interfaces) == 0 {
					t.Logf("System has %d active interfaces but detection found none", activeCount)
					return false
				}

				// Verify detected interfaces are valid
				for i, iface := range interfaces {
					// Check required fields are present
					if iface.Name == "" {
						t.Logf("Interface %d has empty name", i)
						return false
					}

					if len(iface.Addrs) == 0 {
						t.Logf("Interface %d (%s) has no addresses", i, iface.Name)
						return false
					}

					// Verify interface is actually up
					if !iface.IsUp {
						t.Logf("Interface %d (%s) is marked as down but was detected", i, iface.Name)
						return false
					}

					// Verify addresses are valid IPs
					for j, addr := range iface.Addrs {
						ip := net.ParseIP(addr)
						if ip == nil {
							t.Logf("Interface %d (%s) has invalid IP address at index %d: %s", i, iface.Name, j, addr)
							return false
						}
					}
				}

				// Property: At most one interface should be marked as default
				defaultCount := 0
				for _, iface := range interfaces {
					if iface.IsDefault {
						defaultCount++
					}
				}

				if defaultCount > 1 {
					t.Logf("Multiple interfaces marked as default: %d", defaultCount)
					return false
				}

				return true
			}

			// If no active interfaces, detection should return empty list (not an error)
			return len(interfaces) == 0
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_NetworkInterfaceDetectionConsistency verifies that repeated calls
// to DetectNetworkInterfaces return consistent results
func TestProperty_NetworkInterfaceDetectionConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Network interface detection is consistent across multiple calls", prop.ForAll(
		func() bool {
			// Detect interfaces twice
			interfaces1, err1 := installer.DetectNetworkInterfaces()
			if err1 != nil {
				t.Logf("First detection failed: %v", err1)
				return false
			}

			interfaces2, err2 := installer.DetectNetworkInterfaces()
			if err2 != nil {
				t.Logf("Second detection failed: %v", err2)
				return false
			}

			// Property: Should return same number of interfaces
			if len(interfaces1) != len(interfaces2) {
				t.Logf("Inconsistent interface count: %d vs %d", len(interfaces1), len(interfaces2))
				return false
			}

			// Property: Interface names should be the same (order may differ)
			names1 := make(map[string]bool)
			for _, iface := range interfaces1 {
				names1[iface.Name] = true
			}

			names2 := make(map[string]bool)
			for _, iface := range interfaces2 {
				names2[iface.Name] = true
			}

			// Check all names from first call exist in second call
			for name := range names1 {
				if !names2[name] {
					t.Logf("Interface %s found in first call but not second", name)
					return false
				}
			}

			// Check all names from second call exist in first call
			for name := range names2 {
				if !names1[name] {
					t.Logf("Interface %s found in second call but not first", name)
					return false
				}
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_NetworkInterfaceFieldValidity verifies that all detected interfaces
// have valid field values
func TestProperty_NetworkInterfaceFieldValidity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("All detected network interfaces have valid field values", prop.ForAll(
		func() bool {
			interfaces, err := installer.DetectNetworkInterfaces()
			if err != nil {
				t.Logf("Failed to detect interfaces: %v", err)
				return false
			}

			for i, iface := range interfaces {
				// Property: Name must be non-empty
				if iface.Name == "" {
					t.Logf("Interface %d has empty name", i)
					return false
				}

				// Property: HardwareAddr should be valid MAC format (if present)
				if iface.HardwareAddr != "" {
					_, err := net.ParseMAC(iface.HardwareAddr)
					if err != nil {
						t.Logf("Interface %d (%s) has invalid MAC address: %s", i, iface.Name, iface.HardwareAddr)
						return false
					}
				}

				// Property: Must have at least one address
				if len(iface.Addrs) == 0 {
					t.Logf("Interface %d (%s) has no addresses", i, iface.Name)
					return false
				}

				// Property: All addresses must be valid IPs
				for j, addr := range iface.Addrs {
					ip := net.ParseIP(addr)
					if ip == nil {
						t.Logf("Interface %d (%s) has invalid IP at index %d: %s", i, iface.Name, j, addr)
						return false
					}

					// Property: Addresses should not be link-local (we filter those out)
					if ip.IsLinkLocalUnicast() {
						t.Logf("Interface %d (%s) has link-local address: %s", i, iface.Name, addr)
						return false
					}
				}

				// Property: IsUp should be true (we only detect up interfaces)
				if !iface.IsUp {
					t.Logf("Interface %d (%s) is marked as down", i, iface.Name)
					return false
				}
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_DefaultInterfaceSelection verifies that if a default interface
// is detected, it's a valid interface that exists in the system
func TestProperty_DefaultInterfaceSelection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Default interface selection is valid", prop.ForAll(
		func() bool {
			interfaces, err := installer.DetectNetworkInterfaces()
			if err != nil {
				t.Logf("Failed to detect interfaces: %v", err)
				return false
			}

			// Find default interface
			var defaultIface *installer.NetworkInterfaceInfo
			for i := range interfaces {
				if interfaces[i].IsDefault {
					defaultIface = &interfaces[i]
					break
				}
			}

			// If no default interface, that's acceptable (property still holds)
			if defaultIface == nil {
				return true
			}

			// Property: Default interface must exist in system interfaces
			systemInterfaces, err := net.Interfaces()
			if err != nil {
				t.Logf("Failed to get system interfaces: %v", err)
				return false
			}

			found := false
			for _, sysIface := range systemInterfaces {
				if sysIface.Name == defaultIface.Name {
					found = true

					// Property: Default interface must be up
					if sysIface.Flags&net.FlagUp == 0 {
						t.Logf("Default interface %s is not up", defaultIface.Name)
						return false
					}

					// Property: Default interface must not be loopback
					if sysIface.Flags&net.FlagLoopback != 0 {
						t.Logf("Default interface %s is loopback", defaultIface.Name)
						return false
					}

					break
				}
			}

			if !found {
				t.Logf("Default interface %s not found in system interfaces", defaultIface.Name)
				return false
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
