// +build property

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
	"github.com/mosiko1234/heimdal/sensor/test/mocks"
)

// Feature: monorepo-architecture, Property 1: Packet Parser Interface Compatibility
// Validates: Requirements 2.4
func TestProperty_PacketParserInterfaceCompatibility(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Packet parser successfully processes packets from any PacketCaptureProvider implementation",
		prop.ForAll(
			func(packets []*platform.Packet) bool {
				// Create a mock provider with the generated packets
				mockProvider := mocks.NewMockPacketCaptureProvider(packets)

				// Create output channel
				outputChan := make(chan packet.PacketInfo, len(packets))

				// Create analyzer with the mock provider
				analyzer, err := packet.NewAnalyzer(mockProvider, outputChan, nil)
				if err != nil {
					t.Logf("Failed to create analyzer: %v", err)
					return false
				}

				// Open the provider
				err = mockProvider.Open("test0", true, "")
				if err != nil {
					t.Logf("Failed to open provider: %v", err)
					return false
				}
				defer mockProvider.Close()

				// Process packets directly instead of using Start
				// This avoids race conditions with goroutines in property tests
				processedCount := 0
				for i := 0; i < len(packets); i++ {
					pkt, err := mockProvider.ReadPacket()
					if err != nil {
						t.Logf("Failed to read packet %d: %v", i, err)
						return false
					}

					if pkt == nil {
						t.Logf("Got nil packet at index %d", i)
						return false
					}

					info, err := analyzer.ProcessPacket(pkt)
					if err != nil {
						t.Logf("Failed to process packet %d: %v", i, err)
						return false
					}

					if info == nil {
						t.Log("ProcessPacket returned nil info")
						return false
					}

					processedCount++
				}

				// Property: All packets should be successfully processed
				return processedCount == len(packets)
			},
			genPacketSlice(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genPacketSlice generates a slice of valid packets
func genPacketSlice() gopter.Gen {
	return gen.SliceOfN(10, genPacket())
}

// genPacket generates a valid packet
func genPacket() gopter.Gen {
	return gopter.CombineGens(
		genMAC(),
		genMAC(),
		genIP(),
		genIP(),
		gen.UInt16(),
		gen.UInt16(),
		genProtocol(),
		gen.UInt32Range(0, 65535),
	).Map(func(values []interface{}) *platform.Packet {
		return &platform.Packet{
			Timestamp:   time.Now(),
			SrcMAC:      values[0].(net.HardwareAddr),
			DstMAC:      values[1].(net.HardwareAddr),
			SrcIP:       values[2].(net.IP),
			DstIP:       values[3].(net.IP),
			SrcPort:     values[4].(uint16),
			DstPort:     values[5].(uint16),
			Protocol:    values[6].(string),
			PayloadSize: values[7].(uint32),
			RawData:     []byte{},
		}
	})
}


