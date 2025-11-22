package property

import (
	"net"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/mosiko1234/heimdal/sensor/internal/core/packet"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// Feature: monorepo-architecture, Property 2: Packet Metadata Extraction Completeness
// Validates: Requirements 2.5
//
// Property: For any valid network packet, the packet analysis module should extract
// all required fields (protocol, source/destination addresses, ports) and return a
// complete PacketInfo structure.
func TestProperty_PacketMetadataExtractionCompleteness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Packet metadata extraction is complete for all valid packets",
		prop.ForAll(
			func(srcMAC net.HardwareAddr, dstIP net.IP, dstPort uint16, protocol string, payloadSize uint32) bool {
				// Create a valid packet
				pkt := &platform.Packet{
					Timestamp:   time.Now(),
					SrcMAC:      srcMAC,
					DstMAC:      genHardwareAddr(),
					SrcIP:       genIPAddress(),
					DstIP:       dstIP,
					SrcPort:     genPort(),
					DstPort:     dstPort,
					Protocol:    protocol,
					PayloadSize: payloadSize,
					RawData:     make([]byte, payloadSize),
				}

				// Create analyzer with mock provider
				outputChan := make(chan packet.PacketInfo, 10)
				mockProvider := &MockPacketCaptureProvider{}
				analyzer, err := packet.NewAnalyzer(mockProvider, outputChan, nil)
				if err != nil {
					t.Logf("Failed to create analyzer: %v", err)
					return false
				}

				// Process the packet
				info, err := analyzer.ProcessPacket(pkt)
				if err != nil {
					t.Logf("Failed to process packet: %v", err)
					return false
				}

				// Verify all required fields are present and correct
				if info == nil {
					t.Log("PacketInfo is nil")
					return false
				}

				// Check that source MAC is extracted
				if info.SrcMAC == "" {
					t.Log("SrcMAC is empty")
					return false
				}
				if info.SrcMAC != srcMAC.String() {
					t.Logf("SrcMAC mismatch: expected %s, got %s", srcMAC.String(), info.SrcMAC)
					return false
				}

				// Check that destination IP is extracted
				if dstIP != nil && info.DstIP == "" {
					t.Log("DstIP is empty when it should be present")
					return false
				}
				if dstIP != nil && info.DstIP != dstIP.String() {
					t.Logf("DstIP mismatch: expected %s, got %s", dstIP.String(), info.DstIP)
					return false
				}

				// Check that destination port is extracted
				if info.DstPort != dstPort {
					t.Logf("DstPort mismatch: expected %d, got %d", dstPort, info.DstPort)
					return false
				}

				// Check that protocol is extracted
				if info.Protocol != protocol {
					t.Logf("Protocol mismatch: expected %s, got %s", protocol, info.Protocol)
					return false
				}

				// Check that size is extracted
				if info.Size != payloadSize {
					t.Logf("Size mismatch: expected %d, got %d", payloadSize, info.Size)
					return false
				}

				// Check that timestamp is set
				if info.Timestamp.IsZero() {
					t.Log("Timestamp is zero")
					return false
				}

				return true
			},
			genValidHardwareAddr(),
			genValidIPAddress(),
			genValidPort(),
			genValidProtocol(),
			genValidPayloadSize(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generators for property-based testing

func genValidHardwareAddr() gopter.Gen {
	return gen.SliceOfN(6, gen.UInt8()).
		Map(func(bytes []uint8) net.HardwareAddr {
			return net.HardwareAddr(bytes)
		}).
		SuchThat(func(mac net.HardwareAddr) bool {
			// Ensure MAC is not all zeros
			for _, b := range mac {
				if b != 0 {
					return true
				}
			}
			return false
		})
}

func genValidIPAddress() gopter.Gen {
	return gen.OneGenOf(genIPv4Address(), genIPv6Address())
}

func genIPv4Address() gopter.Gen {
	return gen.SliceOfN(4, gen.UInt8()).
		Map(func(bytes []uint8) net.IP {
			return net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])
		})
}

func genIPv6Address() gopter.Gen {
	return gen.SliceOfN(16, gen.UInt8()).
		Map(func(bytes []uint8) net.IP {
			return net.IP(bytes)
		})
}

func genValidPort() gopter.Gen {
	return gen.UInt16Range(1, 65535)
}

func genValidProtocol() gopter.Gen {
	return gen.OneConstOf("TCP", "UDP", "ICMP", "ICMPv6")
}

func genValidPayloadSize() gopter.Gen {
	return gen.UInt32Range(0, 65535)
}

// Helper functions for generating random data in tests

func genHardwareAddr() net.HardwareAddr {
	return net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
}

func genIPAddress() net.IP {
	return net.IPv4(192, 168, 1, 1)
}

func genPort() uint16 {
	return 8080
}

// MockPacketCaptureProvider is a minimal mock for testing
type MockPacketCaptureProvider struct {
	packets []*platform.Packet
	index   int
}

func (m *MockPacketCaptureProvider) Open(interfaceName string, promiscuous bool, filter string) error {
	return nil
}

func (m *MockPacketCaptureProvider) ReadPacket() (*platform.Packet, error) {
	if m.index >= len(m.packets) {
		return nil, nil
	}
	pkt := m.packets[m.index]
	m.index++
	return pkt, nil
}

func (m *MockPacketCaptureProvider) Close() error {
	return nil
}

func (m *MockPacketCaptureProvider) GetStats() (*platform.CaptureStats, error) {
	return &platform.CaptureStats{
		PacketsCaptured: uint64(m.index),
		PacketsDropped:  0,
		PacketsFiltered: 0,
	}, nil
}
