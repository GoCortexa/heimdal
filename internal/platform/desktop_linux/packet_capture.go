//go:build linux
// +build linux

package desktop_linux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// LinuxPacketCapture implements PacketCaptureProvider for Linux desktop using libpcap
type LinuxPacketCapture struct {
	handle       *pcap.Handle
	packetSource *gopacket.PacketSource
	stats        platform.CaptureStats
}

// NewLinuxPacketCapture creates a new Linux desktop packet capture provider
func NewLinuxPacketCapture() *LinuxPacketCapture {
	return &LinuxPacketCapture{}
}

// IsLibpcapAvailable checks if libpcap is available on the system
func IsLibpcapAvailable() bool {
	// libpcap should be available on most Linux systems
	// Check if we can list devices
	_, err := pcap.FindAllDevs()
	return err == nil
}

// CheckCapabilities checks if the current user has required capabilities for packet capture
// Returns true if user has CAP_NET_RAW or CAP_NET_ADMIN, or is running as root
func CheckCapabilities() (bool, error) {
	// Check if running as root
	if os.Geteuid() == 0 {
		return true, nil
	}

	// Check for capabilities using getcap
	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command("getcap", exePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// getcap might not be installed or executable doesn't have caps
		// Try to open a device to test permissions
		return testPacketCapturePermissions()
	}

	// Check if output contains CAP_NET_RAW or CAP_NET_ADMIN
	outputStr := string(output)
	hasCapNetRaw := strings.Contains(outputStr, "cap_net_raw")
	hasCapNetAdmin := strings.Contains(outputStr, "cap_net_admin")

	if hasCapNetRaw || hasCapNetAdmin {
		return true, nil
	}

	// If no capabilities found, test by trying to open a device
	return testPacketCapturePermissions()
}

// testPacketCapturePermissions tests if we can actually capture packets
func testPacketCapturePermissions() (bool, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return false, fmt.Errorf("failed to list devices: %w", err)
	}

	if len(devices) == 0 {
		return false, fmt.Errorf("no network devices found")
	}

	// Try to open the first device with a very short timeout
	handle, err := pcap.OpenLive(devices[0].Name, 65536, false, 100*time.Millisecond)
	if err != nil {
		// Check if it's a permission error
		if strings.Contains(err.Error(), "permission") ||
			strings.Contains(err.Error(), "Operation not permitted") ||
			strings.Contains(err.Error(), "socket: Operation not permitted") {
			return false, nil
		}
		// Other errors might not be permission-related
		return false, fmt.Errorf("failed to test packet capture: %w", err)
	}
	handle.Close()

	return true, nil
}

// GetCapabilityGuidance returns instructions for granting packet capture capabilities
func GetCapabilityGuidance() string {
	exePath, err := os.Executable()
	if err != nil {
		exePath = "/path/to/heimdal-desktop"
	}

	return fmt.Sprintf(`Packet capture requires special capabilities on Linux.

You have several options:

Option 1: Grant capabilities to the executable (recommended)
  sudo setcap cap_net_raw,cap_net_admin=eip %s

Option 2: Run with sudo (temporary)
  sudo %s

Option 3: Add your user to the pcap group (if available)
  sudo usermod -a -G pcap $USER
  # Then log out and log back in

After granting capabilities, restart the Heimdal application.

For more information about packet capture on Linux, visit:
https://www.tcpdump.org/manpages/pcap.3pcap.html`, exePath, exePath)
}

// RequestCapabilities attempts to request capabilities via pkexec or sudo
func RequestCapabilities() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Try pkexec first (graphical sudo)
	cmd := exec.Command("pkexec", "setcap", "cap_net_raw,cap_net_admin=eip", exePath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	// If pkexec fails, try sudo
	cmd = exec.Command("sudo", "setcap", "cap_net_raw,cap_net_admin=eip", exePath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set capabilities: %w, output: %s", err, string(output))
	}

	return nil
}

