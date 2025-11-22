// Package packet provides shared packet capture and analysis functionality.
//
// The Analyzer component works with any PacketCaptureProvider implementation
// to capture and analyze network packets. It extracts relevant metadata for
// behavioral profiling and implements rate limiting to prevent resource exhaustion.
//
// This is the core shared logic used by both hardware and desktop products.
package packet

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/mosiko1234/heimdal/sensor/internal/platform"
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

// Analyzer processes packets from any capture provider
type Analyzer struct {
	provider    platform.PacketCaptureProvider
	rateLimiter *rate.Limiter
	outputChan  chan<- PacketInfo
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// Config contains configuration for the packet analyzer
type Config struct {
	// RateLimit is the maximum packets per second to process (0 = unlimited)
	RateLimit int
	// BufferSize is the size of the output channel buffer
	BufferSize int
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		RateLimit:  10000, // 10,000 packets per second
		BufferSize: 1000,  // 1000 packet buffer
	}
}

// NewAnalyzer creates a new packet analyzer instance
func NewAnalyzer(provider platform.PacketCaptureProvider, outputChan chan<- PacketInfo, cfg *Config) (*Analyzer, error) {
	if provider == nil {
		return nil, fmt.Errorf("packet capture provider is required")
	}
	if outputChan == nil {
		return nil, fmt.Errorf("output channel is required")
	}
	if cfg == nil {
		cfg = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create rate limiter
	var limiter *rate.Limiter
	if cfg.RateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimit)
	}

	analyzer := &Analyzer{
		provider:    provider,
		rateLimiter: limiter,
		outputChan:  outputChan,
		ctx:         ctx,
		cancel:      cancel,
	}

	return analyzer, nil
}

// Start begins packet capture and analysis
func (a *Analyzer) Start(interfaceName string, promiscuous bool, filter string) error {
	// Open the packet capture provider
	if err := a.provider.Open(interfaceName, promiscuous, filter); err != nil {
		return fmt.Errorf("failed to open packet capture: %w", err)
	}

	log.Printf("[Packet Analyzer] Started packet capture on interface %s (promiscuous: %v, filter: %s)",
		interfaceName, promiscuous, filter)

	// Start packet processing goroutine
	a.wg.Add(1)
	go a.captureLoop()

	return nil
}

// Stop gracefully stops the packet analyzer
func (a *Analyzer) Stop() error {
	log.Println("[Packet Analyzer] Stopping packet capture...")
	a.cancel()

	// Close the provider
	if err := a.provider.Close(); err != nil {
		log.Printf("[Packet Analyzer] Warning: error closing provider: %v", err)
	}

	a.wg.Wait()
	log.Println("[Packet Analyzer] Stopped")
	return nil
}

// GetStats returns capture statistics from the provider
func (a *Analyzer) GetStats() (*platform.CaptureStats, error) {
	return a.provider.GetStats()
}

// captureLoop continuously captures and processes packets
func (a *Analyzer) captureLoop() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			// Read packet from provider
			packet, err := a.provider.ReadPacket()
			if err != nil {
				// Check if context was cancelled
				if a.ctx.Err() != nil {
					return
				}
				// Log error and continue
				log.Printf("[Packet Analyzer] Error reading packet: %v", err)
				continue
			}

			if packet == nil {
				continue
			}

			// Process the packet
			a.processPacket(packet)
		}
	}
}

// processPacket extracts metadata from a captured packet
func (a *Analyzer) processPacket(packet *platform.Packet) {
	// Check rate limit - drop packet if exceeded
	if a.rateLimiter != nil && !a.rateLimiter.Allow() {
		return
	}

	// Extract packet information
	packetInfo := a.extractPacketInfo(packet)
	if packetInfo == nil {
		return
	}

	// Send to output channel (non-blocking)
	select {
	case a.outputChan <- *packetInfo:
		// Successfully sent
	default:
		// Channel full, drop packet to prevent blocking
	}
}

// extractPacketInfo extracts PacketInfo from a platform.Packet
func (a *Analyzer) extractPacketInfo(packet *platform.Packet) *PacketInfo {
	// Validate required fields
	if packet.SrcMAC == nil || len(packet.SrcMAC) == 0 {
		return nil
	}

	// Create PacketInfo struct with extracted metadata
	info := &PacketInfo{
		Timestamp: packet.Timestamp,
		SrcMAC:    packet.SrcMAC.String(),
		Protocol:  packet.Protocol,
		Size:      packet.PayloadSize,
	}

	// Extract destination IP
	if packet.DstIP != nil {
		info.DstIP = packet.DstIP.String()
	}

	// Extract destination port
	info.DstPort = packet.DstPort

	return info
}

// ProcessPacket is a public method for testing that processes a single packet
// This allows tests to inject packets directly without going through the capture loop
func (a *Analyzer) ProcessPacket(packet *platform.Packet) (*PacketInfo, error) {
	if packet == nil {
		return nil, fmt.Errorf("packet is nil")
	}

	info := a.extractPacketInfo(packet)
	if info == nil {
		return nil, fmt.Errorf("failed to extract packet info")
	}

	return info, nil
}
