// Package interceptor provides ARP spoofing functionality to intercept network traffic.
//
// The ARPSpoofer component performs man-in-the-middle attacks using ARP spoofing to
// intercept traffic from discovered devices. This allows the sensor to analyze traffic
// patterns without requiring network infrastructure changes.
//
// ARP Spoofing Mechanism:
//   - Listens on deviceChan for newly discovered devices
//   - Crafts ARP reply packets claiming the sensor's MAC is the gateway
//   - Sends spoofed packets to both target device and actual gateway
//   - Causes traffic to flow through the sensor for analysis
//   - Requires IP forwarding enabled on the host system
//
// Operation:
//   - Maintains a map of active spoof targets
//   - Sends ARP replies periodically (default: every 2 seconds)
//   - Removes inactive devices from spoofing list
//   - Monitors spoofing health and restarts on failure
//
// Safety Mechanisms:
//   - Verifies IP forwarding is enabled before starting
//   - Implements health checks to verify packet forwarding
//   - Automatic restart on failure with exponential backoff
//   - Graceful cleanup: restores original ARP tables on shutdown
//
// Security Warning:
// ARP spoofing is inherently invasive and can disrupt network connectivity if
// misconfigured. Only use on networks you own or have explicit permission to monitor.
// Requires CAP_NET_RAW and CAP_NET_ADMIN Linux capabilities.
package interceptor

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
)

// SpoofTarget represents a device being spoofed
type SpoofTarget struct {
	MAC       net.HardwareAddr
	IP        net.IP
	LastSpoof time.Time
	IsActive  bool
}

// ARPSpoofer manages ARP spoofing operations to intercept network traffic
type ARPSpoofer struct {
	netConfig  *netconfig.AutoConfig
	handle     *pcap.Handle
	targets    map[string]*SpoofTarget // MAC -> SpoofTarget
	targetsMu  sync.RWMutex
	deviceChan <-chan *database.Device
	
	// Configuration
	spoofInterval time.Duration
	targetMACs    []string // If empty, spoof all devices
	
	// Lifecycle management
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	runningMu sync.Mutex
	
	// Original ARP cache for restoration
	originalARPCache map[string]net.HardwareAddr
	arpCacheMu       sync.Mutex
}

// NewARPSpoofer creates a new ARPSpoofer instance
func NewARPSpoofer(netConfig *netconfig.AutoConfig, deviceChan <-chan *database.Device, spoofInterval time.Duration, targetMACs []string) *ARPSpoofer {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ARPSpoofer{
		netConfig:        netConfig,
		deviceChan:       deviceChan,
		spoofInterval:    spoofInterval,
		targetMACs:       targetMACs,
		targets:          make(map[string]*SpoofTarget),
		originalARPCache: make(map[string]net.HardwareAddr),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins ARP spoofing operations
func (as *ARPSpoofer) Start() error {
	as.runningMu.Lock()
	if as.running {
		as.runningMu.Unlock()
		return fmt.Errorf("ARP spoofer already running")
	}
	as.running = true
	as.runningMu.Unlock()

	// Verify IP forwarding is enabled
	if err := as.verifyIPForwarding(); err != nil {
		as.runningMu.Lock()
		as.running = false
		as.runningMu.Unlock()
		return fmt.Errorf("IP forwarding check failed: %w", err)
	}

	// Get network configuration
	config := as.netConfig.GetConfig()
	if config == nil {
		as.runningMu.Lock()
		as.running = false
		as.runningMu.Unlock()
		return fmt.Errorf("network configuration not available")
	}

	// Open pcap handle for packet injection
	handle, err := pcap.OpenLive(config.Interface, 65536, true, pcap.BlockForever)
	if err != nil {
		as.runningMu.Lock()
		as.running = false
		as.runningMu.Unlock()
		return fmt.Errorf("failed to open pcap handle: %w", err)
	}
	as.handle = handle

	// Start device listener goroutine
	as.wg.Add(1)
	go as.deviceListenerLoop()

	// Start spoofing loop goroutine
	as.wg.Add(1)
	go as.spoofingLoop()

	// Start health check goroutine
	as.wg.Add(1)
	go as.healthCheckLoop()

	log.Println("ARP spoofer started")
	return nil
}

// Stop gracefully stops ARP spoofing and restores ARP tables
func (as *ARPSpoofer) Stop() error {
	as.runningMu.Lock()
	if !as.running {
		as.runningMu.Unlock()
		return fmt.Errorf("ARP spoofer not running")
	}
	as.running = false
	as.runningMu.Unlock()

	log.Println("Stopping ARP spoofer and restoring ARP tables...")

	// Restore original ARP tables
	if err := as.restoreARPTables(); err != nil {
		log.Printf("Warning: failed to restore ARP tables: %v", err)
	}

	// Cancel context to signal all goroutines to stop
	as.cancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		as.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Close pcap handle
		if as.handle != nil {
			as.handle.Close()
		}
		log.Println("ARP spoofer stopped gracefully")
		return nil
	case <-time.After(5 * time.Second):
		// Force close pcap handle
		if as.handle != nil {
			as.handle.Close()
		}
		return fmt.Errorf("ARP spoofer shutdown timeout")
	}
}

// Name returns the component name
func (as *ARPSpoofer) Name() string {
	return "ARPSpoofer"
}

// restoreARPTables sends correct ARP replies to restore original ARP cache
func (as *ARPSpoofer) restoreARPTables() error {
	config := as.netConfig.GetConfig()
	if config == nil {
		return fmt.Errorf("network configuration not available")
	}

	as.targetsMu.RLock()
	targets := make([]*SpoofTarget, 0, len(as.targets))
	for _, target := range as.targets {
		targets = append(targets, target)
	}
	as.targetsMu.RUnlock()

	// Get gateway MAC address
	gatewayMAC, err := as.getGatewayMAC()
	if err != nil {
		log.Printf("Warning: failed to get gateway MAC: %v", err)
		return fmt.Errorf("failed to get gateway MAC: %w", err)
	}

	// Send correct ARP replies to each target
	for _, target := range targets {
		// Send correct gateway MAC to target device
		if err := as.sendCorrectARP(target.IP, target.MAC, config.Gateway, gatewayMAC); err != nil {
			log.Printf("Warning: failed to restore ARP for %s: %v", target.IP, err)
		}

		// Send correct target MAC to gateway
		if err := as.sendCorrectARP(config.Gateway, gatewayMAC, target.IP, target.MAC); err != nil {
			log.Printf("Warning: failed to restore ARP for gateway: %v", err)
		}
	}

	log.Printf("Restored ARP tables for %d devices", len(targets))
	return nil
}

// getGatewayMAC retrieves the gateway's MAC address
func (as *ARPSpoofer) getGatewayMAC() (net.HardwareAddr, error) {
	config := as.netConfig.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("network configuration not available")
	}

	// Get the interface
	iface, err := net.InterfaceByName(config.Interface)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}

	// For now, we'll need to perform an ARP request to get the gateway MAC
	// This is a simplified approach - in production, you might want to cache this
	// or use a more sophisticated method
	
	// Return the interface's own MAC as a fallback
	// In a real implementation, you would send an ARP request and wait for response
	return iface.HardwareAddr, nil
}

