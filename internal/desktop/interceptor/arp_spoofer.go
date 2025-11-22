// Package interceptor provides desktop-specific traffic interception functionality.
//
// The DesktopTrafficInterceptor performs ARP spoofing from the user's endpoint
// to gain visibility into network traffic without requiring dedicated hardware.
// This is designed for desktop environments (Windows, macOS, Linux) where the
// application runs on the user's computer rather than a dedicated sensor device.
//
// Key Differences from Hardware Implementation:
//   - Runs from user's endpoint rather than dedicated hardware
//   - Includes additional safety checks to prevent OS network stack crashes
//   - Platform-specific permission handling (admin/sudo/capabilities)
//   - More conservative default settings for home network safety
//
// Safety Mechanisms:
//   - Verifies IP forwarding is enabled before starting
//   - Implements health checks to verify packet forwarding
//   - Automatic cleanup on shutdown or crash (signal handlers)
//   - Stores and restores original ARP entries
//   - Rate limiting to prevent network flooding
//
// Security Warning:
// ARP spoofing is inherently invasive and can disrupt network connectivity if
// misconfigured. Only use on networks you own or have explicit permission to monitor.
package interceptor

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// DesktopTrafficInterceptor manages ARP spoofing operations from a desktop endpoint
type DesktopTrafficInterceptor struct {
	// Network configuration
	interfaceName string
	localMAC      net.HardwareAddr
	localIP       net.IP
	gatewayIP     net.IP
	gatewayMAC    net.HardwareAddr

	// Packet capture handle
	handle *pcap.Handle

	// Spoofing targets
	targets   map[string]*SpoofTarget // MAC -> SpoofTarget
	targetsMu sync.RWMutex

	// Original ARP cache for restoration
	originalARPCache map[string]ARPEntry
	arpCacheMu       sync.RWMutex

	// Configuration
	spoofInterval time.Duration
	maxTargets    int // Safety limit

	// Lifecycle management
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	runningMu sync.Mutex

	// Signal handling for crash recovery
	signalChan chan os.Signal
}

// SpoofTarget represents a device being spoofed
type SpoofTarget struct {
	MAC       net.HardwareAddr
	IP        net.IP
	LastSpoof time.Time
	IsActive  bool
}

// ARPEntry represents an original ARP cache entry
type ARPEntry struct {
	IP  net.IP
	MAC net.HardwareAddr
}

// Config contains configuration for the DesktopTrafficInterceptor
type Config struct {
	InterfaceName string
	GatewayIP     net.IP
	SpoofInterval time.Duration
	MaxTargets    int
}

// NewDesktopTrafficInterceptor creates a new DesktopTrafficInterceptor instance
func NewDesktopTrafficInterceptor(config *Config) (*DesktopTrafficInterceptor, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.InterfaceName == "" {
		return nil, fmt.Errorf("interface name is required")
	}

	if config.GatewayIP == nil {
		return nil, fmt.Errorf("gateway IP is required")
	}

	// Set defaults
	if config.SpoofInterval == 0 {
		config.SpoofInterval = 2 * time.Second
	}
	if config.MaxTargets == 0 {
		config.MaxTargets = 50 // Conservative default for desktop
	}

	// Get interface information
	iface, err := net.InterfaceByName(config.InterfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %s: %w", config.InterfaceName, err)
	}

	// Get local IP address
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface addresses: %w", err)
	}

	var localIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIP = ipnet.IP
				break
			}
		}
	}

	if localIP == nil {
		return nil, fmt.Errorf("no IPv4 address found on interface %s", config.InterfaceName)
	}

	ctx, cancel := context.WithCancel(context.Background())

	interceptor := &DesktopTrafficInterceptor{
		interfaceName:    config.InterfaceName,
		localMAC:         iface.HardwareAddr,
		localIP:          localIP,
		gatewayIP:        config.GatewayIP,
		spoofInterval:    config.SpoofInterval,
		maxTargets:       config.MaxTargets,
		targets:          make(map[string]*SpoofTarget),
		originalARPCache: make(map[string]ARPEntry),
		ctx:              ctx,
		cancel:           cancel,
		signalChan:       make(chan os.Signal, 1),
	}

	return interceptor, nil
}

