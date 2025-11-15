package netconfig

import (
	"net"
	"testing"
)

func TestNewAutoConfig(t *testing.T) {
	ac := NewAutoConfig()
	if ac == nil {
		t.Fatal("NewAutoConfig returned nil")
	}

	// Initially, config should be nil
	if ac.GetConfig() != nil {
		t.Error("Expected initial config to be nil")
	}
}

func TestGetInterfaceIP(t *testing.T) {
	ac := NewAutoConfig()

	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Failed to get interfaces: %v", err)
	}

	// Find a valid interface with an IP
	var validInterface net.Interface
	found := false
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			ip, err := ac.getInterfaceIP(iface)
			if err == nil && ip != nil {
				validInterface = iface
				found = true
				break
			}
		}
	}

	if !found {
		t.Skip("No valid network interface found for testing")
	}

	ip, err := ac.getInterfaceIP(validInterface)
	if err != nil {
		t.Errorf("Failed to get IP for interface %s: %v", validInterface.Name, err)
	}

	if ip == nil {
		t.Error("Expected non-nil IP address")
	}

	if ip.To4() == nil {
		t.Error("Expected IPv4 address")
	}
}

func TestThreadSafeAccess(t *testing.T) {
	ac := NewAutoConfig()

	// Set a mock config
	ac.mu.Lock()
	ac.config = &NetworkConfig{
		Interface: "eth0",
		LocalIP:   net.ParseIP("192.168.1.100"),
		Gateway:   net.ParseIP("192.168.1.1"),
		CIDR:      "192.168.1.0/24",
	}
	ac.mu.Unlock()

	// Test thread-safe getters
	if ac.GetInterface() != "eth0" {
		t.Errorf("Expected interface eth0, got %s", ac.GetInterface())
	}

	if !ac.GetLocalIP().Equal(net.ParseIP("192.168.1.100")) {
		t.Errorf("Expected local IP 192.168.1.100, got %s", ac.GetLocalIP())
	}

	if !ac.GetGatewayIP().Equal(net.ParseIP("192.168.1.1")) {
		t.Errorf("Expected gateway 192.168.1.1, got %s", ac.GetGatewayIP())
	}

	if ac.GetCIDR() != "192.168.1.0/24" {
		t.Errorf("Expected CIDR 192.168.1.0/24, got %s", ac.GetCIDR())
	}

	// Test GetConfig returns a copy
	config := ac.GetConfig()
	if config == nil {
		t.Fatal("GetConfig returned nil")
	}

	if config.Interface != "eth0" {
		t.Errorf("Expected interface eth0, got %s", config.Interface)
	}
}

func TestFindPrimaryInterface(t *testing.T) {
	ac := NewAutoConfig()

	iface, ip, err := ac.findPrimaryInterface()
	if err != nil {
		t.Skipf("No primary interface found (this is expected in some environments): %v", err)
	}

	if iface == "" {
		t.Error("Expected non-empty interface name")
	}

	if ip == nil {
		t.Error("Expected non-nil IP address")
	}

	if ip.To4() == nil {
		t.Error("Expected IPv4 address")
	}

	t.Logf("Found primary interface: %s with IP: %s", iface, ip)
}
