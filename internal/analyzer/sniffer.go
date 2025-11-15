// Package analyzer provides packet capture and analysis functionality.
//
// The Sniffer component captures network packets from the interface in promiscuous mode
// and extracts relevant metadata for behavioral profiling. It uses the gopacket library
// for efficient packet parsing and implements rate limiting to prevent resource exhaustion.
//
// Packet Capture:
//   - Opens network interface in promiscuous mode using pcap
//   - Applies BPF filter to reduce noise: "not broadcast and not multicast"
//   - Captures all packets flowing through the interface
//
// Packet Processing:
//   - Parses Ethernet layer for source MAC address
//   - Parses IP layer for destination IP address
//   - Parses TCP/UDP layer for destination port and protocol
//   - Creates PacketInfo struct with extracted metadata
//   - Sends to packetChan for behavioral profiling
//
// Rate Limiting:
//   - Implements rate limiter (default: 10,000 packets/second)
//   - Uses buffered channel (size 1000) to prevent blocking
//   - Non-blocking send: drops packets if channel is full
//   - Prevents goroutine blocking during high traffic periods
//
// Optimization:
//   - Zero-copy packet processing where possible
//   - Minimal memory allocation per packet
//   - Efficient filtering to reduce processing load
//
// The Sniffer implements the Component interface for lifecycle management by the orchestrator.
package analyzer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/time/rate"

	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
)

// PacketInfo represents extracted metadata from a captured packet
type PacketInfo struct {
	Timestamp time.Time
	SrcMAC    string
	DstIP     string
	DstPort   uint16
	Protocol  string
	Size      uint32
}

// Sniffer captures and analyzes network packets
type Sniffer struct {
	netConfig   *netconfig.NetworkConfig
	handle      *pcap.Handle
	packetChan  chan<- PacketInfo
	rateLimiter *rate.Limiter
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewSniffer creates a new packet sniffer instance
func NewSniffer(netConfig *netconfig.NetworkConfig, packetChan chan<- PacketInfo) (*Sniffer, error) {
	if netConfig == nil {
		return nil, fmt.Errorf("network configuration is required")
	}
	if packetChan == nil {
		return nil, fmt.Errorf("packet channel is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create rate limiter: 10,000 packets per second
	limiter := rate.NewLimiter(rate.Limit(10000), 10000)

	sniffer := &Sniffer{
		netConfig:   netConfig,
		packetChan:  packetChan,
		rateLimiter: limiter,
		ctx:         ctx,
		cancel:      cancel,
	}

	return sniffer, nil
}

// Start begins packet capture and analysis
func (s *Sniffer) Start() error {
	// Open network interface in promiscuous mode
	handle, err := pcap.OpenLive(
		s.netConfig.Interface,
		1600,  // snapshot length
		true,  // promiscuous mode
		pcap.BlockForever,
	)
	if err != nil {
		return fmt.Errorf("failed to open interface %s: %w", s.netConfig.Interface, err)
	}
	s.handle = handle

	// Apply BPF filter to reduce noise
	bpfFilter := "not broadcast and not multicast"
	if err := s.handle.SetBPFFilter(bpfFilter); err != nil {
		s.handle.Close()
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	log.Printf("[Sniffer] Started packet capture on interface %s with filter: %s", s.netConfig.Interface, bpfFilter)

	// Start packet processing goroutine
	s.wg.Add(1)
	go s.captureLoop()

	return nil
}

// Stop gracefully stops the packet sniffer
func (s *Sniffer) Stop() error {
	log.Println("[Sniffer] Stopping packet capture...")
	s.cancel()
	
	if s.handle != nil {
		s.handle.Close()
	}
	
	s.wg.Wait()
	log.Println("[Sniffer] Stopped")
	return nil
}

// Name returns the component name
func (s *Sniffer) Name() string {
	return "Sniffer"
}

// captureLoop continuously captures and processes packets
func (s *Sniffer) captureLoop() {
	defer s.wg.Done()

	packetSource := gopacket.NewPacketSource(s.handle, s.handle.LinkType())
	packetSource.NoCopy = true // Performance optimization

	for {
		select {
		case <-s.ctx.Done():
			return
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}
			s.processPacket(packet)
		}
	}
}

// processPacket extracts metadata from a captured packet
func (s *Sniffer) processPacket(packet gopacket.Packet) {
	// Check rate limit - drop packet if exceeded
	if !s.rateLimiter.Allow() {
		return
	}

	// Extract Ethernet layer for source MAC
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}
	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return
	}

	// Extract IP layer for destination IP
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		// Try IPv6
		ipLayer = packet.Layer(layers.LayerTypeIPv6)
		if ipLayer == nil {
			return
		}
	}

	var dstIP string
	var protocol string
	var dstPort uint16

	// Handle IPv4
	if ipv4, ok := ipLayer.(*layers.IPv4); ok {
		dstIP = ipv4.DstIP.String()
		protocol = ipv4.Protocol.String()
	} else if ipv6, ok := ipLayer.(*layers.IPv6); ok {
		// Handle IPv6
		dstIP = ipv6.DstIP.String()
		protocol = ipv6.NextHeader.String()
	} else {
		return
	}

	// Extract TCP/UDP layer for destination port
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer != nil {
		tcp, ok := tcpLayer.(*layers.TCP)
		if ok {
			dstPort = uint16(tcp.DstPort)
			protocol = "TCP"
		}
	} else {
		udpLayer := packet.Layer(layers.LayerTypeUDP)
		if udpLayer != nil {
			udp, ok := udpLayer.(*layers.UDP)
			if ok {
				dstPort = uint16(udp.DstPort)
				protocol = "UDP"
			}
		}
	}

	// Create PacketInfo struct with extracted metadata
	packetInfo := PacketInfo{
		Timestamp: time.Now(),
		SrcMAC:    eth.SrcMAC.String(),
		DstIP:     dstIP,
		DstPort:   dstPort,
		Protocol:  protocol,
		Size:      uint32(len(packet.Data())),
	}

	// Send to channel (non-blocking)
	select {
	case s.packetChan <- packetInfo:
		// Successfully sent
	default:
		// Channel full, drop packet to prevent blocking
	}
}