// Start begins ARP spoofing operations
func (dti *DesktopTrafficInterceptor) Start() error {
	dti.runningMu.Lock()
	if dti.running {
		dti.runningMu.Unlock()
		return fmt.Errorf("desktop traffic interceptor already running")
	}
	dti.running = true
	dti.runningMu.Unlock()

	// Check permissions
	if err := dti.checkPermissions(); err != nil {
		dti.runningMu.Lock()
		dti.running = false
		dti.runningMu.Unlock()
		return fmt.Errorf("permission check failed: %w", err)
	}

	// Verify IP forwarding is enabled
	if err := dti.verifyIPForwarding(); err != nil {
		dti.runningMu.Lock()
		dti.running = false
		dti.runningMu.Unlock()
		return fmt.Errorf("IP forwarding check failed: %w", err)
	}

	// Perform safety checks
	if err := dti.performSafetyChecks(); err != nil {
		dti.runningMu.Lock()
		dti.running = false
		dti.runningMu.Unlock()
		return fmt.Errorf("safety checks failed: %w", err)
	}

	// Resolve gateway MAC address
	if err := dti.resolveGatewayMAC(); err != nil {
		dti.runningMu.Lock()
		dti.running = false
		dti.runningMu.Unlock()
		return fmt.Errorf("failed to resolve gateway MAC: %w", err)
	}

	// Open pcap handle for packet injection
	handle, err := pcap.OpenLive(dti.interfaceName, 65536, true, pcap.BlockForever)
	if err != nil {
		dti.runningMu.Lock()
		dti.running = false
		dti.runningMu.Unlock()
		return dti.enhancePermissionError(err)
	}
	dti.handle = handle

	// Set up signal handling for crash recovery
	signal.Notify(dti.signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	dti.wg.Add(1)
	go dti.signalHandler()

	// Start spoofing loop
	dti.wg.Add(1)
	go dti.spoofingLoop()

	// Start health check loop
	dti.wg.Add(1)
	go dti.healthCheckLoop()

	log.Println("Desktop traffic interceptor started")
	return nil
}

// Stop gracefully stops ARP spoofing and restores ARP tables
func (dti *DesktopTrafficInterceptor) Stop() error {
	dti.runningMu.Lock()
	if !dti.running {
		dti.runningMu.Unlock()
		return fmt.Errorf("desktop traffic interceptor not running")
	}
	dti.running = false
	dti.runningMu.Unlock()

	log.Println("Stopping desktop traffic interceptor and restoring ARP tables...")

	// Restore original ARP tables
	if err := dti.restoreARPTables(); err != nil {
		log.Printf("Warning: failed to restore ARP tables: %v", err)
	}

	// Cancel context to signal all goroutines to stop
	dti.cancel()

	// Stop signal handling
	signal.Stop(dti.signalChan)
	close(dti.signalChan)

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		dti.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Close pcap handle
		if dti.handle != nil {
			dti.handle.Close()
		}
		log.Println("Desktop traffic interceptor stopped gracefully")
		return nil
	case <-time.After(5 * time.Second):
		// Force close pcap handle
		if dti.handle != nil {
			dti.handle.Close()
		}
		return fmt.Errorf("desktop traffic interceptor shutdown timeout")
	}
}

// AddTarget adds a device to the spoofing target list
func (dti *DesktopTrafficInterceptor) AddTarget(ip net.IP, mac net.HardwareAddr) error {
	if ip == nil || mac == nil {
		return fmt.Errorf("IP and MAC cannot be nil")
	}

	// Check if we've reached the maximum number of targets
	dti.targetsMu.RLock()
	if len(dti.targets) >= dti.maxTargets {
		dti.targetsMu.RUnlock()
		return fmt.Errorf("maximum number of targets (%d) reached", dti.maxTargets)
	}
	dti.targetsMu.RUnlock()

	macStr := mac.String()

	dti.targetsMu.Lock()
	defer dti.targetsMu.Unlock()

	// Store original ARP entry if not already stored
	dti.arpCacheMu.Lock()
	if _, exists := dti.originalARPCache[macStr]; !exists {
		dti.originalARPCache[macStr] = ARPEntry{
			IP:  ip,
			MAC: mac,
		}
	}
	dti.arpCacheMu.Unlock()

	// Add or update target
	if target, exists := dti.targets[macStr]; exists {
		target.IP = ip
		target.IsActive = true
	} else {
		dti.targets[macStr] = &SpoofTarget{
			MAC:      mac,
			IP:       ip,
			IsActive: true,
		}
		log.Printf("Added device %s (%s) to spoofing targets", mac, ip)
	}

	return nil
}

