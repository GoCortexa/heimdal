// Package netconfig provides automatic network configuration detection for zero-touch provisioning.
//
// The AutoConfig component automatically detects the local network configuration at startup,
// enabling the sensor to operate without manual configuration. It identifies the primary
// network interface, gateway IP, and subnet information required for device discovery and
// traffic interception.
//
// Detection Process:
//   1. Enumerate all network interfaces
//   2. Find interfaces with valid IP addresses (not loopback)
//   3. Parse /proc/net/route to determine gateway IP
//   4. Calculate subnet mask and CIDR notation
//   5. Provide thread-safe read access to configuration
//
// Auto-Detection Behavior:
//   - Searches for interfaces in order: eth0, wlan0, en0, other active interfaces
//   - Selects first interface with a valid IP address and gateway
//   - Retries every 5 seconds until a valid network is found
//   - Blocks application startup until network is detected
//
// The detected configuration is used by:
//   - Device Discovery: for ARP scanning subnet range
//   - Traffic Interceptor: for ARP spoofing operations
//   - Packet Analyzer: for capturing on the correct interface
//
// Thread Safety:
// All access to the network configuration is protected by RWMutex for safe concurrent access.
package netconfig

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/errors"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
)

// NetworkConfig holds the detected network configuration
type NetworkConfig struct {
	Interface string
	LocalIP   net.IP
	Gateway   net.IP
	Subnet    *net.IPNet
	CIDR      string
}

// AutoConfig manages network auto-detection with thread-safe access
type AutoConfig struct {
	config *NetworkConfig
	logger *logger.Logger
	mu     sync.RWMutex
}

// NewAutoConfig creates a new AutoConfig instance
func NewAutoConfig() *AutoConfig {
	return &AutoConfig{
		logger: logger.NewComponentLogger("NetConfig"),
	}
}

// DetectNetwork attempts to detect the network configuration
// It retries every 5 seconds until successful
func (ac *AutoConfig) DetectNetwork() error {
	ac.logger.Info("Starting network detection...")
	
	for {
		config, err := ac.detectNetworkOnce()
		if err == nil {
			ac.mu.Lock()
			ac.config = config
			ac.mu.Unlock()
			ac.logger.Info("Network detected successfully: interface=%s, ip=%s, gateway=%s",
				config.Interface, config.LocalIP, config.Gateway)
			return nil
		}

		// Log the error and retry after 5 seconds
		ac.logger.Warn("Failed to detect network: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}
}

// detectNetworkOnce performs a single network detection attempt
func (ac *AutoConfig) detectNetworkOnce() (*NetworkConfig, error) {
	// Find primary network interface
	iface, localIP, err := ac.findPrimaryInterface()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find primary interface")
	}

	ac.logger.Debug("Found primary interface: %s with IP %s", iface, localIP)

	// Get gateway IP
	gateway, err := ac.getGateway(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}

	// Get subnet information
	subnet, cidr, err := ac.getSubnet(iface, localIP)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnet: %w", err)
	}

	return &NetworkConfig{
		Interface: iface,
		LocalIP:   localIP,
		Gateway:   gateway,
		Subnet:    subnet,
		CIDR:      cidr,
	}, nil
}

// findPrimaryInterface finds the primary network interface (eth0, wlan0, etc.)
func (ac *AutoConfig) findPrimaryInterface() (string, net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	// Preferred interface names in order of priority
	preferredNames := []string{"eth0", "wlan0", "en0", "enp", "wlp"}

	// First pass: look for preferred interface names
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Check if this is a preferred interface
		for _, preferred := range preferredNames {
			if strings.HasPrefix(iface.Name, preferred) {
				localIP, err := ac.getInterfaceIP(iface)
				if err == nil && localIP != nil {
					return iface.Name, localIP, nil
				}
			}
		}
	}

	// Second pass: find any interface with a valid IP
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		localIP, err := ac.getInterfaceIP(iface)
		if err == nil && localIP != nil {
			return iface.Name, localIP, nil
		}
	}

	return "", nil, fmt.Errorf("no suitable network interface found")
}

// getInterfaceIP gets the IPv4 address for an interface
func (ac *AutoConfig) getInterfaceIP(iface net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		// We want IPv4 addresses only
		if ipNet.IP.To4() != nil {
			return ipNet.IP, nil
		}
	}

	return nil, fmt.Errorf("no IPv4 address found")
}

// getGateway parses /proc/net/route to find the default gateway
func (ac *AutoConfig) getGateway(iface string) (net.IP, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/net/route: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	
	// Skip header line
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty route table")
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		
		if len(fields) < 3 {
			continue
		}

		// Check if this is the default route (destination 00000000) for our interface
		if fields[0] == iface && fields[1] == "00000000" {
			// Gateway is in field 2, in hex format (little-endian)
			gatewayHex := fields[2]
			
			// Parse hex string to uint32
			var gatewayInt uint32
			_, err := fmt.Sscanf(gatewayHex, "%X", &gatewayInt)
			if err != nil {
				return nil, fmt.Errorf("failed to parse gateway hex: %w", err)
			}

			// Convert to IP (little-endian byte order)
			gateway := make(net.IP, 4)
			binary.LittleEndian.PutUint32(gateway, gatewayInt)
			
			return gateway, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading route table: %w", err)
	}

	return nil, fmt.Errorf("no default gateway found for interface %s", iface)
}

// getSubnet gets the subnet information for the interface
func (ac *AutoConfig) getSubnet(iface string, localIP net.IP) (*net.IPNet, string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list interfaces: %w", err)
	}

	for _, i := range interfaces {
		if i.Name != iface {
			continue
		}

		addrs, err := i.Addrs()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get addresses for %s: %w", iface, err)
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Find the subnet that matches our local IP
			if ipNet.IP.To4() != nil && ipNet.IP.Equal(localIP) {
				// Calculate CIDR notation
				ones, _ := ipNet.Mask.Size()
				cidr := fmt.Sprintf("%s/%d", ipNet.IP.Mask(ipNet.Mask).String(), ones)
				
				return ipNet, cidr, nil
			}
		}
	}

	return nil, "", fmt.Errorf("subnet not found for interface %s", iface)
}

// GetConfig returns a copy of the current network configuration (thread-safe)
func (ac *AutoConfig) GetConfig() *NetworkConfig {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ac.config == nil {
		return nil
	}

	// Return a copy to prevent external modification
	return &NetworkConfig{
		Interface: ac.config.Interface,
		LocalIP:   ac.config.LocalIP,
		Gateway:   ac.config.Gateway,
		Subnet:    ac.config.Subnet,
		CIDR:      ac.config.CIDR,
	}
}

// GetInterface returns the detected interface name (thread-safe)
func (ac *AutoConfig) GetInterface() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ac.config == nil {
		return ""
	}
	return ac.config.Interface
}

// GetLocalIP returns the detected local IP address (thread-safe)
func (ac *AutoConfig) GetLocalIP() net.IP {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ac.config == nil {
		return nil
	}
	return ac.config.LocalIP
}

// GetGatewayIP returns the detected gateway IP address (thread-safe)
func (ac *AutoConfig) GetGatewayIP() net.IP {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ac.config == nil {
		return nil
	}
	return ac.config.Gateway
}

// GetSubnet returns the detected subnet (thread-safe)
func (ac *AutoConfig) GetSubnet() *net.IPNet {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ac.config == nil {
		return nil
	}
	return ac.config.Subnet
}

// GetCIDR returns the CIDR notation of the subnet (thread-safe)
func (ac *AutoConfig) GetCIDR() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ac.config == nil {
		return ""
	}
	return ac.config.CIDR
}