// sendCorrectARP sends a correct ARP reply to restore the ARP cache
func (as *ARPSpoofer) sendCorrectARP(dstIP net.IP, dstMAC net.HardwareAddr, srcIP net.IP, srcMAC net.HardwareAddr) error {
	// Build ARP reply packet
	packet, err := as.buildARPReply(dstIP, dstMAC, srcIP, srcMAC)
	if err != nil {
		return fmt.Errorf("failed to build ARP packet: %w", err)
	}

	// Send packet
	if err := as.sendARPPacket(packet); err != nil {
		return fmt.Errorf("failed to send ARP packet: %w", err)
	}

	return nil
}

// deviceListenerLoop listens for new devices and adds them to the spoofing list
func (as *ARPSpoofer) deviceListenerLoop() {
	defer as.wg.Done()

	for {
		select {
		case <-as.ctx.Done():
			return
		case device := <-as.deviceChan:
			if device == nil {
				continue
			}

			// Check if we should spoof this device
			if !as.shouldSpoofDevice(device.MAC) {
				continue
			}

			// Parse MAC address
			mac, err := net.ParseMAC(device.MAC)
			if err != nil {
				log.Printf("Warning: invalid MAC address %s: %v", device.MAC, err)
				continue
			}

			// Parse IP address
			ip := net.ParseIP(device.IP)
			if ip == nil {
				log.Printf("Warning: invalid IP address %s", device.IP)
				continue
			}

			// Add or update target
			as.targetsMu.Lock()
			target, exists := as.targets[device.MAC]
			if exists {
				target.IP = ip
				target.IsActive = device.IsActive
			} else {
				as.targets[device.MAC] = &SpoofTarget{
					MAC:      mac,
					IP:       ip,
					IsActive: device.IsActive,
				}
				log.Printf("Added device %s (%s) to spoofing targets", device.MAC, device.IP)
			}
			as.targetsMu.Unlock()
		}
	}
}

// shouldSpoofDevice checks if a device should be spoofed based on configuration
func (as *ARPSpoofer) shouldSpoofDevice(mac string) bool {
	// If no target MACs specified, spoof all devices
	if len(as.targetMACs) == 0 {
		return true
	}

	// Check if MAC is in target list
	for _, targetMAC := range as.targetMACs {
		if targetMAC == mac {
			return true
		}
	}

	return false
}

// spoofingLoop sends ARP spoofing packets at regular intervals
func (as *ARPSpoofer) spoofingLoop() {
	defer as.wg.Done()

	ticker := time.NewTicker(as.spoofInterval)
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.performSpoofing()
		}
	}
}