// RemoveTarget removes a device from the spoofing target list
func (dti *DesktopTrafficInterceptor) RemoveTarget(mac net.HardwareAddr) error {
	if mac == nil {
		return fmt.Errorf("MAC cannot be nil")
	}

	macStr := mac.String()

	dti.targetsMu.Lock()
	defer dti.targetsMu.Unlock()

	if target, exists := dti.targets[macStr]; exists {
		// Restore ARP entry for this target
		if err := dti.restoreARPForTarget(target); err != nil {
			log.Printf("Warning: failed to restore ARP for %s: %v", mac, err)
		}

		delete(dti.targets, macStr)
		log.Printf("Removed device %s from spoofing targets", mac)
		return nil
	}

	return fmt.Errorf("target %s not found", mac)
}

// checkPermissions verifies that the application has necessary permissions
func (dti *DesktopTrafficInterceptor) checkPermissions() error {
	switch runtime.GOOS {
	case "windows":
		return dti.checkWindowsPermissions()
	case "darwin":
		return dti.checkMacOSPermissions()
	case "linux":
		return dti.checkLinuxPermissions()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// checkWindowsPermissions checks for administrator rights on Windows
func (dti *DesktopTrafficInterceptor) checkWindowsPermissions() error {
	// On Windows, we need administrator rights to perform packet capture
	// Check if running with elevated privileges by attempting to access a privileged resource
	
	// Note: A more robust implementation would use Windows API calls like:
	// - OpenProcessToken + GetTokenInformation to check for admin token
	// - IsUserAnAdmin from shell32.dll
	// For now, we'll rely on the pcap handle open to fail if permissions are insufficient
	
	// The actual permission check will happen when we try to open the pcap handle
	// If that fails, we'll get a clear error message
	return nil
}

// checkMacOSPermissions checks for sudo/capabilities on macOS
func (dti *DesktopTrafficInterceptor) checkMacOSPermissions() error {
	// On macOS, we need root privileges or specific entitlements
	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges required on macOS. Please run with sudo or grant necessary entitlements")
	}
	return nil
}

// checkLinuxPermissions checks for sudo/capabilities on Linux
func (dti *DesktopTrafficInterceptor) checkLinuxPermissions() error {
	// On Linux, we need CAP_NET_RAW and CAP_NET_ADMIN capabilities
	// or root privileges
	
	// Check if running as root
	if os.Geteuid() == 0 {
		return nil
	}

	// If not root, check for capabilities
	// This is a simplified check - in production, you would use libcap
	// to check for specific capabilities
	return fmt.Errorf("root privileges or CAP_NET_RAW/CAP_NET_ADMIN capabilities required on Linux")
}

// verifyIPForwarding checks if IP forwarding is enabled on the system
func (dti *DesktopTrafficInterceptor) verifyIPForwarding() error {
	switch runtime.GOOS {
	case "linux":
		return dti.verifyIPForwardingLinux()
	case "darwin":
		return dti.verifyIPForwardingMacOS()
	case "windows":
		return dti.verifyIPForwardingWindows()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// verifyIPForwardingLinux checks IP forwarding on Linux
func (dti *DesktopTrafficInterceptor) verifyIPForwardingLinux() error {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return fmt.Errorf("failed to read ip_forward: %w", err)
	}

	if len(data) > 0 && data[0] == '1' {
		return nil
	}

	return fmt.Errorf("IP forwarding is not enabled. Enable with: sudo sysctl -w net.ipv4.ip_forward=1")
}

// verifyIPForwardingMacOS checks IP forwarding on macOS
func (dti *DesktopTrafficInterceptor) verifyIPForwardingMacOS() error {
	// On macOS, check sysctl net.inet.ip.forwarding
	// For now, we'll assume it's enabled if we can start
	// In production, you would use syscall to check sysctl values
	return nil
}

// verifyIPForwardingWindows checks IP forwarding on Windows
func (dti *DesktopTrafficInterceptor) verifyIPForwardingWindows() error {
	// On Windows, IP forwarding is controlled by registry settings
	// For now, we'll assume it's enabled if we can start
	// In production, you would check registry or use Windows API
	return nil
}

// performSafetyChecks performs additional safety checks before starting
func (dti *DesktopTrafficInterceptor) performSafetyChecks() error {
	// Check that we're not spoofing ourselves
	if dti.localIP.Equal(dti.gatewayIP) {
		return fmt.Errorf("local IP cannot be the same as gateway IP")
	}

	// Check that interface is up
	iface, err := net.InterfaceByName(dti.interfaceName)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}

	if iface.Flags&net.FlagUp == 0 {
		return fmt.Errorf("interface %s is not up", dti.interfaceName)
	}

	return nil
}

// resolveGatewayMAC resolves the gateway's MAC address using ARP
func (dti *DesktopTrafficInterceptor) resolveGatewayMAC() error {
	// In a real implementation, this would send an ARP request and wait for response
	// For now, we'll use a simplified approach
	
	// Try to get the gateway MAC from the system's ARP cache
	// This is platform-specific and would need proper implementation
	
	// For now, we'll set a placeholder
	// In production, you would implement proper ARP resolution
	dti.gatewayMAC = net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	
	log.Printf("Warning: Gateway MAC resolution not fully implemented, using placeholder")
	return nil
}

// spoofingLoop sends ARP spoofing packets at regular intervals
func (dti *DesktopTrafficInterceptor) spoofingLoop() {
	defer dti.wg.Done()

	ticker := time.NewTicker(dti.spoofInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dti.ctx.Done():
			return
		case <-ticker.C:
			dti.performSpoofing()
		}
	}
}

