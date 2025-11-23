package orchestrator

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mosiko1234/heimdal/sensor/internal/discovery"
	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
)

type staticNetConfigProvider struct {
	config *netconfig.NetworkConfig
}

func (s *staticNetConfigProvider) GetConfig() *netconfig.NetworkConfig {
	return s.config
}

func (o *DesktopOrchestrator) buildNetworkConfigProvider() (discovery.NetworkConfigProvider, error) {
	if o.config == nil {
		return nil, fmt.Errorf("desktop configuration is not initialized")
	}

	ifaceName, err := o.resolveInterfaceName()
	if err != nil {
		return nil, err
	}

	localIP, subnet, cidr, err := interfaceIPv4Info(ifaceName)
	if err != nil {
		return nil, err
	}

	gateway := detectGatewayAddress(ifaceName)

	cfg := &netconfig.NetworkConfig{
		Interface: ifaceName,
		LocalIP:   localIP,
		Gateway:   gateway,
		Subnet:    subnet,
		CIDR:      cidr,
	}

	// Persist interface choice for downstream components
	o.config.Network.Interface = ifaceName
	o.config.Network.AutoDetect = false

	return &staticNetConfigProvider{config: cfg}, nil
}

func (o *DesktopOrchestrator) resolveInterfaceName() (string, error) {
	if !o.config.Network.AutoDetect && o.config.Network.Interface != "" {
		return o.config.Network.Interface, nil
	}

	iface, err := detectDefaultInterface()
	if err != nil {
		return "", err
	}

	return iface, nil
}

func detectDefaultInterface() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to probe default route: %w", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	targetIP := localAddr.IP

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to list interfaces: %w", err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				if ipNet.IP.Equal(targetIP) {
					return iface.Name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not determine default network interface")
}

func interfaceIPv4Info(name string) (net.IP, *net.IPNet, string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open interface %s: %w", name, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get addresses for %s: %w", name, err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.To4() == nil {
			continue
		}

		ones, _ := ipNet.Mask.Size()
		network := ipNet.IP.Mask(ipNet.Mask)
		subnet := &net.IPNet{
			IP:   network,
			Mask: ipNet.Mask,
		}
		cidr := fmt.Sprintf("%s/%d", network.String(), ones)

		return ipNet.IP, subnet, cidr, nil
	}

	return nil, nil, "", fmt.Errorf("no IPv4 address found on interface %s", name)
}

func detectGatewayAddress(iface string) net.IP {
	switch runtime.GOOS {
	case "darwin":
		return parseGatewayFromCommand(exec.Command("route", "-n", "get", "default"))
	case "linux":
		return parseGatewayFromProc(iface)
	default:
		return nil
	}
}

func parseGatewayFromCommand(cmd *exec.Cmd) net.IP {
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "gateway:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if ip := net.ParseIP(fields[1]); ip != nil {
					return ip
				}
			}
		}
	}

	return nil
}

func parseGatewayFromProc(iface string) net.IP {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		if fields[0] == iface && fields[1] == "00000000" {
			var gatewayHex uint32
			if _, err := fmt.Sscanf(fields[2], "%X", &gatewayHex); err != nil {
				return nil
			}
			ip := make(net.IP, 4)
			ip[0] = byte(gatewayHex)
			ip[1] = byte(gatewayHex >> 8)
			ip[2] = byte(gatewayHex >> 16)
			ip[3] = byte(gatewayHex >> 24)
			return ip
		}
	}

	return nil
}
