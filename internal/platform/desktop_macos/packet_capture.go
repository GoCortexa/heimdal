//go:build darwin
// +build darwin

package desktop_macos

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// MacOSPacketCapture implements PacketCaptureProvider for macOS using libpcap
type MacOSPacketCapture struct {
	handle       *pcap.Handle
	packetSource *gopacket.PacketSource
	stats        platform.CaptureStats
}

// NewMacOSPacketCapture creates a new macOS packet capture provider
func NewMacOSPacketCapture() *MacOSPacketCapture {
	return &MacOSPacketCapture{}
}

// IsLibpcapAvailable checks if libpcap is available on the system
func IsLibpcapAvailable() bool {
	// libpcap is built into macOS, so we just check if we can list devices
	_, err := pcap.FindAllDevs()
	return err == nil
}

// CheckLibpcapPermissions checks if the current user has permissions for packet capture
func CheckLibpcapPermissions() (bool, error) {
	// Try to open a device briefly to check permissions
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
			strings.Contains(err.Error(), "Operation not permitted") {
			return false, nil
		}
		// Other errors might not be permission-related
		return false, fmt.Errorf("failed to test packet capture: %w", err)
	}
	handle.Close()

	return true, nil
}

// GetLibpcapPermissionGuidance returns instructions for granting libpcap permissions
func GetLibpcapPermissionGuidance() string {
	return `Packet capture requires administrator privileges on macOS.

To grant permissions, you have two options:

Option 1: Run with sudo (temporary)
  sudo /Applications/Heimdal.app/Contents/MacOS/heimdal-desktop

Option 2: Grant permanent permissions (recommended)
  1. Open System Preferences
  2. Go to Security & Privacy
  3. Click the Privacy tab
  4. Select "Full Disk Access" from the left sidebar
  5. Click the lock icon and enter your password
  6. Click the "+" button and add Heimdal.app
  7. Restart the Heimdal application

For more information about packet capture on macOS, visit:
https://www.tcpdump.org/manpages/pcap.3pcap.html`
}

// RequestLibpcapPermissions attempts to request permissions via system dialog
func RequestLibpcapPermissions() error {
	// On macOS, we can use osascript to show a dialog requesting admin privileges
	script := `do shell script "echo 'Requesting packet capture permissions...'" with administrator privileges`
	cmd := exec.Command("osascript", "-e", script)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to request permissions: %w, output: %s", err, string(output))
	}

	return nil
}

// Open initializes packet capture on the specified interface
func (m *MacOSPacketCapture) Open(interfaceName string, promiscuous bool, filter string) error {
	// Check if libpcap is available
	if !IsLibpcapAvailable() {
		return fmt.Errorf("libpcap is not available on this system")
	}

	// Check permissions
	hasPermission, err := CheckLibpcapPermissions()
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions for packet capture. %s", GetLibpcapPermissionGuidance())
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

	m.handle = handle

	// Apply BPF filter if provided
	if filter != "" {
		if err := m.handle.SetBPFFilter(filter); err != nil {
			m.handle.Close()
			return fmt.Errorf("failed to set BPF filter: %w", err)
		}
	}

	// Create packet source
	m.packetSource = gopacket.NewPacketSource(m.handle, m.handle.LinkType())

	return nil
}

// ReadPacket returns the next captured packet
func (m *MacOSPacketCapture) ReadPacket() (*platform.Packet, error) {
	if m.packetSource == nil {
		return nil, fmt.Errorf("packet capture not initialized")
	}

	// Read next packet
	packet, err := m.packetSource.NextPacket()
	if err != nil {
		return nil, err
	}

	// Increment captured packets counter
	m.stats.PacketsCaptured++

	// Parse packet into platform.Packet format
	return m.parsePacket(packet)
}

// parsePacket converts a gopacket.Packet to platform.Packet
func (m *MacOSPacketCapture) parsePacket(packet gopacket.Packet) (*platform.Packet, error) {
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
func (m *MacOSPacketCapture) Close() error {
	if m.handle != nil {
		m.handle.Close()
		m.handle = nil
		m.packetSource = nil
	}
	return nil
}

// GetStats returns capture statistics
func (m *MacOSPacketCapture) GetStats() (*platform.CaptureStats, error) {
	if m.handle == nil {
		return &m.stats, nil
	}

	// Get pcap stats
	pcapStats, err := m.handle.Stats()
	if err != nil {
		// Return our internal stats if pcap stats are unavailable
		return &m.stats, nil
	}

	// Update stats from pcap
	m.stats.PacketsDropped = uint64(pcapStats.PacketsDropped)
	m.stats.PacketsFiltered = uint64(pcapStats.PacketsIfDropped)

	return &m.stats, nil
}

// ListInterfaces returns a list of available network interfaces on macOS
func ListInterfaces() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}
	return devices, nil
}