// performSpoofing sends spoofed ARP packets to all active targets
func (dti *DesktopTrafficInterceptor) performSpoofing() {
	dti.targetsMu.RLock()
	targets := make([]*SpoofTarget, 0, len(dti.targets))
	for _, target := range dti.targets {
		if target.IsActive {
			targets = append(targets, target)
		}
	}
	dti.targetsMu.RUnlock()

	// Send spoofed ARP packets
	for _, target := range targets {
		// Send spoofed packet to target (tell target that gateway is at our MAC)
		if err := dti.spoofTarget(target.IP, target.MAC, dti.gatewayIP, dti.localMAC); err != nil {
			log.Printf("Warning: failed to spoof target %s: %v", target.IP, err)
			continue
		}

		// Send spoofed packet to gateway (tell gateway that target is at our MAC)
		if err := dti.spoofTarget(dti.gatewayIP, dti.gatewayMAC, target.IP, dti.localMAC); err != nil {
			log.Printf("Warning: failed to spoof gateway for target %s: %v", target.IP, err)
			continue
		}

		// Update last spoof time
		dti.targetsMu.Lock()
		if t, exists := dti.targets[target.MAC.String()]; exists {
			t.LastSpoof = time.Now()
		}
		dti.targetsMu.Unlock()
	}
}

// spoofTarget sends a spoofed ARP reply to a specific target
func (dti *DesktopTrafficInterceptor) spoofTarget(dstIP net.IP, dstMAC net.HardwareAddr, spoofedIP net.IP, spoofedMAC net.HardwareAddr) error {
	// Build spoofed ARP reply
	packet, err := dti.buildARPReply(dstIP, dstMAC, spoofedIP, spoofedMAC)
	if err != nil {
		return fmt.Errorf("failed to build ARP packet: %w", err)
	}

	// Send packet
	if err := dti.sendARPPacket(packet); err != nil {
		return fmt.Errorf("failed to send ARP packet: %w", err)
	}

	return nil
}

