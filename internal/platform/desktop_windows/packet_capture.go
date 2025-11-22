// +build windows

package desktop_windows

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

// WindowsPacketCapture implements PacketCaptureProvider for Windows using Npcap
type WindowsPacketCapture struct {
	handle       *pcap.Handle
	packetSource *gopacket.PacketSource
	stats        platform.CaptureStats
}

// NewWindowsPacketCapture creates a new Windows packet capture provider
func NewWindowsPacketCapture() *WindowsPacketCapture {
	return &WindowsPacketCapture{}
}

// IsNpcapInstalled checks if Npcap is installed on the system
func IsNpcapInstalled() bool {
	// Check for Npcap installation by looking for the Npcap service
	cmd := exec.Command("sc", "query", "npcap")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	
	// Check if the service exists
	return strings.Contains(string(output), "SERVICE_NAME: npcap")
}

// GetNpcapInstallationGuidance returns instructions for installing Npcap
func GetNpcapInstallationGuidance() string {
	return `Npcap is required for packet capture on Windows.

Please download and install Npcap from:
https://npcap.com/#download

Installation notes:
1. Download the latest Npcap installer
2. Run the installer with administrator privileges
3. During installation, ensure "WinPcap API-compatible Mode" is enabled
4. Restart the Heimdal application after installation

For more information, visit: https://npcap.com/guide/`
}

// Open initializes packet capture on the specified interface
func (w *WindowsPacketCapture) Open(interfaceName string, promiscuous bool, filter string) error {
	// Check if Npcap is installed
	if !IsNpcapInstalled() {
		return fmt.Errorf("Npcap is not installed. %s", GetNpcapInstallationGuidance())
	}

	// Open the device for packet capture
	handle, err := pcap.OpenLive(
		interfaceName,
		65536,      // snapshot length
		promiscuous, // promiscuous mode
		pcap.BlockForever,
	)
	if err != nil {
		return fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
	}

	w.handle = handle

	// Apply BPF filter if provided
	if filter != "" {
		if err := w.handle.SetBPFFilter(filter); err != nil {
			w.handle.Close()
			return fmt.Errorf("failed to set BPF filter: %w", err)
		}
	}

	// Create packet source
	w.packetSource = gopacket.NewPacketSource(w.handle, w.handle.LinkType())

	return nil
}

// ReadPacket returns the next captured packet
func (w *WindowsPacketCapture) ReadPacket() (*platform.Packet, error) {
	if w.packetSource == nil {
		return nil, fmt.Errorf("packet capture not initialized")
	}

	// Read next packet
	packet, err := w.packetSource.NextPacket()
	if err != nil {
		return nil, err
	}

	// Increment captured packets counter
	w.stats.PacketsCaptured++

	// Parse packet into platform.Packet format
	return w.parsePacket(packet)
}

// parsePacket converts a gopacket.Packet to platform.Packet
func (w *WindowsPacketCapture) parsePacket(packet gopacket.Packet) (*platform.Packet, error) {
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
func (w *WindowsPacketCapture) Close() error {
	if w.handle != nil {
		w.handle.Close()
		w.handle = nil
		w.packetSource = nil
	}
	return nil
}

// GetStats returns capture statistics
func (w *WindowsPacketCapture) GetStats() (*platform.CaptureStats, error) {
	if w.handle == nil {
		return &w.stats, nil
	}

	// Get pcap stats
	pcapStats, err := w.handle.Stats()
	if err != nil {
		// Return our internal stats if pcap stats are unavailable
		return &w.stats, nil
	}

	// Update stats from pcap
	w.stats.PacketsDropped = uint64(pcapStats.PacketsDropped)
	w.stats.PacketsFiltered = uint64(pcapStats.PacketsIfDropped)

	return &w.stats, nil
}

// ListInterfaces returns a list of available network interfaces on Windows
func ListInterfaces() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}
	return devices, nil
}