// Open initializes packet capture on the specified interface
func (l *LinuxPacketCapture) Open(interfaceName string, promiscuous bool, filter string) error {
	// Check if libpcap is available
	if !IsLibpcapAvailable() {
		return fmt.Errorf("libpcap is not available on this system. Install libpcap-dev: sudo apt-get install libpcap-dev")
	}

	// Check capabilities
	hasCapability, err := CheckCapabilities()
	if err != nil {
		return fmt.Errorf("failed to check capabilities: %w", err)
	}
	if !hasCapability {
		return fmt.Errorf("insufficient capabilities for packet capture. %s", GetCapabilityGuidance())
	}

	// Open the device for packet capture
	handle, err := pcap.OpenLive(
		interfaceName,
		65536,       // snapshot length
		promiscuous, // promiscuous mode
		pcap.BlockForever,
	)
	if err != nil {
		return fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
	}

	l.handle = handle

	// Apply BPF filter if provided
	if filter != "" {
		if err := l.handle.SetBPFFilter(filter); err != nil {
			l.handle.Close()
			return fmt.Errorf("failed to set BPF filter: %w", err)
		}
	}

	// Create packet source
	l.packetSource = gopacket.NewPacketSource(l.handle, l.handle.LinkType())

	return nil
}

// ReadPacket returns the next captured packet
func (l *LinuxPacketCapture) ReadPacket() (*platform.Packet, error) {
	if l.packetSource == nil {
		return nil, fmt.Errorf("packet capture not initialized")
	}

	// Read next packet
	packet, err := l.packetSource.NextPacket()
	if err != nil {
		return nil, err
	}

	// Increment captured packets counter
	l.stats.PacketsCaptured++

	// Parse packet into platform.Packet format
	return l.parsePacket(packet)
}

// parsePacket converts a gopacket.Packet to platform.Packet
func (l *LinuxPacketCapture) parsePacket(packet gopacket.Packet) (*platform.Packet, error) {
	result := &platform.Packet{
		Timestamp: packet.Metadata().Timestamp,
		RawData:   packet.Data(),
	}

	// Extract Ethernet layer
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth, _ := ethLayer.(*layers.Ethernet)
		result.SrcMAC = eth.SrcMAC
		result.DstMAC = eth.DstMAC
	}

	// Extract IP layer
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		result.SrcIP = ip.SrcIP
		result.DstIP = ip.DstIP
		result.Protocol = ip.Protocol.String()
	} else if ipLayer := packet.Layer(layers.LayerTypeIPv6); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv6)
		result.SrcIP = ip.SrcIP
		result.DstIP = ip.DstIP
		result.Protocol = ip.NextHeader.String()
	}

	// Extract TCP layer
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		result.SrcPort = uint16(tcp.SrcPort)
		result.DstPort = uint16(tcp.DstPort)
		result.Protocol = "TCP"
		if app := packet.ApplicationLayer(); app != nil {
			result.PayloadSize = uint32(len(app.Payload()))
		}
	}

	// Extract UDP layer
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		result.SrcPort = uint16(udp.SrcPort)
		result.DstPort = uint16(udp.DstPort)
		result.Protocol = "UDP"
		if app := packet.ApplicationLayer(); app != nil {
			result.PayloadSize = uint32(len(app.Payload()))
		}
	}

	// Extract ICMP layer
	if icmpLayer := packet.Layer(layers.LayerTypeICMPv4); icmpLayer != nil {
		result.Protocol = "ICMP"
	} else if icmpLayer := packet.Layer(layers.LayerTypeICMPv6); icmpLayer != nil {
		result.Protocol = "ICMPv6"
	}

	// Set payload size if not already set
	if result.PayloadSize == 0 && packet.ApplicationLayer() != nil {
		result.PayloadSize = uint32(len(packet.ApplicationLayer().Payload()))
	}

	return result, nil
}

// Close releases packet capture resources
func (l *LinuxPacketCapture) Close() error {
	if l.handle != nil {
		l.handle.Close()
		l.handle = nil
		l.packetSource = nil
	}
	return nil
}

// GetStats returns capture statistics
func (l *LinuxPacketCapture) GetStats() (*platform.CaptureStats, error) {
	if l.handle == nil {
		return &l.stats, nil
	}

	// Get pcap stats
	pcapStats, err := l.handle.Stats()
	if err != nil {
		// Return our internal stats if pcap stats are unavailable
		return &l.stats, nil
	}

	// Update stats from pcap
	l.stats.PacketsDropped = uint64(pcapStats.PacketsDropped)
	l.stats.PacketsFiltered = uint64(pcapStats.PacketsIfDropped)

	return &l.stats, nil
}

// ListInterfaces returns a list of available network interfaces on Linux
func ListInterfaces() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}
	return devices, nil
}
