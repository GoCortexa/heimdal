package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

// scanMDNS performs mDNS/DNS-SD discovery
func (s *Scanner) scanMDNS() {
	netConfig := s.netConfig.GetConfig()
	if netConfig == nil {
		log.Println("Network configuration not available, skipping mDNS scan")
		return
	}

	log.Println("Starting mDNS discovery scan")

	// Create a channel to receive mDNS entries
	entriesCh := make(chan *mdns.ServiceEntry, 100)
	
	// Create context with timeout for the scan
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	// Start goroutine to process entries
	done := make(chan struct{})
	deviceCount := 0
	
	go func() {
		defer close(done)
		for entry := range entriesCh {
			s.processMDNSEntry(entry)
			deviceCount++
		}
	}()

	// Perform mDNS query for common service types
	serviceTypes := []string{
		"_workstation._tcp",  // Workstations
		"_device-info._tcp",  // Device info
		"_http._tcp",         // HTTP services
		"_ssh._tcp",          // SSH services
		"_smb._tcp",          // SMB/Samba
		"_airplay._tcp",      // AirPlay devices
		"_googlecast._tcp",   // Chromecast devices
		"_hap._tcp",          // HomeKit devices
		"_homekit._tcp",      // HomeKit devices (alternate)
		"_printer._tcp",      // Printers
		"_ipp._tcp",          // Internet Printing Protocol
		"_scanner._tcp",      // Scanners
		"_raop._tcp",         // Remote Audio Output Protocol (AirPlay)
	}

	// Query each service type
	for _, serviceType := range serviceTypes {
		select {
		case <-ctx.Done():
			break
		default:
		}

		if err := s.queryMDNSService(ctx, serviceType, entriesCh); err != nil {
			log.Printf("Error querying mDNS service %s: %v", serviceType, err)
		}
	}

	// Close entries channel and wait for processing to complete
	close(entriesCh)
	<-done

	log.Printf("mDNS scan completed: discovered %d service entries", deviceCount)
}

// queryMDNSService queries a specific mDNS service type
func (s *Scanner) queryMDNSService(ctx context.Context, serviceType string, entriesCh chan<- *mdns.ServiceEntry) error {
	// Set up mDNS query parameters
	params := &mdns.QueryParam{
		Service:             serviceType,
		Domain:              "local",
		Timeout:             2 * time.Second,
		Entries:             entriesCh,
		WantUnicastResponse: false,
	}

	// Execute the query
	if err := mdns.Query(params); err != nil {
		return fmt.Errorf("mDNS query failed: %w", err)
	}

	return nil
}

// processMDNSEntry processes a single mDNS service entry
func (s *Scanner) processMDNSEntry(entry *mdns.ServiceEntry) {
	if entry == nil {
		return
	}

	// Extract device information
	var ip string
	var mac string
	name := entry.Name

	// Get IP address (prefer IPv4)
	if entry.AddrV4 != nil {
		ip = entry.AddrV4.String()
	} else if entry.AddrV6 != nil {
		ip = entry.AddrV6.String()
	} else {
		// No IP address available
		return
	}

	// Try to resolve MAC address from IP using ARP cache
	mac = s.getMACFromIP(ip)
	if mac == "" {
		// If we can't get MAC, we can't uniquely identify the device
		log.Printf("mDNS: Found device %s at %s but couldn't resolve MAC address", name, ip)
		return
	}

	// Clean up the name (remove service type suffix)
	name = s.cleanMDNSName(name)

	// Update device with mDNS information
	log.Printf("mDNS: Discovered device %s (%s) at %s", name, mac, ip)
	s.updateDevice(mac, ip, name, "")
}

// getMACFromIP attempts to get MAC address from IP using ARP cache
func (s *Scanner) getMACFromIP(ip string) string {
	// First check our internal device map
	s.devicesMu.RLock()
	for mac, device := range s.devices {
		if device.IP == ip {
			s.devicesMu.RUnlock()
			return mac
		}
	}
	s.devicesMu.RUnlock()

	// Try to read from system ARP cache
	mac, err := s.readARPCache(ip)
	if err == nil && mac != "" {
		return mac
	}

	return ""
}

// readARPCache reads the system ARP cache to find MAC for an IP
func (s *Scanner) readARPCache(ip string) (string, error) {
	// Get ARP table entries
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Try to find the IP in the ARP cache by checking network interfaces
	// This is a simplified approach - in production, you might want to parse /proc/net/arp
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Check if the IP is in the same subnet
			if ipNet.Contains(net.ParseIP(ip)) {
				// For now, we'll rely on our internal device map
				// A more robust implementation would parse /proc/net/arp
				return "", fmt.Errorf("MAC not found in cache")
			}
		}
	}

	return "", fmt.Errorf("IP not found in ARP cache")
}

// cleanMDNSName cleans up mDNS service names
func (s *Scanner) cleanMDNSName(name string) string {
	// Remove domain suffix (.local)
	name = strings.TrimSuffix(name, ".local")
	name = strings.TrimSuffix(name, ".local.")

	// Remove service type suffixes
	serviceTypes := []string{
		"._workstation._tcp",
		"._device-info._tcp",
		"._http._tcp",
		"._ssh._tcp",
		"._smb._tcp",
		"._airplay._tcp",
		"._googlecast._tcp",
		"._hap._tcp",
		"._homekit._tcp",
		"._printer._tcp",
		"._ipp._tcp",
		"._scanner._tcp",
		"._raop._tcp",
	}

	for _, suffix := range serviceTypes {
		name = strings.TrimSuffix(name, suffix)
	}

	// Trim any remaining dots
	name = strings.Trim(name, ".")

	return name
}

// startMDNSListener starts a passive mDNS listener (for future enhancement)
// This would listen for mDNS announcements without actively querying
func (s *Scanner) startMDNSListener() error {
	// This is a placeholder for passive mDNS listening
	// The hashicorp/mdns library primarily supports active queries
	// For passive listening, you might need to use a different library
	// or implement custom multicast UDP listening on 224.0.0.251:5353
	
	log.Println("Passive mDNS listening not yet implemented")
	return nil
}
