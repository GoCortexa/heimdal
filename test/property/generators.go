// +build property

package property

import (
	"net"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
)

// genMAC generates a valid MAC address
func genMAC() gopter.Gen {
	return gen.SliceOfN(6, gen.UInt8()).Map(func(bytes []uint8) net.HardwareAddr {
		mac := make(net.HardwareAddr, 6)
		for i, b := range bytes {
			mac[i] = b
		}
		return mac
	})
}

// genIP generates a valid IPv4 address
func genIP() gopter.Gen {
	return gen.SliceOfN(4, gen.UInt8()).Map(func(bytes []uint8) net.IP {
		return net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])
	})
}

// genProtocol generates a valid protocol name
func genProtocol() gopter.Gen {
	protocols := []string{"TCP", "UDP", "ICMP", "ARP", "DNS", "HTTP", "HTTPS"}
	return gen.OneConstOf(
		protocols[0], protocols[1], protocols[2], protocols[3],
		protocols[4], protocols[5], protocols[6],
	)
}
