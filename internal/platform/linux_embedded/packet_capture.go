// Package linux_embedded provides platform-specific implementations for embedded Linux systems (Raspberry Pi).
// This package implements the platform abstraction interfaces for hardware sensors running on ARM64 Linux.
package linux_embedded

import (
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// LinuxEmbeddedPacketCapture implements PacketCaptureProvider for embedded Linux systems
// using libpcap for high-performance packet capture with raw sockets or AF_PACKET.
type LinuxEmbeddedPacketCapture struct {
	handle       *pcap.Handle
	packetSource *gopacket.PacketSource
	interfaceName string
	stats        platform.CaptureStats
}

// NewLinuxEmbeddedPacketCapture creates a new packet capture provider for embedded Linux
func NewLinuxEmbeddedPacketCapture() *LinuxEmbeddedPacketCapture {
	return &LinuxEmbeddedPacketCapture{}
}

// Open initializes packet capture on the specified interface
func (l *LinuxEmbeddedPacketCapture) Open(interfaceName string, promiscuous bool, filter string) error {
	if interfaceName == "" {
		return fmt.Errorf("interface name is required")
	}

	// Open network interface with pcap
	// Using pcap.BlockForever for timeout to wait indefinitely for packets
	handle, err := pcap.OpenLive(
		interfaceName,
		1600,        // snapshot length (enough for most packets)
		promiscuous, // promiscuous mode
		pcap.BlockForever,
	)
	if err != nil {
		return fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
	}

	// Apply BPF filter if provided
	if filter != "" {
		if err := handle.SetBPFFilter(filter); err != nil {
			handle.Close()
			return fmt.Errorf("failed to set BPF filter '%s': %w", filter, err)
		}
	}

	l.handle = handle
	l.interfaceName = interfaceName

	// Create packet source with NoCopy optimization for better performance
	l.packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
	l.packetSource.NoCopy = true

	// Initialize stats
	l.stats = platform.CaptureStats{
		PacketsCaptured: 0,
		PacketsDropped:  0,
		PacketsFiltered: 0,
	}

	return nil
}

// ReadPacket returns the next captured packet
// Returns nil packet when no more packets are available
func (l *LinuxEmbeddedPacketCapture) ReadPacket() (*platform.Packet, error) {
	if l.handle == nil || l.packetSource == nil {
		return nil, fmt.Errorf("packet capture not initialized, call Open first")
	}

	// Read next packet from source
	goPacket, ok := <-l.packetSource.Packets()
	if !ok {
		// Channel closed, no more packets
		return nil, nil
	}

	// Parse packet and extract metadata
	packet, err := l.parsePacket(goPacket)
	if err != nil {
		// Skip packets that can't be parsed
		return nil, err
	}

	// Update statistics
	l.stats.PacketsCaptured++

	return packet, nil
}

// parsePacket extracts metadata from a gopacket.Packet into platform.Packet
func (l *LinuxEmbeddedPacketCapture) parsePacket(goPacket gopacket.Packet) (*platform.Packet, error) {
	if goPacket == nil {
		return nil, fmt.Errorf("nil packet")
	}

	packet := &platform.Packet{
		Timestamp:   time.Now(),
		PayloadSize: uint32(len(goPacket.Data())),
		RawData:     goPacket.Data(),
	}

	// Extract Ethernet layer for MAC addresses
	ethLayer := goPacket.Layer(layers.LayerTypeEthernet)
	if ethLayer != nil {
		eth, ok := ethLayer.(*layers.Ethernet)
		if ok {
			packet.SrcMAC = eth.SrcMAC
			packet.DstMAC = eth.DstMAC
		}
	}

	// Extract IP layer for IP addresses
	ipLayer := goPacket.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ipv4, ok := ipLayer.(*layers.IPv4)
		if ok {
			packet.SrcIP = ipv4.SrcIP
			packet.DstIP = ipv4.DstIP
			packet.Protocol = ipv4.Protocol.String()
		}
	} else {
		// Try IPv6
		ipLayer = goPacket.Layer(layers.LayerTypeIPv6)
		if ipLayer != nil {
			ipv6, ok := ipLayer.(*layers.IPv6)
			if ok {
				packet.SrcIP = ipv6.SrcIP
				packet.DstIP = ipv6.DstIP
				packet.Protocol = ipv6.NextHeader.String()
			}
		}
	}

	// Extract TCP layer for ports
	tcpLayer := goPacket.Layer(layers.LayerTypeTCP)
	if tcpLayer != nil {
		tcp, ok := tcpLayer.(*layers.TCP)
		if ok {
			packet.SrcPort = uint16(tcp.SrcPort)
			packet.DstPort = uint16(tcp.DstPort)
			packet.Protocol = "TCP"
		}
	} else {
		// Try UDP
		udpLayer := goPacket.Layer(layers.LayerTypeUDP)
		if udpLayer != nil {
			udp, ok := udpLayer.(*layers.UDP)
			if ok {
				packet.SrcPort = uint16(udp.SrcPort)
				packet.DstPort = uint16(udp.DstPort)
				packet.Protocol = "UDP"
			}
		}
	}

	return packet, nil
}

// Close releases packet capture resources
func (l *LinuxEmbeddedPacketCapture) Close() error {
	if l.handle != nil {
		l.handle.Close()
		l.handle = nil
		l.packetSource = nil
	}
	return nil
}

// GetStats returns capture statistics (packets captured, dropped, etc.)
func (l *LinuxEmbeddedPacketCapture) GetStats() (*platform.CaptureStats, error) {
	if l.handle == nil {
		return nil, fmt.Errorf("packet capture not initialized")
	}

	// Get pcap stats
	pcapStats, err := l.handle.Stats()
	if err != nil {
		return nil, fmt.Errorf("failed to get pcap stats: %w", err)
	}

	// Update our stats with pcap stats
	l.stats.PacketsDropped = uint64(pcapStats.PacketsDropped)
	l.stats.PacketsFiltered = uint64(pcapStats.PacketsIfDropped)

	// Return a copy of the stats
	statsCopy := l.stats
	return &statsCopy, nil
}