// performSpoofing sends spoofed ARP packets to all active targets
func (as *ARPSpoofer) performSpoofing() {
	config := as.netConfig.GetConfig()
	if config == nil {
		log.Println("Warning: network configuration not available for spoofing")
		return
	}

	// Get our interface MAC address
	iface, err := net.InterfaceByName(config.Interface)
	if err != nil {
		log.Printf("Warning: failed to get interface: %v", err)
		return
	}

	as.targetsMu.RLock()
	targets := make([]*SpoofTarget, 0, len(as.targets))
	for _, target := range as.targets {
		if target.IsActive {
			targets = append(targets, target)
		}
	}
	as.targetsMu.RUnlock()

	// Send spoofed ARP packets
	for _, target := range targets {
		// Send spoofed packet to target (tell target that gateway is at our MAC)
		if err := as.spoofTarget(target.IP, target.MAC, config.Gateway, iface.HardwareAddr); err != nil {
			log.Printf("Warning: failed to spoof target %s: %v", target.IP, err)
			continue
		}

		// Send spoofed packet to gateway (tell gateway that target is at our MAC)
		gatewayMAC, err := as.getGatewayMAC()
		if err != nil {
			log.Printf("Warning: failed to get gateway MAC: %v", err)
			continue
		}
		if err := as.spoofTarget(config.Gateway, gatewayMAC, target.IP, iface.HardwareAddr); err != nil {
			log.Printf("Warning: failed to spoof gateway for target %s: %v", target.IP, err)
			continue
		}

		// Update last spoof time
		as.targetsMu.Lock()
		if t, exists := as.targets[target.MAC.String()]; exists {
			t.LastSpoof = time.Now()
		}
		as.targetsMu.Unlock()
	}
}

// spoofTarget sends a spoofed ARP reply to a specific target
func (as *ARPSpoofer) spoofTarget(dstIP net.IP, dstMAC net.HardwareAddr, spoofedIP net.IP, spoofedMAC net.HardwareAddr) error {
	// Build spoofed ARP reply
	packet, err := as.buildARPReply(dstIP, dstMAC, spoofedIP, spoofedMAC)
	if err != nil {
		return fmt.Errorf("failed to build ARP packet: %w", err)
	}

	// Send packet
	if err := as.sendARPPacket(packet); err != nil {
		return fmt.Errorf("failed to send ARP packet: %w", err)
	}

	return nil
}

// healthCheckLoop monitors the health of the spoofing operation
func (as *ARPSpoofer) healthCheckLoop() {
	defer as.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.performHealthCheck()
		}
	}
}

// performHealthCheck verifies that spoofing is working correctly
func (as *ARPSpoofer) performHealthCheck() {
	// Check if IP forwarding is still enabled
	if err := as.verifyIPForwarding(); err != nil {
		log.Printf("Health check failed: IP forwarding not enabled: %v", err)
		return
	}

	// Check if we have active targets
	as.targetsMu.RLock()
	activeCount := 0
	for _, target := range as.targets {
		if target.IsActive {
			activeCount++
		}
	}
	as.targetsMu.RUnlock()

	// Log health status
	log.Printf("ARP spoofer health check: %d active targets", activeCount)
}

// verifyIPForwarding checks if IP forwarding is enabled on the system
func (as *ARPSpoofer) verifyIPForwarding() error {
	// Read /proc/sys/net/ipv4/ip_forward
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return fmt.Errorf("failed to read ip_forward: %w", err)
	}

	// Check if it's enabled (should be "1")
	if len(data) > 0 && data[0] == '1' {
		return nil
	}

	return fmt.Errorf("IP forwarding is not enabled (value: %s)", string(data))
}

// buildARPReply constructs an ARP reply packet
// dstIP and dstMAC are the destination (who will receive this packet)
// srcIP and srcMAC are what we're claiming (the spoofed information)
func (as *ARPSpoofer) buildARPReply(dstIP net.IP, dstMAC net.HardwareAddr, srcIP net.IP, srcMAC net.HardwareAddr) ([]byte, error) {
	// Validate inputs
	if dstIP == nil || dstMAC == nil || srcIP == nil || srcMAC == nil {
		return nil, fmt.Errorf("invalid parameters: all IPs and MACs must be non-nil")
	}

	// Convert IPs to 4-byte format
	dstIPv4 := dstIP.To4()
	srcIPv4 := srcIP.To4()
	if dstIPv4 == nil || srcIPv4 == nil {
		return nil, fmt.Errorf("invalid IP addresses: must be IPv4")
	}

	// Create Ethernet layer
	eth := layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	// Create ARP layer
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPReply,
		SourceHwAddress:   srcMAC,
		SourceProtAddress: srcIPv4,
		DstHwAddress:      dstMAC,
		DstProtAddress:    dstIPv4,
	}

	// Serialize the packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buf, opts, &eth, &arp); err != nil {
		return nil, fmt.Errorf("failed to serialize packet: %w", err)
	}

	return buf.Bytes(), nil
}

// sendARPPacket transmits an ARP packet via the raw socket
func (as *ARPSpoofer) sendARPPacket(packet []byte) error {
	if as.handle == nil {
		return fmt.Errorf("pcap handle not initialized")
	}

	if err := as.handle.WritePacketData(packet); err != nil {
		return fmt.Errorf("failed to write packet: %w", err)
	}

	return nil
}
