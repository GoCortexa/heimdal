package discovery

import (
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
)

// scanARP performs ARP scanning across the subnet and returns the number of devices discovered.
func (s *Scanner) scanARP() (int, error) {
	netConfig := s.netConfig.GetConfig()
	if netConfig == nil {
		log.Println("Network configuration not available, skipping ARP scan")
		return 0, fmt.Errorf("network configuration not available")
	}

	log.Printf("Starting ARP scan on %s (%s)", netConfig.Interface, netConfig.CIDR)

	// Open pcap handle for sending and receiving ARP packets
	handle, err := pcap.OpenLive(netConfig.Interface, 65536, true, pcap.BlockForever)
	if err != nil {
		if isPermissionError(err) {
			s.reportStatus(StatusLevelError, "ARP scan requires elevated privileges: %v. %s", err, permissionGuidance())
		}
		return 0, fmt.Errorf("error opening pcap handle: %w", err)
	}
	defer handle.Close()

	// Set BPF filter to capture only ARP replies
	if err := handle.SetBPFFilter("arp"); err != nil {
		return 0, fmt.Errorf("error setting BPF filter: %w", err)
	}

	// Get local MAC address
	iface, err := net.InterfaceByName(netConfig.Interface)
	if err != nil {
		return 0, fmt.Errorf("error getting interface %s: %w", netConfig.Interface, err)
	}
	srcMAC := iface.HardwareAddr

	// Start goroutine to listen for ARP replies
	replyChan := make(chan arpReply, 100)
	done := make(chan struct{})

	go s.listenARPReplies(handle, replyChan, done)

	// Send ARP requests to all IPs in subnet
	if err := s.sendARPRequests(handle, netConfig, srcMAC); err != nil {
		close(done)
		return 0, fmt.Errorf("error sending ARP requests: %w", err)
	}

	// Wait for replies (with timeout)
	timeout := time.After(s.options.ARPReplyTimeout)
	replyCount := 0

collectReplies:
	for {
		select {
		case reply := <-replyChan:
			s.updateDevice(reply.MAC, reply.IP, "", "")
			replyCount++
		case <-timeout:
			break collectReplies
		case <-s.ctx.Done():
			break collectReplies
		}
	}

	close(done)
	log.Printf("ARP scan completed: discovered %d devices", replyCount)
	return replyCount, nil
}

// arpReply represents an ARP reply
type arpReply struct {
	MAC string
	IP  string
}

// listenARPReplies listens for ARP reply packets
func (s *Scanner) listenARPReplies(handle *pcap.Handle, replyChan chan<- arpReply, done chan struct{}) {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		select {
		case <-done:
			return
		case <-s.ctx.Done():
			return
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}

			// Parse ARP layer
			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}

			arp, ok := arpLayer.(*layers.ARP)
			if !ok {
				continue
			}

			// We only care about ARP replies
			if arp.Operation != layers.ARPReply {
				continue
			}

			// Extract source MAC and IP from the reply
			mac := net.HardwareAddr(arp.SourceHwAddress).String()
			ip := net.IP(arp.SourceProtAddress).String()

			select {
			case replyChan <- arpReply{MAC: mac, IP: ip}:
			default:
				// Channel full, drop reply
			}
		}
	}
}

// sendARPRequests sends ARP requests to all IPs in the subnet
func (s *Scanner) sendARPRequests(handle *pcap.Handle, netCfg *netconfig.NetworkConfig, srcMAC net.HardwareAddr) error {
	// Generate all IPs in the subnet
	ips := s.generateSubnetIPs(netCfg.Subnet)

	log.Printf("Sending ARP requests to %d IPs in subnet", len(ips))

	for _, ip := range ips {
		// Skip our own IP
		if ip.Equal(netCfg.LocalIP) {
			continue
		}

		// Build and send ARP request
		if err := s.sendARPRequest(handle, srcMAC, netCfg.LocalIP, ip); err != nil {
			log.Printf("Error sending ARP request to %s: %v", ip, err)
			return err
		}

		// Small delay to avoid flooding the network
		time.Sleep(1 * time.Millisecond)

		// Check if we should stop
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}
	}

	return nil
}

// sendARPRequest sends a single ARP request packet
func (s *Scanner) sendARPRequest(handle *pcap.Handle, srcMAC net.HardwareAddr, srcIP, dstIP net.IP) error {
	// Build Ethernet layer
	eth := layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Broadcast
		EthernetType: layers.EthernetTypeARP,
	}

	// Build ARP layer
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   srcMAC,
		SourceProtAddress: srcIP.To4(),
		DstHwAddress:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    dstIP.To4(),
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buf, opts, &eth, &arp); err != nil {
		return fmt.Errorf("failed to serialize ARP packet: %w", err)
	}

	// Send packet
	if err := handle.WritePacketData(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to send ARP packet: %w", err)
	}

	return nil
}

// generateSubnetIPs generates all IP addresses in a subnet
func (s *Scanner) generateSubnetIPs(subnet *net.IPNet) []net.IP {
	var ips []net.IP

	// Get the network address and mask
	ip := subnet.IP.To4()
	if ip == nil {
		return ips
	}

	mask := subnet.Mask

	// Calculate network and broadcast addresses
	network := ip.Mask(mask)
	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = network[i] | ^mask[i]
	}

	// Generate all IPs between network and broadcast
	for ip := incrementIP(network); !ip.Equal(broadcast); ip = incrementIP(ip) {
		ips = append(ips, copyIP(ip))
	}

	return ips
}

// incrementIP increments an IP address by 1
func incrementIP(ip net.IP) net.IP {
	// Make a copy
	result := make(net.IP, len(ip))
	copy(result, ip)

	// Increment from the last byte
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] != 0 {
			break
		}
	}

	return result
}

// copyIP creates a copy of an IP address
func copyIP(ip net.IP) net.IP {
	result := make(net.IP, len(ip))
	copy(result, ip)
	return result
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "permission") ||
		strings.Contains(msg, "operation not permitted") ||
		strings.Contains(msg, "access is denied")
}

func permissionGuidance() string {
	switch runtime.GOOS {
	case "darwin":
		return "Grant Full Disk Access and run the agent with sudo on first launch to allow packet capture."
	case "linux":
		return "Run as root or grant CAP_NET_RAW and CAP_NET_ADMIN (e.g., sudo setcap cap_net_raw,cap_net_admin=eip <binary>)."
	case "windows":
		return "Run as Administrator and install Npcap with WinPcap compatibility mode."
	default:
		return "Ensure the agent has permissions to open raw sockets on this platform."
	}
}