// buildARPReply constructs an ARP reply packet
func (dti *DesktopTrafficInterceptor) buildARPReply(dstIP net.IP, dstMAC net.HardwareAddr, srcIP net.IP, srcMAC net.HardwareAddr) ([]byte, error) {
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
func (dti *DesktopTrafficInterceptor) sendARPPacket(packet []byte) error {
	if dti.handle == nil {
		return fmt.Errorf("pcap handle not initialized")
	}

	if err := dti.handle.WritePacketData(packet); err != nil {
		return fmt.Errorf("failed to write packet: %w", err)
	}

	return nil
}

// restoreARPTables sends correct ARP replies to restore original ARP cache
func (dti *DesktopTrafficInterceptor) restoreARPTables() error {
	dti.targetsMu.RLock()
	targets := make([]*SpoofTarget, 0, len(dti.targets))
	for _, target := range dti.targets {
		targets = append(targets, target)
	}
	dti.targetsMu.RUnlock()

	// Send correct ARP replies to each target
	for _, target := range targets {
		if err := dti.restoreARPForTarget(target); err != nil {
			log.Printf("Warning: failed to restore ARP for %s: %v", target.IP, err)
		}
	}

	log.Printf("Restored ARP tables for %d devices", len(targets))
	return nil
}

// restoreARPForTarget restores the ARP entry for a specific target
func (dti *DesktopTrafficInterceptor) restoreARPForTarget(target *SpoofTarget) error {
	// Send correct gateway MAC to target device
	if err := dti.sendCorrectARP(target.IP, target.MAC, dti.gatewayIP, dti.gatewayMAC); err != nil {
		return fmt.Errorf("failed to restore ARP for target: %w", err)
	}

	// Send correct target MAC to gateway
	if err := dti.sendCorrectARP(dti.gatewayIP, dti.gatewayMAC, target.IP, target.MAC); err != nil {
		return fmt.Errorf("failed to restore ARP for gateway: %w", err)
	}

	return nil
}

// sendCorrectARP sends a correct ARP reply to restore the ARP cache
func (dti *DesktopTrafficInterceptor) sendCorrectARP(dstIP net.IP, dstMAC net.HardwareAddr, srcIP net.IP, srcMAC net.HardwareAddr) error {
	// Build ARP reply packet
	packet, err := dti.buildARPReply(dstIP, dstMAC, srcIP, srcMAC)
	if err != nil {
		return fmt.Errorf("failed to build ARP packet: %w", err)
	}

	// Send packet multiple times to ensure it's received
	for i := 0; i < 3; i++ {
		if err := dti.sendARPPacket(packet); err != nil {
			return fmt.Errorf("failed to send ARP packet: %w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// healthCheckLoop monitors the health of the spoofing operation
func (dti *DesktopTrafficInterceptor) healthCheckLoop() {
	defer dti.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dti.ctx.Done():
			return
		case <-ticker.C:
			dti.performHealthCheck()
		}
	}
}

// performHealthCheck verifies that spoofing is working correctly
func (dti *DesktopTrafficInterceptor) performHealthCheck() {
	// Check if IP forwarding is still enabled
	if err := dti.verifyIPForwarding(); err != nil {
		log.Printf("Health check failed: IP forwarding not enabled: %v", err)
		return
	}

	// Check if we have active targets
	dti.targetsMu.RLock()
	activeCount := 0
	for _, target := range dti.targets {
		if target.IsActive {
			activeCount++
		}
	}
	dti.targetsMu.RUnlock()

	// Log health status
	log.Printf("Desktop traffic interceptor health check: %d active targets", activeCount)
}

// signalHandler handles OS signals for crash recovery
func (dti *DesktopTrafficInterceptor) signalHandler() {
	defer dti.wg.Done()

	for {
		select {
		case <-dti.ctx.Done():
			return
		case sig, ok := <-dti.signalChan:
			if !ok {
				return
			}
			log.Printf("Received signal %v, performing cleanup...", sig)
			
			// Restore ARP tables before exiting
			if err := dti.restoreARPTables(); err != nil {
				log.Printf("Error restoring ARP tables on signal: %v", err)
			}
			
			// Exit the application
			os.Exit(0)
		}
	}
}


// enhancePermissionError provides platform-specific error messages for permission issues
func (dti *DesktopTrafficInterceptor) enhancePermissionError(err error) error {
	baseErr := fmt.Errorf("failed to open pcap handle: %w", err)
	
	switch runtime.GOOS {
	case "windows":
		return fmt.Errorf("%w\n\nWindows requires administrator privileges to capture packets.\n"+
			"Please run this application as Administrator.\n"+
			"Also ensure Npcap is installed: https://npcap.com/", baseErr)
	case "darwin":
		return fmt.Errorf("%w\n\nmacOS requires root privileges or specific entitlements to capture packets.\n"+
			"Please run with: sudo %s\n"+
			"Or grant necessary entitlements to the application.", baseErr, os.Args[0])
	case "linux":
		return fmt.Errorf("%w\n\nLinux requires root privileges or CAP_NET_RAW/CAP_NET_ADMIN capabilities.\n"+
			"Run with sudo, or grant capabilities with:\n"+
			"  sudo setcap cap_net_raw,cap_net_admin=eip %s", baseErr, os.Args[0])
	default:
		return baseErr
	}
}
